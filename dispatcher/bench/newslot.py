# -*- coding: utf-8 -*-
"""
Слот, которого дообученная модель никогда не видела.

Свёртки по сценариям проверяют новую ситуацию из готовых слотов. Здесь
проверяется тяжёлый случай: автор завёл слот, какого в онтологии не было,
и написал к нему несколько затравочных формулировок.

Слот целиком вычёркивается из обучения. В банке от него остаются только
затравки — как у настоящего нового слота. Проверяемся на остальных его
формулировках: дотянется ли до них модель, которая этого слота не знает.

Сравниваются стоковый энкодер и дообученный: дообучение могло «затянуть»
пространство под известные слоты и испортить перенос на незнакомый.

  python3 -m bench.newslot --epochs 20 --seeds 5
"""

from __future__ import annotations

import argparse
from collections import Counter

import numpy as np
import torch
import torch.nn.functional as F
from transformers import AutoModel, AutoTokenizer

from dispatcher.data.loader import load_all
from dispatcher.data.ontology import Ontology
from bench.finetune import BASE_PATH, encode_batch


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--epochs", type=int, default=20)
    ap.add_argument("--batch", type=int, default=128)
    ap.add_argument("--lr", type=float, default=2e-5)
    ap.add_argument("--temp", type=float, default=0.05)
    ap.add_argument(
        "--seeds",
        type=int,
        default=5,
        help="сколько формулировок автор пишет к новому слоту",
    )
    ap.add_argument("--holdout", type=int, default=8, help="сколько слотов прячем")
    args = ap.parse_args()

    onto = Ontology.load()
    loaded = load_all(onto)
    rows = []
    for sid in sorted(loaded.scenarios):
        for fact in loaded.scenarios[sid].facts.values():
            for q in fact.questions:
                rows.append((q, fact.slot, sid))

                # прячем слоты средней населённости: у редких мало данных на проверку,
                # у самых частых вычёркивание перекроит весь корпус
    count = Counter(s for _, s, _ in rows)
    usable = [s for s, n in count.items() if 30 <= n <= 90]
    hidden = set(sorted(usable)[: args.holdout])
    print(f"прячем {len (hidden )} слотов: {', '.join (sorted (hidden ))}")
    print(f"формулировок у них: {sum (count [s ]for s in hidden )}")

    tok = AutoTokenizer.from_pretrained(BASE_PATH)
    rng = np.random.default_rng(0)

    # затравки автора и проверочная часть
    seed_texts, probe = [], []
    for slot in hidden:
        own = [(q, s, sid) for q, s, sid in rows if s == slot]
        order = rng.permutation(len(own))
        for i in order[: args.seeds]:
            seed_texts.append(own[i])
        for i in order[args.seeds :]:
            probe.append(own[i])

    bank = [(q, s) for q, s, _ in rows if s not in hidden] + [
        (q, s) for q, s, _ in seed_texts
    ]
    print(f"банк {len (bank )} формулировок, проверка {len (probe )}")

    def evaluate(model, tag):
        model.eval()
        with torch.no_grad():
            B = encode_batch(model, tok, [q for q, _ in bank], "passage: ")
            Q = encode_batch(model, tok, [q for q, _, _ in probe], "query: ")
        bank_slots = np.array([s for _, s in bank])
        hits = 0
        for i in range(0, len(probe), 512):
            with torch.no_grad():
                sims = Q[i : i + 512] @ B.T
            for k, (_, gold, sid) in enumerate(probe[i : i + 512]):
                sc = loaded.scenarios[sid]
                row = sims[k].clone()
                row[~np.isin(bank_slots, list(sc.slots))] = -1e4
                picked = bank_slots[row.argmax().item()]
                # скидка на замещение, как во всех остальных замерах
                ok = {gold} | {m for m, v in sc.answers_for.items() if v == gold}
                hits += picked in ok
        print(f"  {tag :<34} {100 *hits /len (probe ):5.1f}%")
        return 100 * hits / len(probe)

    print("\nновый слот, которого модель не видела:")
    model = AutoModel.from_pretrained(BASE_PATH).cuda()
    before = evaluate(model, "стоковый энкодер")

    train_rows = [(q, s) for q, s, _ in rows if s not in hidden]
    slots = sorted({s for _, s in train_rows})
    index = {s: i for i, s in enumerate(slots)}
    opt = torch.optim.AdamW(model.parameters(), lr=args.lr)
    model.train()
    for _ in range(args.epochs):
        order = rng.permutation(len(train_rows))
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
                own = (y[:, None] == y[None, :]).float()
                own.fill_diagonal_(0)
                sims.fill_diagonal_(-1e4)
                has_pos = own.sum(1) > 0
                loss = (
                    -(F.log_softmax(sims, 1) * own).sum(1)[has_pos]
                    / own.sum(1)[has_pos]
                )
                loss = loss.mean()
            loss.backward()
            opt.step()
            opt.zero_grad(set_to_none=True)

    after = evaluate(model, "дообученный без этих слотов")
    print(f"\nперенос на незнакомый слот: {after -before :+.1f} пункта")


if __name__ == "__main__":
    main()
