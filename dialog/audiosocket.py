# -*- coding: utf-8 -*-
"""AudioSocket media transport — per-call socket, writer thread, monitor thread, mic recording."""

from __future__ import annotations

import os
import shutil
import socket
import threading
import time
import wave

from dialog.core.media.asterisk import (
    _KIND_AUDIO, _KIND_ERROR, _KIND_HANGUP, _KIND_UUID, Call, MediaConfig,
    pack_audio, unpack,
)
from dialog.core.media.gigaam_stt import GigaAMSTT

from .lifecycle import WAV_KEEP, WAV_ROOT, close, entry, norm_sid


class _SilentSTT:
    """Non-prod STT: silence. Prod uses GigaAM only (no Stub/vosk/whisper)."""

    def partial(self, pcm16: bytes) -> str:
        return ""

    def final(self) -> str:
        return ""


def audiosocket_listener(host: str = "127.0.0.1", port: int = 9001) -> None:
    """AudioSocket routing by first-frame UUID (call_id). No session → close (AD-6)."""
    tts = None
    if os.environ.get("DIALOG_PROD") == "1":
        from dialog.core.media.silero_tts import SileroTTS
        tts = SileroTTS(device="cuda")
    srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind((host, port))
    srv.settimeout(2.0)  # bounded accept; accepted sockets stay blocking
    srv.listen(8)
    print(f"audiosocket :{port}", flush=True)
    while True:
        try:
            conn, _ = srv.accept()
        except socket.timeout:
            continue
        threading.Thread(target=_pump_one_call, args=(conn, tts), daemon=True).start()


def _pump_one_call(conn: socket.socket, tts) -> None:
    """One connection = one call: route by UUID, pump audio, teardown on hangup."""
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

    wt = threading.Thread(target=_writer_thread, args=(call, conn, out, done, stats, lock),
                          daemon=True)
    wt.start()
    threading.Thread(target=_monitor_thread, args=(stats, out, done, lock), daemon=True).start()
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
            _save_mic_wav(call, mic)
            _rotate_wav()
        done.set()
        wt.join(timeout=1)
        rf.close()
        conn.close()
        try:
            close(sid)
        except KeyError:
            pass  # HTTP close already reaped the session


def _writer_thread(call: Call, conn: socket.socket, out: bytearray,
                   done: threading.Event, stats: dict, lock: threading.Lock) -> None:
    """20 ms tick, 320-byte chunks. Drives the half-duplex window (see below)."""
    nxt = time.monotonic()
    busy = False
    while not done.is_set():
        with lock:
            chunk = bytes(out[:320])
            del out[:320]
            # Half-duplex window is load-bearing: while audio is queued, set
            # call.playing = b"" (truthy) so Call.on_mic drops incoming mic
            # frames — self-playback must not feed the VAD. Clear to None on
            # idle so the next operator phrase is heard again. Don't "simplify".
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


def _monitor_thread(stats: dict, out: bytearray, done: threading.Event,
                    lock: threading.Lock) -> None:
    """2-second tick diagnostic: rx/tx delta, silence gap, queue, max sendall."""
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


def _save_mic_wav(call: Call, mic: bytearray) -> None:
    with wave.open(str(call.log_dir / "mic.wav"), "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(8000)
        w.writeframes(bytes(mic))


def _rotate_wav() -> None:
    """Wav stays on dialog volume with rotation (no recording_path in schema)."""
    try:
        dirs = sorted(WAV_ROOT.iterdir(), key=lambda p: p.stat().st_mtime)
    except OSError:
        return
    # ponytail: global scan + rmtree; fine at DIALOG_WAV_KEEP=100
    for old in dirs[:-WAV_KEEP]:
        shutil.rmtree(old, ignore_errors=True)
