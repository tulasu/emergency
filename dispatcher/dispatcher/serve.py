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
import faulthandler
import json
import signal
import threading
import time
import wave
from pathlib import Path
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


def _make_stt(kind: str | None):
    """Собрать STT по имени. None/'' — StubSTT."""
    if kind == "vosk":
        from .media.vosk_stt import VoskSTT
        print("поднимаю vosk small-ru (~45 МБ, CPU)...", flush=True)
        return VoskSTT()
    if kind == "gigaam":
        from .media.gigaam_stt import GigaAMSTT
        print("поднимаю GigaAM-v2 CTC onnx (~0.9 ГБ, CPU)...", flush=True)
        return GigaAMSTT()
    if kind == "whisper":
        from .media.whisper_stt import WhisperSTT
        print("поднимаю whisper-tiny (~150 МБ), это до минуты...", flush=True)
        return WhisperSTT(model="tiny", language="ru")
    return StubSTT()


_AUDIOSOCKET_HOST = "127.0.0.1"
_AUDIOSOCKET_PORT = 9001
# лог фраз оператора по звонкам: wav + turns.jsonl — датасет под дообучение STT/NLU
_CALLS_LOG = Path(__file__).resolve().parents[1] / "data" / "calls"


def _audiosocket_listener(stt_kind: str | None, tts=None) -> None:
    """AudioSocket из asterisk: TCP-сокет, 1+3+payload фреймы (SLIN 16 бит).

    Каждый звонок — одно соединение, в нём живёт Call. После зависания
    сокета сессия закрывается."""
    import socket as _socket

    from .media.asterisk import (
        _KIND_AUDIO, _KIND_ERROR, _KIND_HANGUP, Call, MediaConfig,
        pack_audio, unpack,
    )

    kind = _make_stt(stt_kind)
    srv = _socket.socket(_socket.AF_INET, _socket.SOCK_STREAM)
    srv.setsockopt(_socket.SOL_SOCKET, _socket.SO_REUSEADDR, 1)
    srv.bind((_AUDIOSOCKET_HOST, _AUDIOSOCKET_PORT))
    srv.listen(8)
    print(f"audiosocket :{_AUDIOSOCKET_PORT}", flush=True)

    def pump(conn: _socket.socket) -> None:
        # Подхватываем последнюю открытую сессию — её положил orchestrator
        # через /sessions/open до originate. Для прод-варианта нужен
        # routing по call_id из channel vars, это шаг 7+.
        if not SVC.sessions:
            conn.close()
            return
        sid = next(reversed(SVC.sessions))
        sess = SVC.sessions[sid]
        out = bytearray()
        lock = threading.Lock()
        done = threading.Event()

        def send(pcm: bytes) -> None:
            with lock:
                out.extend(pcm)

        call = Call(sess, kind, send, MediaConfig(log_root=_CALLS_LOG), tts=tts)
        # счётчики тракта: пишутся потоками чтения/отправки, читаются сторожем
        stats = {"rx": 0, "tx": 0, "rx_at": time.monotonic(), "send_ms": 0.0}

        def writer() -> None:
            # Отдельный поток со своими часами: 320 байт (20 мс) строго раз
            # в 20 мс. Раньше ответ уходил по фрейму на каждый входящий, и
            # пока STT думал над фразой, в звуке были дыры, потом рывок пачкой.
            nxt = time.monotonic()
            busy = False
            while not done.is_set():
                with lock:
                    chunk = bytes(out[:320])
                    del out[:320]
                    # half-duplex: пока есть что играть, микрофон в STT не идёт.
                    # Сбрасываем только на переходе «доиграли» — иначе затрём
                    # playing, который Call._answer выставил до первого send.
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
                        return
                    stats["tx"] += 1
                    stats["send_ms"] = max(stats["send_ms"],
                                           (time.monotonic() - t_send) * 1000)
                nxt += 0.02
                lag = nxt - time.monotonic()
                if lag > 0:
                    time.sleep(lag)
                elif lag < -0.2:  # проспали (GC, нагрузка) — не догоняем пачкой
                    nxt = time.monotonic()

        def monitor() -> None:
            # независимый от чтения взгляд на тракт: если фреймы от Asterisk
            # перестали приходить, поток чтения молчит — а этот нет
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
        rf = conn.makefile("rb")
        mic = bytearray()  # весь входящий звук звонка — mic.wav для разбора
        frames, rms_sum = 0, 0
        try:
            while True:
                fk, data = unpack(rf)
                if fk in (_KIND_HANGUP, _KIND_ERROR):
                    return
                if fk != _KIND_AUDIO:  # UUID первым фреймом, DTMF — пока мимо
                    continue
                mic += data
                frames += 1
                stats["rx"] += 1
                stats["rx_at"] = time.monotonic()
                rms_sum += audioop.rms(data, 2)
                if frames % 250 == 0:  # раз в 5 с — состояние тракта
                    print(f"[mic] rms~{rms_sum // 250} playing={call.playing is not None} "
                          f"heard={call._heard} silent_ms={call._silent_ms} "
                          f"queue_ms={len(out) // 16}", flush=True)
                    rms_sum = 0
                # сторож: фрейм — это 20 мс, а тут STT, голосование, LLM и TTS.
                # Застряли дольше 3 с — стеки всех потоков в лог, чтобы
                # увидеть, кто держит (звонок при этом «глохнет»)
                faulthandler.dump_traceback_later(3, exit=False)
                call.on_mic(data)  # SLIN 8 кГц, 320 байт = 20 мс
                faulthandler.cancel_dump_traceback_later()
        finally:
            faulthandler.cancel_dump_traceback_later()
            call.flush_log()
            if call.log_dir is not None and mic:
                with wave.open(str(call.log_dir / "mic.wav"), "wb") as w:
                    w.setnchannels(1)
                    w.setsampwidth(2)
                    w.setframerate(8000)
                    w.writeframes(bytes(mic))
            done.set()
            wt.join(timeout=1)
            rf.close()
            conn.close()
            SVC.close(sid)

    while True:
        conn, _addr = srv.accept()
        threading.Thread(target=pump, args=(conn,), daemon=True).start()


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
    ap.add_argument("--stt", default=None, choices=["gigaam", "vosk", "whisper"],
                    help="движок STT: gigaam | vosk | whisper | пусто = Stub")
    ap.add_argument("--ensemble", default=None,
                    choices=["majority", "llm-lead", "cascade+", "strict"],
                    help="голосование e5 + laya + LLM (нужен tools/llama_server.sh)")
    ap.add_argument("--improv", action="store_true",
                    help="ответы вне сценария от LLM, озвучка Silero на лету")
    ap.add_argument("--no-audiosocket", action="store_true",
                    help="не поднимать TCP-листенер AudioSocket (только HTTP)")
    args = ap.parse_args()
    # kill -USR1 <pid> — стеки всех потоков в stderr (docker logs)
    faulthandler.register(signal.SIGUSR1, all_threads=True)
    if args.require_cuda:
        from .nlu import encoder as enc

        if enc.available_device() != "cuda":
            raise SystemExit("нет CUDA — прод-воркер не стартует (--require-cuda)")
    SVC = Service(model=args.model, backend=args.backend, device=args.device,
                  arbiter_kind=args.arbiter, thin_rescue=args.thin_rescue,
                  ensemble=args.ensemble, improv=args.improv)
    tts = None
    if args.improv:
        from .media.silero_tts import SileroTTS

        print("поднимаю Silero для живых ответов...", flush=True)
        tts = SileroTTS(device=args.device or "cpu")
    print(f"слушаю :{args.port}, сценариев: {len(SVC.loaded.scenarios)}, "
          f"понимание: {args.model or 'лексика'}"
          f"{' + голосование ' + args.ensemble if args.ensemble else ''}"
          f"{' + импровизация' if args.improv else ''}", flush=True)
    if not args.no_audiosocket:
        threading.Thread(target=_audiosocket_listener,
                         args=(args.stt, tts), daemon=True).start()
    ThreadingHTTPServer(("0.0.0.0", args.port), H).serve_forever()


if __name__ == "__main__":
    main()
