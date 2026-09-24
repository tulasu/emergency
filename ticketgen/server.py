# -*- coding: utf-8 -*-
"""Pure ticketgen function: prompt -> draft_scenario/dialog_report over HTTP.

No DB, no queue, no ticketgen<->dialog link. Traineebox owns the job rows
(building_dialog/checking_dialog): its worker claims jobs, POSTs here,
lints via dialog, approves via AtomicPublish. Transport pinned: HTTP
POST /draft (spine bet); ticketgen never polls jobs.

Endpoints:
  POST /draft  {"prompt": "..."} -> draft + dialog_report
  GET  /health -> {"ok": true}
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from catalog import load_catalog  # noqa: E402
from llm import LLMClient  # noqa: E402
from pipeline import Pipeline  # noqa: E402

PIPE: Pipeline | None = None
PIPE_LOCK = threading.Lock()
MAX_BODY = 1_000_000


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def _body(self) -> dict:
        try:
            n = int(self.headers.get("Content-Length", 0) or 0)
        except (TypeError, ValueError):
            raise ValueError("bad Content-Length") from None
        if n < 0 or n > MAX_BODY:
            raise ValueError("body too large")
        try:
            return json.loads(self.rfile.read(n) or b"{}")
        except ValueError as e:
            raise ValueError(f"bad JSON: {e}") from None

    def _send(self, obj: dict, code: int = 200) -> None:
        raw = json.dumps(obj, ensure_ascii=False).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/health":
            self._send({"ok": True})
        else:
            self._send({"error": "unknown route"}, 404)

    def do_POST(self) -> None:  # noqa: N802
        try:
            b = self._body()
            if self.path != "/draft":
                self._send({"error": "unknown route"}, 404)
                return
            prompt = (b.get("prompt") or "").strip()
            if not prompt:
                self._send({"error": "prompt required"}, 400)
                return
            assert PIPE is not None
            with PIPE_LOCK:
                result = PIPE.draft_scenario(prompt)
                report = PIPE.dialog_report(result)
            self._send({
                "draft_title": result.draft_title,
                "scenario_text": result.scenario_text,
                "draft_reference": result.draft_reference,
                "dialog_report": report,
            })
        except ValueError as e:  # bad body/prompt → 400, never 500
            self._send({"error": str(e)}, 400)
        except Exception as e:  # noqa: BLE001 — pure function never crashes the socket
            self._send({"error": f"{type(e).__name__}: {e}"}, 500)


def main() -> None:
    global PIPE
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--catalog", default=os.environ.get("TICKETGEN_CATALOG_PATH", "/catalog"),
                    help="catalog dir (or $TICKETGEN_CATALOG_PATH)")
    ap.add_argument("--port", type=int, default=int(os.environ.get("TICKETGEN_PORT", "8001")))
    args = ap.parse_args()
    PIPE = Pipeline(load_catalog(Path(args.catalog)), LLMClient())
    print(f"ticketgen serve :{args.port} catalog={args.catalog}", flush=True)
    ThreadingHTTPServer(("0.0.0.0", args.port), H).serve_forever()


if __name__ == "__main__":
    main()
