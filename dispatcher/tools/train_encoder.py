# -*- coding: utf-8 -*-
"""
Дообучение энкодера на всём корпусе и сохранение весов для рантайма.

`bench/finetune.py` мерит прирост по свёрткам и веса выбрасывает. Здесь
обучается одна модель на всех сценариях — та, что пойдёт в работу.

Цель контрастная, с отрицательными примерами внутри пачки: формулировки
одного слота сходятся, разных расходятся. По свёрткам это давало +4.1
пункта на сценариях, которых модель не видела.

  python3 -m tools.train_encoder                    20 эпох, в models/e5-small-tuned
  python3 -m tools.train_encoder --epochs 30 --out models/e5-v2
"""

from __future__ import annotations

import argparse
import json
import time
from pathlib import Path

import numpy as np
import torch
import torch.nn.functional as F
from transformers import AutoModel, AutoTokenizer

from dispatcher.data.loader import load_all
from dispatcher.data.ontology import Ontology

SOURCE_PATH = "models/e5-small-torch"


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--epochs", type=int, default=20)
    ap.add_argument("--batch", type=int, default=128)
    ap.add_argument("--lr", type=float, default=2e-5)
    ap.add_argument("--temp", type=float, default=0.05)
    ap.add_argument("--out", default="models/e5-small-tuned")
    ap.add_argument("--device", default="cuda")
    args = ap.parse_args()

    onto = Ontology.load()
    loaded = load_all(onto)
    pairs = [
        (q, fact.slot)
        for sc in loaded.scenarios.values()
        for fact in sc.facts.values()
        for q in fact.questions
    ]
    slots = sorted({s for _, s in pairs})
    index = {s: i for i, s in enumerate(slots)}
    print(f"формулировок {len (pairs )}, слотов {len (slots )}")

    tok = AutoTokenizer.from_pretrained(SOURCE_PATH)
    model = AutoModel.from_pretrained(SOURCE_PATH).to(args.device)
    opt = torch.optim.AdamW(model.parameters(), lr=args.lr)
    rng = np.random.default_rng(0)

    model.train()
    t0 = time.perf_counter()
    for epoch in range(args.epochs):
        order = rng.permutation(len(pairs))
        losses = []
        for i in range(0, len(order), args.batch):
            batch = [pairs[j] for j in order[i : i + args.batch]]
            if len(batch) < 8:
                continue
            enc = tok(
                ["query: " + q for q, _ in batch],
                padding=True,
                truncation=True,
                max_length=48,
                return_tensors="pt",
            ).to(args.device)
            y = torch.tensor([index[s] for _, s in batch], device=args.device)
            with torch.autocast(args.device, dtype=torch.bfloat16):
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
            losses.append(float(loss))
        if (epoch + 1) % 5 == 0 or epoch == 0:
            print(
                f"  эпоха {epoch +1 :>2}: потеря {np .mean (losses ):.4f}", flush=True
            )

    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    model.eval()
    model.save_pretrained(out)
    tok.save_pretrained(out)
    (out / "training.json").write_text(
        json.dumps(
            {
                "source": SOURCE_PATH,
                "epochs": args.epochs,
                "batch": args.batch,
                "lr": args.lr,
                "temp": args.temp,
                "phrasings": len(pairs),
                "slots": len(slots),
                "scenarios": len(loaded.scenarios),
                "seconds": round(time.perf_counter() - t0),
            },
            ensure_ascii=False,
            indent=1,
        ),
        encoding="utf-8",
    )
    print(f"\nсохранено в {out } за {time .perf_counter ()-t0 :.0f} с")
    print(
        "веса обучены на ВСЕХ сценариях — мерить ими по корпусу нельзя, "
        "цифра будет завышена"
    )


if __name__ == "__main__":
    raise SystemExit(main())
