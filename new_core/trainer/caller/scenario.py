# -*- coding: utf-8 -*-
"""Сценарий: факты заявителя, его персона, критичные для карточки поля."""
from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path

from .contracts import Mood, Style


@dataclass
class Fact:
    key: str
    group: str
    questions: list[str]                      # как об этом спрашивают
    answers: dict[str, str]                   # plain / short / confirm / correct
    value_numbers: set[int] = field(default_factory=set)
    requires: list[str] = field(default_factory=list)
    audio: dict[str, str] = field(default_factory=dict)

    def answer(self, style: Style) -> str:
        return self.answers.get(style.value) or self.answers["plain"]


@dataclass
class Profile:
    """Настройка преподавателя. По умолчанию — спокойный заявитель."""
    name: str = "calm"
    max_facts_per_turn: int = 2
    mood_ceiling: Mood = Mood.WORRIED
    idle_turns_to_worry: int = 3
    critical_wait: int = 6
    initiative_cooldown: int = 5
    min_confidence: float = 0.45

    @staticmethod
    def preset(name: str) -> "Profile":
        if name == "calm":
            return Profile()
        if name == "normal":
            return Profile(name, 2, Mood.TENSE, 2, 5, 4, 0.45)
        if name == "hard":
            return Profile(name, 1, Mood.TENSE, 2, 4, 3, 0.55)
        raise ValueError(f"нет профиля {name}")


@dataclass
class Scenario:
    id: str
    facts: dict[str, Fact]
    critical: list[str]
    profile: Profile
    opening: str
    meta: dict

    @staticmethod
    def load(path: str | Path) -> "Scenario":
        raw = json.loads(Path(path).read_text(encoding="utf-8"))
        facts = {}
        for f in raw["facts"]:
            a = dict(f["answers"])
            a.setdefault("short", a["plain"])
            a.setdefault("confirm", "Да, всё верно")
            facts[f["key"]] = Fact(
                key=f["key"], group=f.get("group", f["key"]),
                questions=f["questions"], answers=a,
                value_numbers=set(f.get("value_numbers", [])),
                requires=f.get("requires", []),
                audio=f.get("audio", {}),
            )
        persona = raw.get("persona", {})
        return Scenario(
            id=raw["id"], facts=facts,
            critical=raw.get("critical", []),
            profile=Profile.preset(persona.get("profile", "calm")),
            opening=persona.get("opening", "Алло! Помогите!"),
            meta=raw.get("meta", {}),
        )

    def unknown_keys(self, keys: list[str]) -> list[str]:
        return [k for k in keys if k not in self.facts]
