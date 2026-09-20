# -*- coding: utf-8 -*-
"""
Чтение сценариев и раскладка их фактов по слотам онтологии.

Канонический формат факта: {id, slot, numbers?, questions?, answers,
disclosure?, requires?} — одни английские идентификаторы, русский только
в текстах для людей. Старые файлы {key, group, value_numbers} читаются
как раньше: слот выводится онтологией, key работает как id.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path

from ..types import Disclosure, Fact, Profile, Scenario
from .ontology import Ontology

DATA = Path(__file__).resolve().parents[2] / "data"


@dataclass(slots=True)
class LoadIssue:
    scenario: str
    key: str
    kind: str  # unmapped | crowded
    detail: str = ""


@dataclass(slots=True)
class Loaded:
    scenarios: dict[str, Scenario]
    issues: list[LoadIssue] = field(default_factory=list)
    # slot -> формулировки со всего корпуса, в порядке появления
    questions: dict[str, list[str]] = field(default_factory=dict)
    # тот же порядок: из какого сценария пришла каждая формулировка.
    # Нужен, чтобы мерить на сценарии, которого банк не видел.
    sources: dict[str, list[str]] = field(default_factory=dict)

    @property
    def ok(self) -> bool:
        return not self.issues


def load_scenario(
    path: Path, onto: Ontology, issues: list[LoadIssue] | None = None
) -> Scenario:
    raw = json.loads(path.read_text(encoding="utf-8"))
    sid = raw["id"]
    issues = issues if issues is not None else []

    facts: dict[str, Fact] = {}
    by_slot: dict[str, list[str]] = {}

    for item in raw["facts"]:
        key = item.get("id", item.get("key"))
        if key is None:
            issues.append(LoadIssue(sid, "?", "unmapped", "у факта нет id"))
            continue
        slot = item.get("slot") or onto.resolve(
            key, item.get("group") or "", sid
        )
        if slot is None:
            issues.append(
                LoadIssue(sid, key, "unmapped", f'группа «{item.get("group", "")}»')
            )
            continue

        answers = dict(item["answers"])
        answers.setdefault("short", answers["plain"])
        answers.setdefault("confirm", "Да, всё верно.")

        disclosure = (
            Disclosure(item["disclosure"])
            if "disclosure" in item
            else onto.slots[slot].default_disclosure
        )

        fact = Fact(
            key=key,
            slot=slot,
            answers=answers,
            numbers=frozenset(item.get("numbers", item.get("value_numbers", ()))),
            requires=tuple(item.get("requires", ())),
            disclosure=disclosure,
            questions=tuple(item.get("questions", ())),
            audio=dict(item.get("audio", {})),
        )
        facts[key] = fact
        by_slot.setdefault(slot, []).append(key)

    # слот с тремя и более фактами — сигнал, что слот слишком крупный:
    # ответ из трёх кусков оператор не запишет
    for slot, keys in by_slot.items():
        if len(keys) > 2:
            issues.append(
                LoadIssue(
                    sid, ", ".join(keys), "crowded", f"{len(keys)} факта в слоте {slot}"
                )
            )

    # кто отвечает за слот, которого у этого заявителя нет
    answers_for: dict[str, str] = {}
    for sid_slot, slot_def in onto.slots.items():
        if sid_slot in by_slot:
            continue
        for other in slot_def.fallback:
            if other in by_slot:
                answers_for[sid_slot] = other
                break

    persona = raw.get("persona", {})
    critical = tuple(k for k in raw.get("critical", ()) if k in facts)

    return Scenario(
        id=sid,
        facts=facts,
        critical=critical,
        profile=Profile.preset(persona.get("profile", "calm")),
        opening=persona.get("opening", "Алло! Помогите!"),
        meta=raw.get("meta", {}),
        by_slot=by_slot,
        answers_for=answers_for,
    )


def load_all(onto: Ontology | None = None, root: Path = DATA / "scenarios") -> Loaded:
    onto = onto or Ontology.load()
    issues: list[LoadIssue] = []
    scenarios: dict[str, Scenario] = {}
    questions: dict[str, list[str]] = {}
    sources: dict[str, list[str]] = {}
    seen: dict[str, set[str]] = {}

    for path in sorted(root.glob("*.json")):
        sc = load_scenario(path, onto, issues)
        scenarios[sc.id] = sc
        for fact in sc.facts.values():
            bucket = questions.setdefault(fact.slot, [])
            origin = sources.setdefault(fact.slot, [])
            dedup = seen.setdefault(fact.slot, set())
            for q in fact.questions:
                norm = " ".join(q.lower().replace("ё", "е").split()).strip(" ?.!")
                if norm and norm not in dedup:
                    dedup.add(norm)
                    bucket.append(q)
                    origin.append(sc.id)

    _merge_audio_index(scenarios)

    return Loaded(scenarios, issues, questions, sources)


def _merge_audio_index(scenarios: dict[str, Scenario]) -> None:
    """Предрендер шага 2: audio_id из data/audio/index.json -> Fact.audio.

    Индекса нет (синтез ещё не гоняли) — молча ничего, Reply.audio_id
    остаётся None и плеер говорит через живой TTS по Reply.text."""
    try:
        raw = (DATA / "audio" / "index.json").read_text(encoding="utf-8")
    except OSError:
        return
    files = json.loads(raw).get("files", {})
    for aid, rel in files.items():
        if not aid.startswith("a/"):
            continue
        _, sid, fkey_fs, style_ext = aid.split("/", 3)
        style = style_ext.removesuffix(".wav")
        sc = scenarios.get(sid)
        if sc is None:
            continue
        # fskey необратим (# -> _), поэтому ищем по санитизированному
        for key, fact in sc.facts.items():
            if key.replace("#", "_").replace("/", "_") == fkey_fs:
                fact.audio.setdefault(style, rel)
                break
