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
class DialogContext:
    """Снимок диалога для NLU (эллипсисы и LM Studio)."""
    recent_ops: list[str] = field(default_factory=list)   # последние реплики оператора
    recent_keys: list[str] = field(default_factory=list)  # недавно обсуждавшиеся факты

    def nonempty(self) -> bool:
        return bool(self.recent_ops or self.recent_keys)


@dataclass
class Timing:
    """Кто сколько занял на ходе. Заполняется Engine + NLU."""
    total_ms: float = 0.0
    understand_ms: float = 0.0
    decide_ms: float = 0.0
    speak_ms: float = 0.0
    lexical_ms: float = 0.0
    llm_called: bool = False
    llm_ms: float = 0.0
    llm_status: str = "skip"   # skip | cache | ok | timeout | error
    nlu_path: str = "lexical"  # lexical | hybrid_llm | cache | act

    def format(self) -> str:
        llm = self.llm_status
        if self.llm_called or self.llm_status not in ("skip",):
            llm = f"{self.llm_status}:{self.llm_ms:.0f}"
        return (f"timing total={self.total_ms:.0f}ms "
                f"understand={self.understand_ms:.0f} "
                f"decide={self.decide_ms:.0f} "
                f"speak={self.speak_ms:.0f} | "
                f"lex={self.lexical_ms:.0f} "
                f"llm={llm} path={self.nlu_path}")


@dataclass
class Understanding:
    """Этап 2 -> этап 3. О чём спросили, но не какими словами."""
    keys: list[str] = field(default_factory=list)
    act: Act = Act.ASK
    confidence: float = 0.0
    words: int = 0
    value_mismatch: bool = False
    source: str = ""            # lexical | llm | cache | hybrid — для журнала
    topical: int = 0            # смысловых слов, кроме местоимений
    timing: Timing = field(default_factory=Timing)


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
    timing: Timing = field(default_factory=Timing)
