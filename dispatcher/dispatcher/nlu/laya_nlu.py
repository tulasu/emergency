# -*- coding: utf-8 -*-
"""
Понимание целиком на laya, без банка формулировок.

Проверка идеи: что если реплику классифицировать сразу в слоты сценария, а
банк не нужен вовсе? Тогда новая ситуация не требует ни одной формулировки —
достаточно фактов и ответов заявителя.

Описанием варианта служит сам ответ: «этаж возгорания: На тринадцатом этаже,
балкон весь в огне». Это то, что заявитель может сказать, и ничего сверх
данных сценария для этого не нужно.

Отличие от `laya_arbiter`: там модель выбирает из пяти кандидатов, которых
отобрал банк, здесь — из всех слотов сценария сразу. Вариантов больше
(в корпусе от 3 до 17 на сценарий), и это главный риск затеи.
"""

from __future__ import annotations

import time
from dataclasses import dataclass, field
from typing import Any

from ..data.ontology import Ontology
from ..types import Act, Scenario, Understanding
from . import normalize, rules
from .laya import FIELD, NOTHING, QUESTION, make_router, refused


@dataclass
class LayaUnderstander:
    """Реплика -> слот сценария за один проход модели."""

    scenario: Scenario
    ontology: Ontology
    device: str = "cpu"
    min_confidence: float = 0.35
    router: Any = None

    calls: int = 0
    refusals: int = 0
    failures: int = 0
    _query: dict = field(default_factory=dict, repr=False)
    _by_label: dict[str, str] = field(default_factory=dict, repr=False)

    def __post_init__(self) -> None:
        if self.router is None:
            self.router = make_router(self.device)

        criteria: dict[str, str] = {}
        for slot, keys in sorted(self.scenario.by_slot.items()):
            sl = self.ontology.slots.get(slot)
            label = sl.label if sl else slot
            while label in self._by_label:
                label += " "
            answer = self.scenario.facts[keys[0]].answers["plain"]
            criteria[label] = f"{label}: {answer}"
            self._by_label[label] = slot
        criteria[NOTHING] = "вопрос не про перечисленное"

        # вопрос один и тот же на весь звонок — собирается один раз
        self._query = {
            "слот": {
                "type": "choice",
                "instructions": QUESTION,
                "criteria": criteria,
            }
        }

    def understand(self, text: str) -> Understanding:
        t0 = time.perf_counter()
        words = normalize.words(text)
        topical = normalize.topical_lemmas(text)
        act = rules.detect_act(text)

        # речевой акт разбирается правилами: модель тут не нужна и только
        # путается — «оставайтесь на линии» не вопрос ни о чём
        if act is Act.SPEECH_ACT:
            return Understanding(
                act=act,
                score=1.0,
                words=len(words),
                topical=len(topical),
                source="rules",
                latency_ms=round(1000 * (time.perf_counter() - t0), 3),
            )

        try:
            res = self.router.predict({FIELD: text}, self._query)
            self.calls += 1
        except Exception:  # noqa: BLE001
            self.failures += 1
            return Understanding(
                act=act, words=len(words), topical=len(topical), source="laya-error"
            )

        answer = res["answers"]["слот"]
        pick = answer.get("choice")
        confidence = float(answer.get("confidence", 0.0))
        latency = round(1000 * (time.perf_counter() - t0), 3)

        if refused(pick, confidence, self.min_confidence):
            self.refusals += 1
            return Understanding(
                act=act,
                score=confidence,
                words=len(words),
                topical=len(topical),
                source="laya",
                latency_ms=latency,
            )

        slot = self._by_label.get(pick)
        if slot is None:
            return Understanding(
                act=act,
                score=confidence,
                words=len(words),
                topical=len(topical),
                source="laya",
                latency_ms=latency,
            )

        keys = list(self.scenario.by_slot.get(slot, ()))
        said = normalize.numbers(text)
        mismatch = any(
            said
            and self.scenario.facts[k].numbers
            and not (said & self.scenario.facts[k].numbers)
            for k in keys
        )
        return Understanding(
            slots=[slot],
            keys=keys,
            act=act,
            score=confidence,
            words=len(words),
            topical=len(topical),
            value_mismatch=mismatch,
            source="laya",
            latency_ms=latency,
        )

    def preview(self, text: str) -> None:
        """Спекуляции нет: проход модели стоит 14 мс — прятать нечего,
        а двойной вызов на каждую гипотезу лишь жжёт GPU."""
        return None

    def save(self) -> None:
        """Совместимость с каскадом: кэшировать нечего."""
