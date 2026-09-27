# -*- coding: utf-8 -*-
"""Реранкер top-5 для Ensemble: оценки пар (реплика, описание слота)."""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Protocol

from ..data.ontology import Ontology


class Scorer(Protocol):
    def score_many(self, text: str, slots: list[str]) -> list[float]: ...


@dataclass
class CrossEncoderScorer:
    ontology: Ontology
    path: str | Path
    device: str = "cpu"
    model: object = field(init=False, repr=False)
    tokenizer: object = field(init=False, repr=False)

    def __post_init__(self) -> None:
        from transformers import AutoModelForSequenceClassification, AutoTokenizer

        self.tokenizer = AutoTokenizer.from_pretrained(self.path, local_files_only=True)
        self.model = AutoModelForSequenceClassification.from_pretrained(
            self.path, local_files_only=True
        ).to(self.device).eval()

    def score_many(self, text: str, slots: list[str]) -> list[float]:
        import torch

        if not slots:
            return []
        pairs = self.tokenizer(
            ["query: " + text] * len(slots),
            ["passage: " + self.ontology.slots[s].label for s in slots],
            padding=True, truncation=True, max_length=96, return_tensors="pt",
        ).to(self.device)
        with torch.inference_mode():
            logits = self.model(**pairs).logits.flatten()
            return torch.sigmoid(logits).tolist()


@dataclass
class RerankerVoter:
    scorer: Scorer
    min_score: float = 0.3
    min_margin: float = 0.02
    calls: int = 0
    refusals: int = 0
    failures: int = 0
    last: str = ""

    def choose(self, text: str, slots: list[str]) -> str | None:
        if not slots:
            return None
        try:
            scores = self.scorer.score_many(text, slots)
            if len(scores) != len(slots):
                raise ValueError("scorer вернул неверное число оценок")
        except Exception as e:  # noqa: BLE001 — не обрываем звонок при сбое модели
            self.failures += 1
            self.last = f"ошибка {type(e).__name__}: {e}"[:200]
            return None
        self.calls += 1
        order = sorted(range(len(slots)), key=scores.__getitem__, reverse=True)
        best = scores[order[0]]
        runner = scores[order[1]] if len(order) > 1 else 0.0
        slot = slots[order[0]]
        self.last = f"{slot} {best:.2f} (разрыв {best - runner:.2f})"
        if best < self.min_score or best - runner < self.min_margin:
            self.refusals += 1
            return None
        return slot
