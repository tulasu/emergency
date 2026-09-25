# -*- coding: utf-8 -*-
"""Ontology/scenario builders from snapshot JSON."""

from __future__ import annotations

from dialog.core.data.ontology import Ontology  # frozen kinds/labels
from dialog.core.types import Disclosure, Fact, Profile, Scenario, Slot, SlotKind


def onto() -> Ontology:
    """Current ontology snapshot. Empty until the first /bank/reload."""
    # Lazy import: lifecycle imports scenario (builders), so we read ONTO live.
    from . import lifecycle
    return lifecycle.ONTO


def ontology_from_snapshot(raw: dict | None) -> Ontology:
    """Build Ontology from the /bank/reload payload fragment.

    Only `id`+`label` required per slot; rest = Slot defaults; `by_alias`
    rebuilt from `aliases` verbatim like `Ontology.load`. Raises ValueError
    on bad slot/kind so the reload keeps the old bank+ontology."""
    if not isinstance(raw, dict):
        raise ValueError("ontology must be an object")
    items = raw.get("slots", [])
    if not isinstance(items, list):
        raise ValueError("ontology.slots must be a list")
    if not items:
        raise ValueError("ontology.slots must be a non-empty list")
    slots: dict[str, Slot] = {}
    by_alias: dict[str, str] = {}
    for item in items:
        if not isinstance(item, dict):
            raise ValueError("ontology slot must be an object")
        sid = item.get("id", "")
        label = item.get("label", "")
        if not isinstance(sid, str) or not sid or "." not in sid:
            raise ValueError(f"bad slot id {sid!r}")
        if sid in slots:
            raise ValueError(f"slot {sid} declared twice")
        if not isinstance(label, str) or not label:
            raise ValueError(f"slot {sid} needs a label")
        try:
            kind = SlotKind(item.get("kind", "value"))
        except ValueError:
            raise ValueError(f"slot {sid}: bad kind {item.get('kind')!r}") from None
        try:
            disclosure = Disclosure(item.get("disclosure", "volunteered"))
        except ValueError:
            raise ValueError(f"slot {sid}: bad disclosure") from None
        aliases_raw = item.get("aliases", ())
        if isinstance(aliases_raw, str) or not isinstance(aliases_raw, (list, tuple)):
            raise ValueError(f"slot {sid}: aliases must be a list")
        aliases = tuple(aliases_raw)
        for a in aliases:
            if not isinstance(a, str) or not a:
                raise ValueError(f"slot {sid}: bad alias {a!r}")
            if a in by_alias:
                raise ValueError(f"alias {a!r} taken by {by_alias[a]}, repeat in {sid}")
            by_alias[a] = sid
        urge = item.get("urge", "")
        if not isinstance(urge, str):
            raise ValueError(f"slot {sid}: urge must be a string")
        slots[sid] = Slot(
            id=sid, label=label, kind=kind, aliases=aliases,
            fallback=tuple(item.get("fallback", ())),
            questions=tuple(item.get("questions", ())),
            urge=urge, default_disclosure=disclosure,
            since=str(item.get("since", "0.1")),
        )
    for sid, slot in slots.items():
        for other in slot.fallback:
            if other not in slots:
                raise ValueError(f"{sid}: fallback to unknown slot {other}")
    overrides = dict(raw.get("overrides") or {})
    for ref, target in overrides.items():
        if target not in slots:
            raise ValueError(f"override {ref} to unknown slot {target}")
        if ":" not in ref:
            raise ValueError(f"override must look like scenario:key, not {ref}")
    return Ontology(str(raw.get("version", "0.1")), slots, by_alias, overrides)


def scenario_from_snapshot(norm: dict) -> Scenario:
    facts: dict[str, Fact] = {}
    by_slot: dict[str, list[str]] = {}
    for f in norm["facts"]:
        key, slot = f["key"], f["slot"]
        answers = dict(f["answers"])
        answers.setdefault("short", answers["plain"])
        answers.setdefault("confirm", "Да, всё верно.")
        facts[key] = Fact(
            key=key, slot=slot, answers=answers,
            numbers=frozenset(f.get("numbers") or ()),
            requires=tuple(f.get("requires") or ()),
            disclosure=Disclosure(f.get("disclosure", "volunteered")),
        )
        by_slot.setdefault(slot, []).append(key)
    # ponytail: default Profile() == calm without the presets() lookup table.
    return Scenario(id=norm["id"], facts=facts, critical=tuple(norm["critical"]),
                    profile=Profile(), opening=norm["opening"],
                    meta={}, by_slot=by_slot, answers_for={})
