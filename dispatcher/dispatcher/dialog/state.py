# -*- coding: utf-8 -*-
"""Состояние звонка. Всё, что политика помнит между репликами."""

from __future__ import annotations

from dataclasses import dataclass, field

from ..types import Mood


@dataclass(slots=True)
class CallState:
    revealed: set[str] = field(default_factory=set)  # ключи уже названных фактов
    asked: dict[str, int] = field(default_factory=dict)
    turn: int = 0
    idle_turns: int = 0  # ходов подряд без толку
    mood: Mood = Mood.COMPOSED
    last_initiative: int = -99
    last_reveal: tuple[str, ...] = ()  # для просьбы повторить

    def reset(self) -> None:
        self.revealed.clear()
        self.asked.clear()
        self.turn = self.idle_turns = 0
        self.mood = Mood.COMPOSED
        self.last_initiative = -99
        self.last_reveal = ()
