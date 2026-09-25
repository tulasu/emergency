# -*- coding: utf-8 -*-
"""Composition root: wiring, HTTP routing, env, bootstrap.

Frozen core 1:1 (dialog/core, vendored): e5-small-tuned, torch/cuda, GigaAM,
majority-ensemble, IMPROV=1 grounded, dual voters (laya 0.5 + LLM),
Qwen3-4B, voter/improv 3s, k=5, tuned cascade thresholds. This file never
reimplements understanding/policy/render — it only wires sessions, locks,
routing, validation and the traineebox close-callback to the shell modules.

Deliberately absent (spec Never): /sessions/check|partial|cancel, /rtp,
StubSTT, vosk/whisper, thin_rescue, Fact.audio, runtime meta,
recording_path, mode=both, student-dials-PIN, partial-preview.
"""

from __future__ import annotations

import json
import os
import sys
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from types import ModuleType

from .audiosocket import audiosocket_listener, _rotate_wav  # noqa: F401
from .bank_source import Bank, BankHolder  # noqa: F401
from .lifecycle import (  # noqa: F401
    HOLDER, SESSIONS, GUARD, _SWAP_GUARD,
    _LEXICAL_CACHE, _LEXICAL_GUARD, MAX_SESSIONS,
    SERVICE_TOKEN, BusyError,
    norm_sid, open_session, final, close, entry, lint_scenario,
    _reload_background, _fetch_bank, _push_turns,
    _frozen_improviser, _frozen_ensemble,
)
from .scenario import onto, ontology_from_snapshot, scenario_from_snapshot  # noqa: F401
from .validator import SnapshotError  # noqa: F401


# ONTO + TRAINEEBOX_URL are rebound at runtime (reload rebinds ONTO; tests
# assign serve.TRAINEEBOX_URL), so a plain `from .lifecycle import` would
# leave a stale snapshot. Make serve a ModuleType subclass that forwards
# these two names to lifecycle on read AND write.
_LIVE = ("ONTO", "TRAINEEBOX_URL")


class _ServeModule(ModuleType):
    def __getattr__(self, name):
        if name in _LIVE:
            from . import lifecycle
            return getattr(lifecycle, name)
        raise AttributeError(name)

    def __setattr__(self, name, value):
        if name in _LIVE:
            from . import lifecycle
            setattr(lifecycle, name, value)
        else:
            self.__dict__[name] = value


sys.modules[__name__].__class__ = _ServeModule


def _int_env(key: str, fallback: int) -> int:
    """Non-numeric env never crashes boot (spec O)."""
    try:
        return int(os.environ.get(key, fallback))
    except (TypeError, ValueError):
        return fallback


def _audiosocket_addr() -> tuple[str, int]:
    """DIALOG_AUDIOSOCKET=host:port; garbage never crashes boot (spec O)."""
    raw = os.environ.get("DIALOG_AUDIOSOCKET", "127.0.0.1:9001")
    host, _, port = raw.rpartition(":")
    try:
        return (host or "127.0.0.1", int(port))
    except (TypeError, ValueError):
        return ("127.0.0.1", 9001)


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
        # ponytail: bridge compose sets DIALOG_AUDIOSOCKET=0.0.0.0:9001; host-mode keeps the default
        _ah, _ap = _audiosocket_addr()
        threading.Thread(target=audiosocket_listener, kwargs={"host": _ah, "port": _ap}, daemon=True).start()
    ThreadingHTTPServer(("0.0.0.0", args.port), H).serve_forever()


if __name__ == "__main__":
    main()
