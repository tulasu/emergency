# -*- coding: utf-8 -*-
"""Snapshot JSON Schema canon (AD-7). Dialog rejects `open` with 400 on drift.

Forbidden in snapshots (spec Never): aliases/overrides/since/urge/mood/
presets/drift, partial-preview, recording_path, runtime meta, mode=both,
student-dials-PIN. The validator rejects them so the frozen dataclasses
(which still carry those fields for file-replay parity) never see them.
"""

from __future__ import annotations

from typing import Any

FORBIDDEN_KEYS = (
    "aliases", "overrides", "since", "urge", "mood", "preset", "presets",
    "drift", "preview", "recording_path", "meta", "pin", "mode",
)

# Snapshot shape canon: unknown top-level keys are drift, not data (AD-7).
# Mirrored in traineebox/internal/dialog/validator.go — keep identical.
ALLOWED_TOP_KEYS = ("id", "opening", "mode", "critical", "facts")
MAX_OPENING_LEN = 2000
MAX_FACTS = 256

DISCLOSURES = ("volunteered", "on_request")


class SnapshotError(ValueError):
    pass


def validate_snapshot(raw: dict[str, Any], known_slots: set[str] | None = None) -> dict[str, Any]:
    """Validate open snapshot. Returns normalized dict or raises SnapshotError."""
    if not isinstance(raw, dict):
        raise SnapshotError("scenario must be an object")
    for k in FORBIDDEN_KEYS:
        if k in raw and k not in ("mode",):
            raise SnapshotError(f"forbidden key {k!r} in snapshot")
    for k in raw:
        if k not in ALLOWED_TOP_KEYS:
            raise SnapshotError(f"unknown key {k!r} in snapshot")
    mode = raw.get("mode", "voice")
    if mode == "both":
        raise SnapshotError("mode=both forbidden")
    if mode not in ("voice", "text"):
        raise SnapshotError(f"bad mode {mode!r}")
    sid = raw.get("id") or ""
    if not sid:
        raise SnapshotError("scenario.id required")
    opening = raw.get("opening")
    if not isinstance(opening, str) or not opening.strip():
        raise SnapshotError("scenario.opening required")
    if len(opening) > MAX_OPENING_LEN:
        raise SnapshotError("scenario.opening too long")
    facts = raw.get("facts")
    if not isinstance(facts, list) or not facts:
        raise SnapshotError("scenario.facts empty")
    if len(facts) > MAX_FACTS:
        raise SnapshotError("scenario.facts too many")
    keys: set[str] = set()
    for f in facts:
        if not isinstance(f, dict):
            raise SnapshotError("fact must be an object")
        key, slot = f.get("key") or f.get("id") or "", f.get("slot") or ""
        if not key or not slot:
            raise SnapshotError("fact key/slot required")
        if known_slots is not None and slot not in known_slots:
            raise SnapshotError(f"unknown slot {slot!r} for fact {key!r}")
        answers = f.get("answers") or {}
        if not isinstance(answers, dict) or not (answers.get("plain") or "").strip():
            raise SnapshotError(f"fact {key!r} needs answers.plain")
        if "audio" in f:
            raise SnapshotError(f"fact {key!r}: Fact.audio forbidden in snapshot")
        disc = f.get("disclosure", "volunteered")
        if disc not in DISCLOSURES:  # unknown string → 400, never 500 in open (spec O)
            raise SnapshotError(f"fact {key!r}: bad disclosure {disc!r}")
        if key in keys:
            raise SnapshotError(f"duplicate fact {key!r}")
        keys.add(key)
    for c in raw.get("critical") or ():
        if c not in keys:
            raise SnapshotError(f"critical {c!r} not a fact")
    for f in facts:
        key = f.get("key") or f.get("id") or ""
        for r in f.get("requires") or ():
            if r not in keys:
                raise SnapshotError(f"fact {key!r} requires unknown {r!r}")
    return {
        "id": sid,
        "opening": raw["opening"],
        "critical": list(raw.get("critical") or ()),
        "facts": [
            {
                "key": f.get("key") or f.get("id"),
                "slot": f["slot"],
                "answers": dict(f["answers"]),
                "numbers": list(f.get("numbers") or ()),
                "requires": list(f.get("requires") or ()),
                "disclosure": f.get("disclosure", "volunteered"),
            }
            for f in facts
        ],
        "mode": mode,
    }
