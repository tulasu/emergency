# -*- coding: utf-8 -*-
"""
Дообучение энкодера на формулировках корпуса.

Цель контрастная, с отрицательными примерами внутри пачки: формулировки
одного слота сходятся, разных — расходятся. Улучшается само представление
текста, а значит и поиск по банку, который на нём стоит.

Проверка в свёртки по сценариям: обучаемся на одних сценариях, меряемся на
других, и банк для отложенных собирается тоже без них. Протокол тот же, что
у всех прежних цифр, поэтому сравнение прямое.

  python3 -m bench.finetune --folds 4 --epochs 20
"""

from __future__ import annotations

import argparse
import statistics
import time

import numpy as np
import torch
import torch.nn.functional as F
from transformers import AutoModel, AutoTokenizer

from dispatcher.data.loader import load_all
from dispatcher.data.ontology import Ontology

BASE_PATH = "models/e5-small-torch"


def collect():
    onto = Ontology.load()
    loaded = load_all(onto)
    rows = []
    for sid in sorted(loaded.scenarios):
        for fact in loaded.scenarios[sid].facts.values():
            for q in fact.questions:
                rows.append((q, fact.slot, sid))
    return loaded, rows


def encode_batch(model, tok, texts, prefix, batch=128, seqlen=48, train=False):
    vectors = []
    for i in range(0, len(texts), batch):
        enc = tok(
            [prefix + t for t in texts[i : i + batch]],
            padding=True,
            truncation=True,
            max_length=seqlen,
            return_tensors="pt",
        ).to("cuda")
        with torch.autocast("cuda", dtype=torch.bfloat16):
            h = model(**enc).last_hidden_state
            m = enc["attention_mask"].unsqueeze(-1)
            v = (h * m).sum(1) / m.sum(1).clamp(min=1e-9)
        vectors.append(F.normalize(v.float(), dim=-1))
    return torch.cat(vectors)


def accuracy(model, tok, rows, heldout, loaded, batch=128):
    """Поиск ближайшей формулировки: банк без отложенных сценариев.

    Кандидаты ограничены слотами сценария — так же работает рантайм, и так
    цифра сравнима со всеми прежними замерами.
    """
    model.eval()
    bank = [(q, s) for q, s, sid in rows if sid not in heldout]
    test = [(q, s, sid) for q, s, sid in rows if sid in heldout]
    if not test:
        return 0.0, []

    with torch.no_grad():
        B = encode_batch(model, tok, [q for q, _ in bank], "passage: ", batch)
        Q = encode_batch(model, tok, [q for q, _, _ in test], "query: ", batch)
    bank_slots = np.array([s for _, s in bank])

    hits = 0
    per_scenario: dict[str, list[int]] = {}
    for i in range(0, len(test), 512):
        chunks = test[i : i + 512]
        with torch.no_grad():
            sims = Q[i : i + 512] @ B.T
        for k, (_, gold, sid) in enumerate(chunks):
            sc = loaded.scenarios[sid]
            own = sc.slots
            row = sims[k].clone()
            row[~np.isin(bank_slots, list(own))] = -1e4
            picked = bank_slots[row.argmax().item()]
            ok_set = {gold} | {m for m, v in sc.answers_for.items() if v == gold}
            ok = picked in ok_set
            hits += ok
            per_scenario.setdefault(sid, []).append(int(ok))
    shares = [100 * sum(v) / len(v) for v in per_scenario.values()]
    return 100 * hits / len(test), shares


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--folds", type=int, default=4)
    ap.add_argument("--epochs", type=int, default=20)
    ap.add_argument("--batch", type=int, default=128)
    ap.add_argument("--lr", type=float, default=2e-5)
    ap.add_argument("--temp", type=float, default=0.05)
    args = ap.parse_args()

    loaded, rows = collect()
    scenarios = sorted(loaded.scenarios)
    print(f"формулировок {len (rows )}, сценариев {len (scenarios )}")

    tok = AutoTokenizer.from_pretrained(BASE_PATH)
    folds = [set(scenarios[i :: args.folds]) for i in range(args.folds)]

    before_all, after_all = [], []
    for number, heldout in enumerate(folds, 1):
        model = AutoModel.from_pretrained(BASE_PATH).cuda()
        before, _ = accuracy(model, tok, rows, heldout, loaded)

        train_rows = [(q, s) for q, s, sid in rows if sid not in heldout]
        slots = sorted({s for _, s in train_rows})
        index = {s: i for i, s in enumerate(slots)}
        opt = torch.optim.AdamW(model.parameters(), lr=args.lr)

        t0 = time.perf_counter()
        model.train()
        for epoch in range(args.epochs):
            order = np.random.permutation(len(train_rows))
            for i in range(0, len(order), args.batch):
                batch = [train_rows[j] for j in order[i : i + args.batch]]
                if len(batch) < 8:
                    continue
                enc = tok(
                    ["query: " + q for q, _ in batch],
                    padding=True,
                    truncation=True,
                    max_length=48,
                    return_tensors="pt",
                ).to("cuda")
                y = torch.tensor([index[s] for _, s in batch], device="cuda")
                with torch.autocast("cuda", dtype=torch.bfloat16):
                    h = model(**enc).last_hidden_state
                    m = enc["attention_mask"].unsqueeze(-1)
                    v = F.normalize((h * m).sum(1) / m.sum(1).clamp(min=1e-9), dim=-1)
                    sims = v @ v.T / args.temp
                    # сосед по слоту — положительный пример, остальные пачки — нет
                    own = (y[:, None] == y[None, :]).float()
                    own.fill_diagonal_(0)
                    sims.fill_diagonal_(-1e4)
                    log_p = F.log_softmax(sims, dim=1)
                    has_pos = own.sum(1) > 0
                    loss = -(log_p * own).sum(1)[has_pos] / own.sum(1)[has_pos]
                    loss = loss.mean()
                loss.backward()
                opt.step()
                opt.zero_grad(set_to_none=True)

        train_secs = time.perf_counter() - t0
        after, shares = accuracy(model, tok, rows, heldout, loaded)
        before_all.append(before)
        after_all.append(after)
        print(
            f"свёртка {number }: было {before :5.1f}%  стало {after :5.1f}%  "
            f"({after -before :+5.1f}), обучение {train_secs :.0f} с, "
            f"медиана по сценарию {statistics .median (shares ):.1f}%"
        )
        del model, opt
        torch.cuda.empty_cache()

    print(
        f"\nитого: было {statistics .mean (before_all ):.1f}%  "
        f"стало {statistics .mean (after_all ):.1f}%  "
        f"({statistics .mean (after_all )-statistics .mean (before_all ):+.1f})"
    )


if __name__ == "__main__":
    main()
