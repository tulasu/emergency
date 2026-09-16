# -*- coding: utf-8 -*-
"""Контракты между этапами. Всё, что этапы знают друг о друге."""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum


class Act(str, Enum):
    ASK = "ask"
    CONFIRM = "confirm"
    ASSERT = "assert"
    SPEECH_ACT = "speech_act"


class Style(str, Enum):
    PLAIN = "plain"
    SHORT = "short"
    CONFIRM = "confirm"
    CORRECT = "correct"
    DONT_KNOW = "dont_know"
    MISHEAR = "mishear"
    SLOW_DOWN = "slow_down"
    ACK = "ack"


class Mood(int, Enum):
    COMPOSED = 0
    WORRIED = 1
    TENSE = 2


@dataclass
class Understanding:
    """Этап 2 -> этап 3. О чём спросили, но не какими словами."""
    keys: list[str] = field(default_factory=list)
    act: Act = Act.ASK
    confidence: float = 0.0
    words: int = 0
    value_mismatch: bool = False
    source: str = ""            # lexical | llm | cache — для журнала
    topical: int = 0            # смысловых слов, кроме местоимений


@dataclass
class Decision:
    """Этап 3 -> этап 4. Что раскрыть и в какой манере."""
    reveal: list[str] = field(default_factory=list)
    style: Style = Style.PLAIN
    mood: Mood = Mood.COMPOSED
    unprompted: str | None = None


@dataclass
class Reply:
    """Этап 4 -> наружу."""
    text: str
    audio: str | None = None    # путь к заранее сгенерированному файлу


@dataclass
class Turn:
    """Строка журнала. Всё, что нужно оценке."""
    n: int
    utterance: str
    understanding: Understanding
    decision: Decision
    reply: Reply
