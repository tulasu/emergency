# -*- coding: utf-8 -*-
"""Session lifecycle + bank/ontology swap + per-call state."""

from __future__ import annotations

import json
import os
import threading
import urllib.request
import uuid as _uuid
from pathlib import Path

from dialog.core.data.ontology import Ontology  # frozen kinds/labels
from dialog.core.nlu.bank import LexicalBank
from dialog.core.nlu.cascade import Cascade
from dialog.core.session import Session

from .bank_source import Bank, BankHolder
from .scenario import ontology_from_snapshot, scenario_from_snapshot
from .validator import SnapshotError, validate_snapshot


def _int_env(key: str, fallback: int) -> int:
    """Non-numeric env never crashes boot (spec O)."""
    try:
        return int(os.environ.get(key, fallback))
    except (TypeError, ValueError):
        return fallback


SERVICE_TOKEN = os.environ.get("INTERNAL_SERVICE_TOKEN", "")
TRAINEEBOX_URL = os.environ.get("TRAINEEBOX_URL", "").rstrip("/")
WAV_ROOT = Path(os.environ.get("DIALOG_WAV_DIR", "/tmp/dialog-wav"))
MAX_SESSIONS = _int_env("DIALOG_MAX_SESSIONS", 1024)
WAV_KEEP = _int_env("DIALOG_WAV_KEEP", 100)


class BusyError(RuntimeError):
    """Sessions capped: too many concurrent calls."""


HOLDER = BankHolder()
SESSIONS: dict[str, dict] = {}
GUARD = threading.Lock()
# Bank+ontology swap together under one guard so open_session never sees
# a new bank with an old ontology (one version — one swap).
_SWAP_GUARD = threading.Lock()
ONTO = Ontology(version="empty", slots={}, by_alias={}, overrides={})
_LEXICAL_CACHE: dict[str, LexicalBank] = {}
_LEXICAL_GUARD = threading.Lock()


def norm_sid(session_id: str) -> str:
    """Canonical call_id: hyphenated lowercase UUID.

    AudioSocket delivers the UUID as 16 raw bytes (hex, no dashes) while
    traineebox generates hyphenated text — normalize both forms (spec A),
    else every voice call misses its session and closes."""
    return str(_uuid.UUID(str(session_id).strip()))


def open_session(session_id: str, snapshot: dict, bank: Bank | None = None,
                 expected_digest: str = "") -> str:
    """Validate + build a session bound to one bank snapshot. Raises KeyError/ValueError."""
    try:
        sid = norm_sid(session_id)
    except ValueError:
        raise SnapshotError("session_id must be UUID (call_id)")
    with _SWAP_GUARD:
        if bank is None:
            bank = HOLDER.current  # locked: bank+onto swap together
        onto_used = ONTO
    if expected_digest and expected_digest != bank.digest:
        raise SnapshotError("bank digest mismatch: call is on a stale bank")
    norm = validate_snapshot(snapshot, bank.slot_ids)
    lexical = _lexical_for(bank)
    sc = scenario_from_snapshot(norm)
    cascade = Cascade(scenario=sc, lexical=lexical, ontology=onto_used, thin_rescue=False)
    impro = None
    if os.environ.get("DIALOG_PROD") == "1":
        impro = _frozen_improviser()
        cascade = _frozen_ensemble(sc, lexical, cascade, onto_used)
    sess = Session.open(sc, cascade, onto_used)
    sess.improv = impro
    with GUARD:
        if len(SESSIONS) >= MAX_SESSIONS:
            raise BusyError(f"too many sessions (cap {MAX_SESSIONS})")
        if sid in SESSIONS:
            raise KeyError(f"session {sid} exists")
        SESSIONS[sid] = {"session": sess, "lock": threading.Lock(),
                         "bank": bank, "onto": onto_used}
    return sess.opening()


def _lexical_for(bank: Bank) -> LexicalBank:
    """Cached lexical bank per digest; hot call path never hits DB."""
    with _LEXICAL_GUARD:
        hit = _LEXICAL_CACHE.get(bank.digest)
        if hit is not None:
            return hit
    if bank.questions:
        sources = {s: ["bank:" + bank.version] * len(q) for s, q in bank.questions.items()}
        built = LexicalBank.build(bank.questions, sources)
    else:
        # Empty bank (before first reload): empty lexical, no file seed.
        # Open then fails at validate_snapshot (unknown slot → 400); lint
        # raises the same way (validate first, like open).
        built = LexicalBank.build({}, {})
    with _LEXICAL_GUARD:
        _LEXICAL_CACHE[bank.digest] = built
        while len(_LEXICAL_CACHE) > 4:  # ponytail: keep current + few old for in-flight
            _LEXICAL_CACHE.pop(next(iter(_LEXICAL_CACHE)))
    return built


def _frozen_improviser():
    """IMPROV=1 grounded (Qwen3-4B, 3s). Only in DIALOG_PROD=1 (needs llama-server)."""
    from dialog.core.dialog.improv import Improviser
    return Improviser(timeout=3.0)


# Prod voters are expensive to build: LayaArbiter cold-loads a 614MB checkpoint
# (~20s) in its warmup. The frozen core builds them once per process (the old
# dispatcher Service.__post_init__), not per call — a 5s traineebox timeout on
# /sessions/open cannot pay a per-session reload. Cache by ontology; the bank
# reload thread warms the cache so the first call is already resident.
_VOTERS_GUARD = threading.Lock()
_VOTERS_CACHE: dict[int, list] = {}


def _prod_voters(onto_used) -> list:
    from dialog.core.nlu.laya_arbiter import LayaArbiter
    from dialog.core.nlu.llm_arbiter import LlmArbiter
    key = id(onto_used)
    with _VOTERS_GUARD:
        hit = _VOTERS_CACHE.get(key)
        if hit is not None:
            return hit
        voters = [LayaArbiter(ontology=onto_used, device="cuda", min_confidence=0.5),
                  LlmArbiter(onto_used)]
        _VOTERS_CACHE[key] = voters
        while len(_VOTERS_CACHE) > 4:  # ponytail: current + few old for in-flight reloads
            _VOTERS_CACHE.pop(next(iter(_VOTERS_CACHE)))
        return voters


def _frozen_ensemble(sc, lexical, cascade, onto_used):
    """Dual voters (laya 0.5 + LLM) majority-ensemble, k=5, voter 3s. Prod only."""
    from dialog.core.nlu.ensemble import Ensemble
    return Ensemble(cascade, _prod_voters(onto_used), rule="majority")


def entry(session_id: str) -> dict:
    with GUARD:
        try:
            return SESSIONS[norm_sid(session_id)]
        except ValueError:
            raise KeyError(f"нет сессии {session_id}") from None
        except KeyError:
            raise KeyError(f"нет сессии {session_id}") from None


def final(session_id: str, text: str) -> dict:
    e = entry(session_id)
    with e["lock"]:
        reply = e["session"].on_final(text)
    return {"text": reply.text, "audio_id": reply.audio_id,
            "style": reply.style.value, "mood": int(reply.mood.value)}


def close(session_id: str) -> list[dict]:
    try:
        sid = norm_sid(session_id)
    except ValueError:
        raise KeyError(f"нет сессии {session_id}") from None
    with GUARD:
        e = SESSIONS.pop(sid, None)
    if e is None:
        raise KeyError(f"нет сессии {session_id}")
    with e["lock"]:
        turns = [{"n": t.n, "utterance": t.utterance, "reply": t.reply.text,
                  "style": t.decision.style.value} for t in e["session"].turns]
        try:
            e["session"].close()
        except Exception:
            pass
    _push_turns(sid, turns)
    return turns


def _push_turns(session_id: str, turns: list[dict]) -> None:
    """dialog writes only via POST /internal/calls/* on close (AD-1)."""
    if not TRAINEEBOX_URL or not turns:
        return
    body = json.dumps({"turns": turns}, ensure_ascii=False).encode()
    req = urllib.request.Request(
        f"{TRAINEEBOX_URL}/internal/calls/{session_id}/turns", data=body,
        headers={"Content-Type": "application/json",
                 "X-Service-Token": SERVICE_TOKEN}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=5) as r:
            r.read()
    except Exception as exc:  # noqa: BLE001 — close must not fail on push
        print(f"[dialog] turns push failed for {session_id}: {exc}", flush=True)


def lint_scenario(snapshot: dict) -> dict:
    with _SWAP_GUARD:
        bank = HOLDER.current  # locked: bank+onto swap together
        onto_used = ONTO
    norm = validate_snapshot(snapshot, bank.slot_ids)
    from dialog.core.data.authoring import lint
    sources = {s: ["bank:" + bank.version] * len(q) for s, q in bank.questions.items()}
    lexical = (LexicalBank.build(bank.questions, sources) if bank.questions
               else LexicalBank.build({}, {}))  # empty bank: validate raises first, like open
    sc = scenario_from_snapshot(norm)
    report = lint(sc, lexical, onto_used)
    unreachable = [h.key for h in report if h.key in sc.critical and h.reachable < 0.5]
    return {"id": sc.id, "facts": len(sc.facts), "critical": list(sc.critical),
            "unreachable": unreachable}


def _build_bank_payload(version: str, digest: str, slots: dict | None,
                        questions: dict | None) -> Bank:
    """Build the new bank (fan-out snapshot, else fetch). Raises ValueError on digest mismatch."""
    if slots:
        bank = Bank.build(version or "v1", slots, questions or {})
        if digest and digest != bank.digest:
            raise ValueError(f"reload digest mismatch: {digest} != {bank.digest}")
        return bank
    fetched_slots, fetched_questions = _fetch_bank(version)
    return Bank.build(version or "v1", fetched_slots, fetched_questions)


def _swap_pair(bank: Bank, ontology: Ontology | None) -> None:
    """Atomic bank+ontology swap; in-flight calls finish on old (AD-8)."""
    global ONTO
    with _SWAP_GUARD:
        HOLDER.swap(bank)
        if ontology is not None:
            ONTO = ontology


def _reload_background(version: str, digest: str = "",
                       slots: dict | None = None, questions: dict | None = None,
                       ontology: dict | None = None) -> None:
    """Background rebuild + atomic swap; a bad payload keeps the old pair and counts a failure."""
    try:
        bank = _build_bank_payload(version, digest, slots, questions)
        new_onto = ontology_from_snapshot(ontology) if ontology is not None else None
        _swap_pair(bank, new_onto)
        # Warm the prod voters for the new ontology off the request path:
        # the first call must not pay Laya's ~20s cold load (5s open timeout).
        if os.environ.get("DIALOG_PROD") == "1":
            _prod_voters(ONTO)
    except Exception as exc:  # noqa: BLE001
        print(f"[dialog] bank reload failed: {exc}", flush=True)
        HOLDER.mark_failure()


def _fetch_bank(version: str) -> tuple[dict, dict]:
    """Pull canon from traineebox. No local seed: empty until /bank/reload."""
    url = os.environ.get("TRAINEEBOX_BANK_URL", "")
    if not url:
        raise RuntimeError("TRAINEEBOX_BANK_URL not set: empty bank until /bank/reload")
    req = urllib.request.Request(url, headers={"X-Service-Token": SERVICE_TOKEN})
    with urllib.request.urlopen(req, timeout=30) as r:
        raw = json.loads(r.read())
    return raw["slots"], raw["questions"]
