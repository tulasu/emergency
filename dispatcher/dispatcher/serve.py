# -*- coding: utf-8 -*-
"""Шаг 6. Тонкий HTTP-сервер над Service (шаг 1) + RTP-приёмник для Go (шаг 5).

Без новых зависимостей: stdlib http.server + Threading. gRPC — когда
упрёмся в HTTP-оверхед (ponytail: один /rtp чанк ~20 мс, JSON хватает).
STT-движок инжектится одной функцией ( transcribe_final(pcm16) ), по
умолчанию заглушка — текстовый путь /sessions/* работает полностью,
аудиопуть ждёт движок.

Прод: --model e5-small-tuned --backend torch --device cuda --arbiter laya
--thin-rescue (см. шаг 7 про GPU-пул).
"""

from __future__ import annotations

import argparse
import audioop
import base64
import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from .media.asterisk import Call, MediaConfig
from .service import Service, reply_to_dict

SVC: Service = None  # type: ignore
LOCK = threading.Lock()
RTPS: dict = {}  # session_id -> {"call": Call, "out": bytearray, "stt": ...}


class StubSTT:
    """Заглушка: гипотез нет, финалы придут из /sessions/final напрямую."""

    def partial(self, pcm16: bytes) -> str:
        return ""

    def final(self) -> str:
        return ""


def _rtp_state(session_id: str, scenario_id: str = "") -> dict:
    st = RTPS.get(session_id)
    if st is None:
        if scenario_id:
            SVC.open(scenario_id, session_id)
        sess = SVC.sessions[session_id]  # KeyError -> 404 наружу
        out: bytearray = bytearray()
        st = {
            "call": Call(sess, StubSTT(), out.extend,
                         MediaConfig(audio_root=MediaConfig().audio_root)),
            "out": out,
        }
        RTPS[session_id] = st
    return st


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

    def do_POST(self) -> None:  # noqa: N802
        try:
            with LOCK:
                self._route()
        except KeyError as e:
            self._send({"error": str(e)}, 404)
        except Exception as e:  # транспорт не должен ронять звонки текстом
            self._send({"error": f"{type(e).__name__}: {e}"}, 500)

    def _route(self) -> None:
        b = self._body()
        p = self.path
        if p == "/sessions/open":
            sid, opening = SVC.open(b["scenario_id"], b.get("session_id"))
            self._send({"session_id": sid, "opening": opening})
        elif p == "/sessions/check":
            if b.get("scenario_id") not in SVC.loaded.scenarios:
                self._send({"error": "unknown scenario"}, 404)
            else:
                self._send({"ok": True})
        elif p == "/sessions/partial":
            SVC.partial(b["session_id"], b.get("text", ""))
            self._send({"ok": True})
        elif p == "/sessions/final":
            self._send(reply_to_dict(SVC.final(b["session_id"], b.get("text", ""))))
        elif p == "/sessions/cancel":
            self._send({"stop_audio_id": SVC.cancel(b["session_id"])})
        elif p == "/sessions/close":
            SVC.close(b["session_id"])
            RTPS.pop(b["session_id"], None)
            self._send({"ok": True})
        elif p == "/rtp":
            # alaw 8к от Asterisk -> linear -> Call (VAD/endpoint/playback)
            raw = base64.b64decode(b.get("audio_b64", ""))
            slin = audioop.alaw2lin(raw, 2) if raw else b""
            st = _rtp_state(b["session_id"], b.get("scenario_id", ""))
            if slin:
                st["call"].on_mic(slin)
            chunk = bytes(st["out"][:320])
            del st["out"][:320]
            self._send({"audio_b64": base64.b64encode(chunk).decode()})
        elif p == "/health":
            self._send({"ok": True,
                         "scenarios": len(SVC.loaded.scenarios),
                         "sessions": len(SVC.sessions)})
        else:
            self._send({"error": "unknown route"}, 404)


def main() -> None:
    global SVC
    ap = argparse.ArgumentParser()
    ap.add_argument("--port", type=int, default=8000)
    ap.add_argument("--model", default=None)
    ap.add_argument("--backend", default="auto")
    ap.add_argument("--device", default=None)
    ap.add_argument("--arbiter", default=None)
    ap.add_argument("--thin-rescue", action="store_true")
    ap.add_argument("--require-cuda", action="store_true",
                    help="fail-fast без CUDA: молчаливый CPU-фолбэк в проде запрещён")
    args = ap.parse_args()
    if args.require_cuda:
        from .nlu import encoder as enc

        if enc.available_device() != "cuda":
            raise SystemExit("нет CUDA — прод-воркер не стартует (--require-cuda)")
    SVC = Service(model=args.model, backend=args.backend, device=args.device,
                  arbiter_kind=args.arbiter, thin_rescue=args.thin_rescue)
    print(f"слушаю :{args.port}, сценариев: {len(SVC.loaded.scenarios)}", flush=True)
    ThreadingHTTPServer(("0.0.0.0", args.port), H).serve_forever()


if __name__ == "__main__":
    main()
