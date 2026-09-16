# -*- coding: utf-8 -*-
"""Сборка: приём -> понимание -> решение -> речь -> журнал."""
from __future__ import annotations

from .contracts import Reply, Turn
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
        u = self.nlu.understand(utterance)
        d = decide(u, self.state, self.sc)
        r: Reply = self.speaker.say(d)
        turn = Turn(len(self.journal.turns) + 1, utterance, u, d, r)
        self.journal.add(turn)
        return turn

    def close(self) -> str:
        save = getattr(self.nlu, "save", None)
        if save:
            save()
        return self.journal.render()
