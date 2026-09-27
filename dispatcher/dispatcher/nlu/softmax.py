# -*- coding: utf-8 -*-
"""Замена оценки населённых слотов банка: замороженный e5 + голова 88+1.

Бедные слоты остаются у лексического банка. Обучение: tools.train_softmax.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class SoftmaxScorer:
    path: str | Path
    device: str = "cpu"
    model: object = field(init=False, repr=False)
    tokenizer: object = field(init=False, repr=False)
    head: object = field(init=False, repr=False)
    slots: tuple[str, ...] = field(init=False)

    def __post_init__(self) -> None:
        import torch
        from safetensors.torch import load_file
        from transformers import AutoModel, AutoTokenizer

        path = Path(self.path)
        meta = json.loads((path / "training.json").read_text(encoding="utf-8"))
        self.slots = tuple(meta["slots"])
        source = meta["source"]
        self.tokenizer = AutoTokenizer.from_pretrained(source, local_files_only=True)
        self.model = AutoModel.from_pretrained(source, local_files_only=True).to(self.device).eval()
        self.head = torch.nn.Linear(self.model.config.hidden_size, len(self.slots) + 1).to(self.device)
        self.head.load_state_dict(load_file(str(path / "head.safetensors"), device=self.device))
        self.head.eval()

    def score(self, text: str, population: dict[str, int], thin_limit: int) -> dict[str, float]:
        import torch

        rich = [i for i, slot in enumerate(self.slots) if population.get(slot, 0) > thin_limit]
        if not rich:
            return {}
        batch = self.tokenizer("query: " + text, return_tensors="pt",
                               truncation=True, max_length=96).to(self.device)
        with torch.inference_mode():
            hidden = self.model(**batch).last_hidden_state
            mask = batch["attention_mask"].unsqueeze(-1)
            pooled = (hidden * mask).sum(dim=1) / mask.sum(dim=1)
            logits = self.head(pooled)[0]
            chosen = torch.tensor([*rich, len(self.slots)], device=self.device)
            probs = torch.softmax(logits[chosen], dim=0)[:-1].tolist()
        return {self.slots[i]: p for i, p in zip(rich, probs)}
