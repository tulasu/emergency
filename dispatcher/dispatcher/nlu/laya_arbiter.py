# -*- coding: utf-8 -*-
"""
Арбитр серой зоны на laya.

Laya решает типизированный вопрос за один проход: ни генерации, ни парсинга,
ни срывов формата — то, на чём развалился Qwen. И, в отличие от него, она
умеет отвечать «ни о чём», а это здесь важнее точности: ложный факт хуже
честного «не знаю».

Замер на двенадцати размеченных случаях с перемешанными позициями:

    Qwen3-1.7B без рассуждений    2-3 из 10     66-135 мс
    Qwen3-1.7B с рассуждениями    5-6 из 10   3500-12150 мс
    laya-multilingual             8 из 12          144 мс (CPU)

Вопросы оператора кириллицей, поэтому берётся мультиязычный чекпоинт:
английский на нечитаемом для него письме даёт ноль при высокой уверенности,
и отсечь это по уверенности нельзя.

Ответ заявителя всё равно берётся из сценария. Модель решает, о чём
спросили, но не что сказать.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from ..data.ontology import Ontology
from .laya import FIELD, NOTHING, QUESTION, make_router, refused


@dataclass
class LayaArbiter:
    """Выбор слота из пяти кандидатов одним проходом."""

    ontology: Ontology
    device: str = "cpu"
    min_confidence: float = 0.5
    router: Any = None
    calls: int = 0
    refusals: int = 0
    failures: int = 0
    last: str = ""  # что было на последнем вызове — для журнала звонка

    def __post_init__(self) -> None:
        if self.router is None:
            self.router = make_router(self.device)
            self._warmup()

    def _warmup(self) -> None:
        """Поднять веса заранее.

        Router грузит чекпоинт лениво, при первом вопросе. В разговоре это
        означает восемьдесят секунд тишины посреди первого хода — заявитель
        будто уснул. Лучше подождать на старте.
        """
        try:
            self.router.predict(
                {FIELD: "какой адрес"},
                {
                    "слот": {
                        "type": "choice",
                        "instructions": "прогрев",
                        "criteria": {"а": "раз", "б": "два"},
                    }
                },
            )
        except Exception as e:  # noqa: BLE001
            # не роняем старт, но и не молчим: без весов laya всегда «ни о чём»
            print(f"laya: прогрев не удался — {type(e).__name__}: {e}"[:300], flush=True)

    def _criterion(self, slot: str) -> str:
        """Чем слот отличается от соседей — это и есть критерий выбора."""
        sl = self.ontology.slots.get(slot)
        if not sl:
            return slot
        examples = ", ".join(sl.questions[:2])
        return f"{sl.label}{'; ' + examples if examples else ''}"

    def choose(self, text: str, slots: list[str]) -> str | None:
        if not slots:
            return None

        criteria: dict[str, str] = {}
        by_label: dict[str, str] = {}
        for slot in slots:
            label = (
                self.ontology.slots[slot].label if slot in self.ontology.slots else slot
            )
            while label in by_label:  # две метки не должны совпасть
                label += " "
            criteria[label] = self._criterion(slot)
            by_label[label] = slot
        criteria[NOTHING] = "вопрос не про перечисленное"

        question = {
            "слот": {
                "type": "choice",
                "instructions": QUESTION,
                "criteria": criteria,
            }
        }
        try:
            res = self.router.predict({FIELD: text}, question)
        except Exception as e:  # noqa: BLE001
            # модель не поднялась или не успела: каскад ответит «не знаю»,
            # звонок из-за арбитра не встаёт
            self.failures += 1
            self.last = f"ошибка {type(e).__name__}: {e}"[:200]
            return None

        self.calls += 1
        answer = res["answers"]["слот"]
        pick = answer.get("choice")
        confidence = float(answer.get("confidence", 0.0))

        self.last = f"{pick} {confidence:.2f}"
        if refused(pick, confidence, self.min_confidence):
            self.refusals += 1
            return None
        return by_label.get(pick)
