# -*- coding: utf-8 -*-
"""Сборка: приём -> понимание -> решение -> речь -> журнал."""
from __future__ import annotations

import time

from .contracts import Reply, Timing, Turn
from .decide import CallerState, decide
from .scenario import Scenario
from .score import Journal
from .speak import Speaker
from .understand import Understander, build


class Engine:
    def __init__(self, scenario: Scenario, understander: str | Understander = "lexical",
                 seed: int | None = None):
        self.sc = scenario
        self.nlu = (understander if not isinstance(understander, str)
                    else build(scenario, understander))
        self.state = CallerState()
        self.speaker = Speaker(scenario, seed)
        self.journal = Journal(scenario)

    def opening(self) -> str:
        return self.sc.opening

    def handle(self, utterance: str) -> Turn:
        t_all = time.perf_counter()
        ctx = self.state.dialog_context()

        t0 = time.perf_counter()
        u = self.nlu.understand(utterance, context=ctx)
        understand_ms = (time.perf_counter() - t0) * 1000

        t0 = time.perf_counter()
        d = decide(u, self.state, self.sc)
        decide_ms = (time.perf_counter() - t0) * 1000

        t0 = time.perf_counter()
        r: Reply = self.speaker.say(d)
        speak_ms = (time.perf_counter() - t0) * 1000

        # история для следующих эллипсисов: реплика + запрошенные/раскрытые ключи
        remembered = list(dict.fromkeys([*(u.keys or []), *(d.reveal or [])]))
        self.state.remember(utterance, remembered)

        timing = Timing(
            total_ms=(time.perf_counter() - t_all) * 1000,
            understand_ms=understand_ms,
            decide_ms=decide_ms,
            speak_ms=speak_ms,
            lexical_ms=u.timing.lexical_ms,
            llm_called=u.timing.llm_called,
            llm_ms=u.timing.llm_ms,
            llm_status=u.timing.llm_status,
            nlu_path=u.timing.nlu_path,
        )
        u.timing = timing
        turn = Turn(len(self.journal.turns) + 1, utterance, u, d, r, timing)
        self.journal.add(turn)
        return turn

    def close(self) -> str:
        save = getattr(self.nlu, "save", None)
        if save:
            save()
        return self.journal.render()
