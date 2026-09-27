# -*- coding: utf-8 -*-
"""
Типы, которыми обмениваются слои ядра.

Здесь и только здесь описано, что слои знают друг о друге. Понимание не знает
текстов ответов, политика не знает, как реплику распознали, рендер не знает
состояния звонка.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import IntEnum, StrEnum
from typing import Protocol


class Act(StrEnum):
    """Что оператор сделал репликой."""

    ASK = "ask"  # спросил
    CONFIRM = "confirm"  # переспросил: «то есть тринадцатый, верно?»
    ASSERT = "assert"  # утверждает, не спрашивая
    SPEECH_ACT = "speech_act"  # «бригада выехала», «оставайтесь на линии»


class Style(StrEnum):
    """В какой манере заявитель отвечает."""

    PLAIN = "plain"
    SHORT = "short"  # уже спрашивали — отвечает суше
    CONFIRM = "confirm"  # подтверждение переспроса
    CORRECT = "correct"  # оператор назвал не то число
    DONT_KNOW = "dont_know"  # спросили о том, чего заявитель знать не может
    MISHEAR = "mishear"  # реплика без смысловых слов — «не расслышала»
    SLOW_DOWN = "slow_down"  # залп вопросов разом
    ACK = "ack"  # ответ на речевой акт


class Mood(IntEnum):
    COMPOSED = 0
    WORRIED = 1
    TENSE = 2


class Disclosure(StrEnum):
    """Когда заявитель выдаёт сведение.

    ON_REQUEST — уточнение «в скобках» из билета: заявитель называет его
    только по прямому вопросу оператора. Главная учебная механика.
    """

    VOLUNTEERED = "volunteered"
    ON_REQUEST = "on_request"


class SlotKind(StrEnum):
    VALUE = "value"  # адрес, номер, ФИО — оператор записывает
    YESNO = "yesno"  # «газом пахнет?» — да/нет
    DESCRIPTION = "description"  # «что горит?» — свободное описание


# --------------------------------------------------------------- онтология


@dataclass(frozen=True, slots=True)
class Slot:
    """Единица сведений, общая для всех сценариев.

    Формулировки принадлежат слоту, а не факту: за слот «улица» проголосовали
    десятки сценариев, поэтому новый сценарий получает их даром.
    """

    id: str
    label: str
    kind: SlotKind = SlotKind.VALUE
    family: str = ""  # addr, caller, victim… — для эллипсиса
    aliases: tuple[str, ...] = ()  # ключи фактов в старых сценариях
    # Если слота в сценарии нет, за него отвечают эти — по порядку.
    # «Назовите адрес» при отсутствии addr.full уходит в addr.street:
    # для этого заявителя улица и есть адрес.
    fallback: tuple[str, ...] = ()
    questions: tuple[str, ...] = ()  # формулировки из онтологии
    urge: str = ""  # как заявитель сам напомнит о слоте
    default_disclosure: Disclosure = Disclosure.VOLUNTEERED
    since: str = "0.1"

    def __post_init__(self) -> None:
        if not self.family:
            object.__setattr__(self, "family", self.id.split(".")[0])


# ---------------------------------------------------------------- сценарий


@dataclass(slots=True)
class Fact:
    """Одно сведение конкретного заявителя."""

    key: str  # id факта: голый слот, при дележе слота — слот#2, слот#3
    slot: str  # id слота онтологии
    answers: dict[str, str]  # стиль -> текст
    numbers: frozenset[int] = frozenset()  # числа в значении — для CORRECT
    requires: tuple[str, ...] = ()  # ключи фактов-предусловий
    disclosure: Disclosure = Disclosure.VOLUNTEERED
    questions: tuple[str, ...] = ()  # локальные формулировки, если есть
    audio: dict[str, str] = field(default_factory=dict)

    def answer(self, style: Style) -> str:
        return self.answers.get(style.value) or self.answers["plain"]


@dataclass(slots=True)
class Profile:
    """Характер заявителя. Настраивается преподавателем."""

    name: str = "calm"
    max_facts_per_turn: int = 2
    mood_ceiling: Mood = Mood.WORRIED
    idle_turns_to_worry: int = 3
    critical_wait: int = 6
    initiative_cooldown: int = 5

    @staticmethod
    def preset(name: str) -> "Profile":
        match name:
            case "calm":
                return Profile()
            case "normal":
                return Profile(name, 2, Mood.TENSE, 2, 5, 4)
            case "hard":
                return Profile(name, 1, Mood.TENSE, 2, 4, 3)
        raise ValueError(f"нет профиля {name}")


@dataclass(slots=True)
class Scenario:
    id: str
    facts: dict[str, Fact]
    critical: tuple[str, ...]
    profile: Profile
    opening: str
    meta: dict
    # Слот -> ключи фактов. Обычно один, но «отёк руки» и «отёк ноги» —
    # это один вопрос «какие травмы?» и один ответ из двух частей.
    by_slot: dict[str, list[str]] = field(default_factory=dict)
    # Слот, которого у заявителя нет -> слот, который отвечает вместо него.
    answers_for: dict[str, str] = field(default_factory=dict)

    def facts_for(self, slot: str) -> list[Fact]:
        """Факты слота: сначала те, что заявитель называет сам."""
        keys = self.by_slot.get(slot) or self.by_slot.get(
            self.answers_for.get(slot, ""), ()
        )
        return sorted(
            (self.facts[k] for k in keys),
            key=lambda f: f.disclosure is Disclosure.ON_REQUEST,
        )

    @property
    def slots(self) -> frozenset[str]:
        """Слоты, на которые заявитель может ответить — свои и замещённые."""
        return frozenset(self.by_slot) | frozenset(self.answers_for)


# ------------------------------------------------------- обмен между слоями


@dataclass(slots=True)
class Understanding:
    """Понимание -> политика. О чём спросили, но не какими словами.

    Различие `slots` и `keys` принципиально: непустые slots при пустых keys
    означают «оператор спросил о понятном, но этот заявитель того не знает»,
    и это не то же самое, что «реплику не разобрали».
    """

    slots: list[str] = field(default_factory=list)
    keys: list[str] = field(default_factory=list)
    act: Act = Act.ASK
    score: float = 0.0
    words: int = 0
    topical: int = 0  # смысловых слов, кроме служебных
    value_mismatch: bool = False  # оператор назвал не то число
    source: str = ""  # cache | rules | bank | arbiter — для журнала
    latency_ms: float = 0.0


@dataclass(slots=True)
class Decision:
    """Политика -> рендер. Что раскрыть и в какой манере."""

    reveal: list[str] = field(default_factory=list)
    style: Style = Style.PLAIN
    mood: Mood = Mood.COMPOSED
    unprompted: str | None = None  # ключ факта, о котором заявитель напомнил сам


@dataclass(slots=True)
class Reply:
    text: str
    audio_id: str | None = None  # место под заранее отрендеренное аудио
    style: Style = Style.PLAIN
    mood: Mood = Mood.COMPOSED


class Understander(Protocol):
    """Всё, что умеет превращать реплику в понимание.

    Каскад с банком и понимание целиком на модели взаимозаменяемы: сессия
    знает только этот метод и о внутренностях не догадывается.
    """

    def understand(self, text: str) -> "Understanding": ...

    def preview(self, text: str) -> "Understanding | None": ...

    def save(self) -> None: ...


@dataclass(slots=True)
class Turn:
    """Строка журнала. Всё, что нужно разбору занятия."""

    n: int
    utterance: str
    understanding: Understanding
    decision: Decision
    reply: Reply
