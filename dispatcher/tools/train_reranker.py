# -*- coding: utf-8 -*-
"""Обучить бинарный cross-encoder на LOO-кандидатах сценариев.

Вход: bench.rerank_data (без blind/live); целевой слот положителен,
остальные top-5 отрицательны. Отрицательные строки pos5miss обучают отказу.
Выход — локальная модель для dispatcher.nlu.reranker.CrossEncoderScorer.

  python3 -m bench.rerank_data
  python3 -m tools.train_reranker --source models/e5-small-torch
"""

from __future__ import annotations

import argparse
import json
import time
from pathlib import Path

import numpy as np
import torch
import torch.nn.functional as F
from transformers import AutoModelForSequenceClassification, AutoTokenizer

from dispatcher.data.loader import DATA
from dispatcher.data.ontology import Ontology


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--data", default=str(DATA / "build" / "rerank.jsonl"))
    ap.add_argument("--source", default="models/e5-small-torch")
    ap.add_argument("--out", default=str(DATA / "build" / "reranker"))
    ap.add_argument("--device", default="cuda")
    ap.add_argument("--epochs", type=int, default=2)
    ap.add_argument("--batch", type=int, default=32)
    ap.add_argument("--lr", type=float, default=2e-5)
    args = ap.parse_args()

    labels = {slot: item.label for slot, item in Ontology.load().slots.items()}
    texts, descriptions, targets = [], [], []
    with open(args.data, encoding="utf-8") as f:
        for line in f:
            row = json.loads(line)
            for i, slot in enumerate(row["cands"]):
                texts.append("query: " + row["text"])
                descriptions.append("passage: " + labels[slot])
                targets.append(float(row["label"] == i) if row["label"] is not None else 0.0)
    if not texts or not any(targets):
        raise ValueError("нет размеченных пар для обучения")
    print(f"пары {len(texts)}, положительных {int(sum(targets))}", flush=True)

    tok = AutoTokenizer.from_pretrained(args.source, local_files_only=True)
    model = AutoModelForSequenceClassification.from_pretrained(
        args.source, num_labels=1, ignore_mismatched_sizes=True, local_files_only=True
    ).to(args.device)
    optimizer = torch.optim.AdamW(model.parameters(), lr=args.lr)
    weight = torch.tensor([(len(targets) - sum(targets)) / sum(targets)], device=args.device)
    rng = np.random.default_rng(0)
    started = time.perf_counter()
    model.train()
    for epoch in range(args.epochs):
        order = rng.permutation(len(targets))
        total_loss = 0.0
        for start in range(0, len(order), args.batch):
            ix = order[start:start + args.batch]
            batch = tok([texts[i] for i in ix], [descriptions[i] for i in ix],
                        padding=True, truncation=True, max_length=96,
                        return_tensors="pt").to(args.device)
            y = torch.tensor([targets[i] for i in ix], device=args.device)
            with torch.autocast(args.device, dtype=torch.bfloat16, enabled=args.device == "cuda"):
                logits = model(**batch).logits.flatten()
                loss = F.binary_cross_entropy_with_logits(logits.float(), y, pos_weight=weight)
            loss.backward()
            optimizer.step()
            optimizer.zero_grad(set_to_none=True)
            total_loss += loss.item() * len(ix)
        print(f"эпоха {epoch + 1}: loss {total_loss / len(targets):.4f}", flush=True)

    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    model.eval().save_pretrained(out)
    tok.save_pretrained(out)
    (out / "training.json").write_text(json.dumps({
        "source": args.source, "data": args.data, "epochs": args.epochs,
        "batch": args.batch, "lr": args.lr, "pairs": len(targets),
        "positives": int(sum(targets)), "seconds": round(time.perf_counter() - started),
    }, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"модель сохранена: {out}")


if __name__ == "__main__":
    main()
