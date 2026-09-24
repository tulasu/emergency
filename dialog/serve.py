# -*- coding: utf-8 -*-
"""Composition root: stateless snapshot executor over the frozen ML core.

Frozen core 1:1 (dialog/core, vendored): e5-small-tuned, torch/cuda, GigaAM,
majority-ensemble, IMPROV=1 grounded, dual voters (laya 0.5 + LLM),
Qwen3-4B, voter/improv 3s, k=5, tuned cascade thresholds. This file never
reimplements understanding/policy/render — it only owns sessions, locks,
routing, validation, bank swap and the traineebox close-callback.

Deliberately absent (spec Never): /sessions/check|partial|cancel, /rtp,
StubSTT, vosk/whisper, thin_rescue, Fact.audio, runtime meta,
recording_path, mode=both, student-dials-PIN, partial-preview.
"""

from __future__ import annotations

import json
import os
import threading
import time
import urllib.request
import uuid as _uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from dialog.core.data.ontology import Ontology  # frozen kinds/labels
from dialog.core.nlu.bank import LexicalBank
from dialog.core.nlu.cascade import Cascade
from dialog.core.session import Session
from dialog.core.types import Disclosure, Fact, Profile, Scenario, Slot, SlotKind

from .bank_source import Bank, BankHolder  # noqa: E402
from .validator import SnapshotError, validate_snapshot  # noqa: E402

SERVICE_TOKEN = os.environ.get("INTERNAL_SERVICE_TOKEN", "")
TRAINEEBOX_URL = os.environ.get("TRAINEEBOX_URL", "").rstrip("/")
WAV_ROOT = Path(os.environ.get("DIALOG_WAV_DIR", "/tmp/dialog-wav"))


def _int_env(key: str, fallback: int) -> int:
    """Non-numeric env never crashes boot (spec O)."""
    try:
        return int(os.environ.get(key, fallback))
    except (TypeError, ValueError):
        return fallback


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


def onto() -> Ontology:
    """Current ontology snapshot. Empty until the first /bank/reload."""
    return ONTO


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


def _frozen_ensemble(sc, lexical, cascade, onto_used):
    """Dual voters (laya 0.5 + LLM) majority-ensemble, k=5, voter 3s. Prod only."""
    from dialog.core.nlu.ensemble import Ensemble
    from dialog.core.nlu.laya_arbiter import LayaArbiter
    from dialog.core.nlu.llm_arbiter import LlmArbiter
    voters = [LayaArbiter(ontology=onto_used, device="cuda", min_confidence=0.5),
              LlmArbiter(onto_used)]
    return Ensemble(cascade, voters, rule="majority")


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


def _check_token(headers) -> bool:
    if not SERVICE_TOKEN:
        return True
    return headers.get("X-Service-Token") == SERVICE_TOKEN


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def _body(self) -> dict:
        n = int(self.headers.get("Content-Length", 0))
        return json.loads(self.rfile.read(n) or b"{}")

    def _send(self, obj: dict, code: int = 200) -> None:
        raw = json.dumps(obj, ensure_ascii=False).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/bank/version":
            b = HOLDER.current
            self._send({"version": b.version, "digest": b.digest})
        elif self.path == "/health":
            b = HOLDER.current
            out: dict = {"ok": True, "sessions": len(SESSIONS),
                         "bank_digest": b.digest, "version": b.version}
            if HOLDER.stale:
                out["bank_stale"] = True
            self._send(out)
        else:
            self._send({"error": "unknown route"}, 404)

    def do_POST(self) -> None:  # noqa: N802
        try:
            self._route()
        except SnapshotError as e:
            self._send({"error": str(e)}, 400)
        except BusyError as e:
            self._send({"error": str(e)}, 503)
        except KeyError as e:
            self._send({"error": str(e)}, 404)
        except ValueError as e:  # bad Content-Length/JSON body → 400, never 500
            self._send({"error": str(e)}, 400)
        except Exception as e:  # noqa: BLE001
            self._send({"error": f"{type(e).__name__}: {e}"}, 500)

    def _route(self) -> None:
        b = self._body()
        p = self.path
        if p == "/sessions/open":
            if not b.get("session_id"):
                self._send({"error": "session_id required"}, 400)
                return
            try:
                opening = open_session(b["session_id"], b.get("scenario", {}),
                                         expected_digest=b.get("bank_digest", ""))
            except KeyError as e:  # duplicate open → 409, not 404
                self._send({"error": str(e)}, 409)
                return
            self._send({"session_id": b["session_id"], "opening": opening})
        elif p == "/sessions/final":
            self._send(final(b["session_id"], (b.get("text") or "")))
        elif p == "/sessions/close":
            close(b["session_id"])
            self._send({"ok": True})
        elif p == "/scenarios/lint":
            self._send(lint_scenario(b["scenario"]))
        elif p == "/bank/reload":
            if not _check_token(self.headers):
                self._send({"error": "bad service token"}, 401)
                return
            # Fan-out carries the snapshot; no fetch round-trip (spec J).
            threading.Thread(target=_reload_background,
                             args=(b.get("version", ""), b.get("digest", ""),
                                   b.get("slots", {}), b.get("questions", {}),
                                   b.get("ontology")),
                             daemon=True).start()
            self._send({"ok": True}, 202)
        else:
            self._send({"error": "unknown route"}, 404)


def _reload_background(version: str, digest: str = "",
                       slots: dict | None = None, questions: dict | None = None,
                       ontology: dict | None = None) -> None:
    """Background rebuild + atomic swap; in-flight calls finish on old (AD-8).

    Bank+ontology swap together under one guard; a bad ontology payload
    keeps the old pair and counts a failure. Absent ontology keeps current
    (old traineebox compat)."""
    global ONTO
    try:
        if slots:
            bank = Bank.build(version or "v1", slots, questions or {})
            if digest and digest != bank.digest:
                raise ValueError(f"reload digest mismatch: {digest} != {bank.digest}")
        else:
            fetched_slots, fetched_questions = _fetch_bank(version)
            bank = Bank.build(version or "v1", fetched_slots, fetched_questions)
        new_onto = ontology_from_snapshot(ontology) if ontology is not None else None
        with _SWAP_GUARD:
            HOLDER.swap(bank)
            if new_onto is not None:
                ONTO = new_onto
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


def audiosocket_listener(host: str = "127.0.0.1", port: int = 9001) -> None:
    """AudioSocket routing by first-frame UUID (call_id). No session → close (AD-6)."""
    import socket as _socket

    from dialog.core.media.asterisk import (
        _KIND_AUDIO, _KIND_ERROR, _KIND_HANGUP, _KIND_UUID, Call, MediaConfig,
        pack_audio, unpack,
    )
    from dialog.core.media.gigaam_stt import GigaAMSTT

    tts = None
    if os.environ.get("DIALOG_PROD") == "1":
        from dialog.core.media.silero_tts import SileroTTS
        tts = SileroTTS(device="cuda")
    srv = _socket.socket(_socket.AF_INET, _socket.SOCK_STREAM)
    srv.setsockopt(_socket.SOL_SOCKET, _socket.SO_REUSEADDR, 1)
    srv.bind((host, port))
    srv.settimeout(2.0)  # bounded accept; accepted sockets stay blocking
    srv.listen(8)
    print(f"audiosocket :{port}", flush=True)

    def pump(conn: _socket.socket) -> None:
        import audioop
        import wave
        rf = conn.makefile("rb")
        try:
            kind, data = unpack(rf)
        except Exception:  # noqa: BLE001
            conn.close()
            return
        if kind != _KIND_UUID:
            conn.close()  # no UUID first frame → close, no session created
            return
        try:
            # 16 raw bytes (hex, no dashes) or hyphenated text → one canon (spec A)
            sid = norm_sid(data.hex() if len(data) == 16 else data.decode(errors="ignore").strip())
        except ValueError:
            conn.close()  # empty/garbage call_id → hangup guard, never a phantom session
            return
        try:
            e = entry(sid)
        except KeyError:
            conn.close()  # unknown UUID → close (AD-6)
            return
        sess = e["session"]
        elock = e["lock"]
        out = bytearray()
        lock = threading.Lock()
        done = threading.Event()

        def send(pcm: bytes) -> None:
            with lock:
                out.extend(pcm)

        cfg = MediaConfig(log_root=WAV_ROOT)
        kind_stt = GigaAMSTT() if os.environ.get("DIALOG_PROD") == "1" else _SilentSTT()
        call = Call(sess, kind_stt, send, cfg, tts=tts)
        stats = {"rx": 0, "tx": 0, "rx_at": time.monotonic(), "send_ms": 0.0}

        def writer() -> None:
            nxt = time.monotonic()
            busy = False
            while not done.is_set():
                with lock:
                    chunk = bytes(out[:320])
                    del out[:320]
                    if chunk:
                        busy = True
                        call.playing = b""
                    elif busy:
                        busy = False
                        call.playing = None
                if chunk:
                    t_send = time.monotonic()
                    try:
                        conn.sendall(pack_audio(chunk))
                    except OSError:
                        done.set()  # writer exit wakes monitor/join, no leak (spec O)
                        return
                    stats["tx"] += 1
                    stats["send_ms"] = max(stats["send_ms"], (time.monotonic() - t_send) * 1000)
                nxt += 0.02
                lag = nxt - time.monotonic()
                if lag > 0:
                    time.sleep(lag)
                elif lag < -0.2:
                    nxt = time.monotonic()

        def monitor() -> None:
            last_rx = last_tx = 0
            while not done.wait(2.0):
                gap = time.monotonic() - stats["rx_at"]
                if gap > 1.0 or stats["send_ms"] > 100:
                    with lock:
                        q = len(out) // 16
                    print(f"[link] rx+{stats['rx'] - last_rx} tx+{stats['tx'] - last_tx} "
                          f"нет входящих {gap:.1f} с, очередь {q} мс, "
                          f"макс. sendall {stats['send_ms']:.0f} мс", flush=True)
                last_rx, last_tx = stats["rx"], stats["tx"]
                stats["send_ms"] = 0.0

        wt = threading.Thread(target=writer, daemon=True)
        wt.start()
        threading.Thread(target=monitor, daemon=True).start()
        mic = bytearray()
        try:
            while True:
                fk, data = unpack(rf)
                if fk in (_KIND_HANGUP, _KIND_ERROR):
                    return
                if fk != _KIND_AUDIO:
                    continue
                mic += data
                stats["rx"] += 1
                stats["rx_at"] = time.monotonic()
                with elock:
                    call.on_mic(data)
        finally:
            call.flush_log()
            if call.log_dir is not None and mic:
                with wave.open(str(call.log_dir / "mic.wav"), "wb") as w:
                    w.setnchannels(1)
                    w.setsampwidth(2)
                    w.setframerate(8000)
                    w.writeframes(bytes(mic))
                _rotate_wav()
            done.set()
            wt.join(timeout=1)
            rf.close()
            conn.close()
            try:
                close(sid)
            except KeyError:
                pass  # HTTP close already reaped the session

    while True:
        try:
            conn, _ = srv.accept()
        except _socket.timeout:
            continue
        threading.Thread(target=pump, args=(conn,), daemon=True).start()


def _rotate_wav() -> None:
    """Wav stays on dialog volume with rotation (no recording_path in schema)."""
    try:
        dirs = sorted(WAV_ROOT.iterdir(), key=lambda p: p.stat().st_mtime)
        for old in dirs[:-WAV_KEEP]:
            for f in sorted(old.rglob("*")):
                try:
                    if f.is_file():
                        f.unlink()
                except OSError:
                    pass
            try:
                old.rmdir()
            except OSError:
                pass
    except OSError:
        pass


class _SilentSTT:
    """Non-prod STT: silence. Prod uses GigaAM only (no Stub/vosk/whisper)."""

    def partial(self, pcm16: bytes) -> str:
        return ""

    def final(self) -> str:
        return ""


def main() -> None:
    import argparse
    ap = argparse.ArgumentParser()
    ap.add_argument("--port", type=int, default=_int_env("DIALOG_PORT", 8000))
    ap.add_argument("--no-audiosocket", action="store_true")
    args = ap.parse_args()
    try:
        HOLDER.swap(Bank.build("boot", *_fetch_bank("boot")))
    except Exception as exc:  # noqa: BLE001 — bank fetch never crashes boot (spec O)
        print(f"dialog boot: bank fetch failed ({exc}), empty until /bank/reload", flush=True)
        HOLDER.mark_failure()
    print(f"dialog boot: bank={HOLDER.current.version} digest={HOLDER.current.digest} "
          f"prod={os.environ.get('DIALOG_PROD') == '1'}", flush=True)
    if not args.no_audiosocket:
        threading.Thread(target=audiosocket_listener, daemon=True).start()
    ThreadingHTTPServer(("0.0.0.0", args.port), H).serve_forever()


if __name__ == "__main__":
    main()
