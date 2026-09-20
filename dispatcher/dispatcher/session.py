# -*- coding: utf-8 -*-
"""
Сессия звонка: понимание -> политика -> речь.

Один и тот же код под двумя входами. В SIP приходят частичные гипотезы STT
и финал, в тестах — готовый текст. Текстовый режим обязан давать тот же
результат, что потоковый, иначе тесты врут.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from .data.ontology import Ontology
from .dialog.policy import decide
from .dialog.render import Renderer
from .dialog.state import CallState
from .types import Reply, Scenario, Turn, Understander, Understanding


@dataclass(slots=True)
class Session:
    scenario: Scenario
    cascade: Understander
    renderer: Renderer
    state: CallState = field(default_factory=CallState)
    turns: list[Turn] = field(default_factory=list)

    # спекуляция: что уже посчитано по последней частичной гипотезе
    _draft_text: str = ""
    _draft: Understanding | None = None

    @staticmethod
    def open(
        scenario: Scenario,
        cascade: Understander,
        onto: Ontology | None = None,
        seed: int | None = None,
    ) -> "Session":
        return Session(scenario, cascade, Renderer(scenario, onto, seed))

    def opening(self) -> str:
        return self.scenario.opening

    # --------------------------------------------------------- потоковый вход

    def on_partial(self, text: str) -> None:
        """Частичная гипотеза STT: считаем заранее, наружу ничего не отдаём.

        К моменту, когда VAD скажет «оператор договорил», ответ обычно уже
        выбран, и задержка понимания в звонке не слышна.
        """
        text = text.strip()
        if len(text) < 6 or text == self._draft_text:
            return
        self._draft_text = text
        self._draft = self.cascade.preview(text)

    def on_final(self, text: str) -> Reply:
        """Оператор договорил. Если текст совпал с гипотезой — ответ готов."""
        text = text.strip()
        if self._draft is not None and text == self._draft_text:
            u = self._draft
        else:
            u = self.cascade.understand(text)
        self._draft = None
        self._draft_text = ""
        return self._advance(text, u)

    def cancel(self) -> str | None:
        """Barge-in: оператор перебил. Спекуляция снимается."""
        self._draft = None
        self._draft_text = ""
        return self.turns[-1].reply.audio_id if self.turns else None

    # ---------------------------------------------------------- текстовый вход

    def say(self, text: str) -> Reply:
        """То же самое без частичных гипотез — для тестов и разбора."""
        return self.on_final(text)

    # ------------------------------------------------------------------ общее

    def _advance(self, text: str, u: Understanding) -> Reply:
        d = decide(u, self.state, self.scenario)
        reply = self.renderer.say(d)
        self.turns.append(Turn(len(self.turns) + 1, text, u, d, reply))
        return reply

    def close(self) -> None:
        self.cascade.save()
