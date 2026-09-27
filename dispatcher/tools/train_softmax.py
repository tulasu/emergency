# -*- coding: utf-8 -*-
"""Обучить экспериментальную голову 88+1 только на вопросах сценариев.

Класс «ни один» задаётся парой (вопрос сценария A, набор слотов сценария B),
где правильного слота нет. Энкодер e5 заморожен; blind/live не читаются.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import torch
import torch.nn.functional as F
from safetensors.torch import save_file
from transformers import AutoModel, AutoTokenizer

from bench.holdout import heldout_texts, normalized
from dispatcher.data.loader import DATA, load_all
from dispatcher.data.ontology import Ontology
from dispatcher.nlu.bank import LexicalBank


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--source", required=True, help="локальный pretrained e5-small")
    ap.add_argument("--out", default=str(DATA / "build" / "softmax"))
    ap.add_argument("--device", default="cuda")
    ap.add_argument("--epochs", type=int, default=150)
    args = ap.parse_args()

    loaded = load_all()
    bank = LexicalBank.build(loaded.questions, loaded.sources)
    slots = sorted(Ontology.load().slots)
    slot_index = {s: i for i, s in enumerate(slots)}
    scenarios = sorted(loaded.scenarios.values(), key=lambda sc: sc.id)
    excluded = heldout_texts()
    skipped = 0
    texts, targets, masks, groups, holdouts = [], [], [], [], []
    none = len(slots)
    full = torch.ones(none + 1, dtype=torch.bool)
    scenario_masks = {
        sc.id: torch.tensor([s in sc.slots for s in slots] + [True], dtype=torch.bool)
        for sc in scenarios
    }
    for sc in scenarios:
        for fact in sc.facts.values():
            alternatives = [other for other in scenarios if fact.slot not in other.slots]
            for index, text in enumerate(fact.questions):
                if normalized(text) in excluded:
                    skipped += 1
                    continue
                texts.append(text)
                targets.append(slot_index[fact.slot])
                masks.append(full)
                groups.append(sc.id)
                holdouts.append(index % 10 == 0)
                # Fallback-слоты вроде addr.building доступны в каждом
                # сценарии: для них нельзя создать честный отказ.
                if alternatives:
                    other = alternatives[index % len(alternatives)]
                    texts.append(text)
                    targets.append(none)
                    masks.append(scenario_masks[other.id])
                    groups.append(sc.id)
                    holdouts.append(index % 10 == 0)
    # Ни одна строка blind/live не участвует ни в обучении, ни в валидации.
    tokenizer = AutoTokenizer.from_pretrained(args.source, local_files_only=True)
    encoder = AutoModel.from_pretrained(args.source, local_files_only=True).to(args.device).eval()
    vectors = []
    with torch.inference_mode():
        for start in range(0, len(texts), 128):
            batch = tokenizer(["query: " + t for t in texts[start:start + 128]],
                              padding=True, truncation=True, max_length=96,
                              return_tensors="pt").to(args.device)
            hidden = encoder(**batch).last_hidden_state
            mask = batch["attention_mask"].unsqueeze(-1)
            vectors.append(((hidden * mask).sum(dim=1) / mask.sum(dim=1)).cpu())
    features = torch.cat(vectors).to(args.device)
    del encoder
    labels = torch.tensor(targets, device=args.device)
    choices = torch.stack(masks).to(args.device)
    # Каждая десятая формулировка факта (и её отрицательный контекст).
    held = torch.tensor(holdouts, device=args.device)
    train = ~held
    torch.manual_seed(0)
    head = torch.nn.Linear(features.shape[1], none + 1, device=args.device)
    optimizer = torch.optim.AdamW(head.parameters(), lr=0.01, weight_decay=0.01)
    for epoch in range(args.epochs):
        head.train()
        logits = head(features[train]).masked_fill(~choices[train], -1e4)
        loss = F.cross_entropy(logits, labels[train])
        optimizer.zero_grad(set_to_none=True)
        loss.backward()
        optimizer.step()
        if (epoch + 1) % 30 == 0:
            print(f"epoch {epoch + 1}: loss {loss.item():.4f}", flush=True)
    head.eval()
    with torch.inference_mode():
        predicted = head(features[held]).masked_fill(~choices[held], -1e4).argmax(dim=1)
        gold = labels[held]
        thin = torch.tensor(
            [bank.population.get(slots[t], 0) <= 40 if t != none else False
             for t, keep in zip(targets, holdouts) if keep],
            device=args.device,
        )
        validation = {
            "present": float((predicted[gold != none] == gold[gold != none]).float().mean()),
            "absent": float((predicted[gold == none] == none).float().mean()),
            "present_thin": float((predicted[thin] == gold[thin]).float().mean()),
            "present_rich": float((predicted[(gold != none) & ~thin]
                                    == gold[(gold != none) & ~thin]).float().mean()),
            "held_thin": int(thin.sum()),
        }
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    save_file({k: v.detach().cpu().contiguous() for k, v in head.state_dict().items()},
              str(out / "head.safetensors"))
    (out / "training.json").write_text(json.dumps({
        "source": args.source, "slots": slots, "epochs": args.epochs,
        "examples": len(texts), "validation": validation,
        "excluded_holdout_phrases": skipped,
        "train_scenarios": sorted(set(groups)),
    }, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"head: {out}; validation: {validation}", flush=True)


if __name__ == "__main__":
    main()
