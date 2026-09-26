# -*- coding: utf-8 -*-
"""audio-synth: stateless Silero waveform daemon (spec 06).

Boundary: pure GPU function, no DB/S3/queue. Loads Silero v4_ru and
the Russian neural Piper fallback at startup. One _lock per process,
speakable digits, 48k→8k resample for Silero, 22.05k→8k for Piper.

    POST /synth {text, voice} -> audio/wav (8kHz mono s16)
    GET  /health -> {"ok": true}

SYNTH_DEVICE=cuda|cpu. Voice allowlist pins the texthash contract:
unknown voices are 400, never silent substitution.
"""

from __future__ import annotations

import io
import json
import os
import subprocess
import threading
import wave
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from speakable import speakable

VOICES = ("aidar", "baya", "kseniya", "xenia", "eugene")
DEFAULT_VOICE = os.environ.get("SYNTH_VOICE", "kseniya")
DEVICE = os.environ.get("SYNTH_DEVICE", "cuda")
PIPER_MODEL_PATH = os.environ.get(
    "PIPER_MODEL_PATH", "/opt/piper/ru_RU-denis-medium.onnx")
SILERO_RATE = 48000
OUT_RATE = 8000
MAX_TEXT = 2000

_lock = threading.Lock()
_model = None
_torch = None
_piper = None


def _device():
    import torch

    return torch.device(DEVICE if torch.cuda.is_available() else "cpu")


def load() -> None:
    """Load Silero once + warmup; called at boot, not per request."""
    global _model, _torch, _piper
    import torch
    from piper import PiperVoice

    _piper = PiperVoice.load(PIPER_MODEL_PATH)
    _torch = torch
    _model, _ = torch.hub.load(
        repo_or_dir="snakers4/silero-models", model="silero_tts",
        language="ru", speaker="v4_ru", trust_repo=True,
    )
    _model.to(_device())
    for text in ("Алло.", "Да, поняла вас.",
                 "Не знаю, не видела, я там не была.",
                 "Москва, улица Красный Казанец, дом девятнадцать Б, на стоянке.") * 2:
        synth_wav(text, DEFAULT_VOICE)


def synth_wav(text: str, voice: str) -> bytes:
    """Synthesize 8kHz mono s16 wav bytes. Thread-safe via _lock."""
    import numpy as np
    import torchaudio.functional as AF

    with _lock, _torch.inference_mode():
        audio = _model.apply_tts(text=speakable(text), speaker=voice,
                                 sample_rate=SILERO_RATE,
                                 put_accent=True, put_yo=True)
        narrow = AF.resample(audio, SILERO_RATE, OUT_RATE).numpy()
    pcm = (narrow * 32767).clip(-32768, 32767).astype("<i2").tobytes()
    buf = io.BytesIO()
    with wave.open(buf, "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(OUT_RATE)
        w.writeframes(pcm)
    return buf.getvalue()


def piper_wav(text: str) -> bytes:
    """Synthesize Russian speech with the locally loaded neural fallback."""
    native = io.BytesIO()
    with _lock, wave.open(native, "wb") as w:
        _piper.synthesize_wav(speakable(text), w)
    out = subprocess.run(
        ["ffmpeg", "-v", "error", "-i", "pipe:0",
         "-ar", str(OUT_RATE), "-ac", "1", "-c:a", "pcm_s16le",
         "-f", "s16le", "pipe:1"],
        input=native.getvalue(), stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        check=False,
    )
    if out.returncode != 0 or not out.stdout:
        raise RuntimeError("Piper fallback failed")
    buf = io.BytesIO()
    with wave.open(buf, "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(OUT_RATE)
        w.writeframes(out.stdout)
    return buf.getvalue()


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def _send(self, body: bytes, code: int, ctype: str):
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/health":
            if _model is None:
                self._send(b'{"ok": false}', 503, "application/json")
                return
            self._send(b'{"ok": true}', 200, "application/json")
            return
        self._send(b'{"error": "unknown route"}', 404, "application/json")

    def do_POST(self):
        if self.path != "/synth":
            self._send(b'{"error": "unknown route"}', 404, "application/json")
            return
        try:
            length = int(self.headers.get("Content-Length") or 0)
        except ValueError:
            length = 0
        if length <= 0 or length > 65536:
            self._send(b'{"error": "bad body"}', 400, "application/json")
            return
        try:
            req = json.loads(self.rfile.read(length))
        except (ValueError, OSError):
            self._send(b'{"error": "bad json"}', 400, "application/json")
            return
        text = req.get("text") if isinstance(req, dict) else None
        voice = req.get("voice", DEFAULT_VOICE) if isinstance(req, dict) else DEFAULT_VOICE
        if not isinstance(text, str) or not text.strip():
            self._send(b'{"error": "text required"}', 400, "application/json")
            return
        if len(text) > MAX_TEXT:
            self._send(b'{"error": "text too long"}', 400, "application/json")
            return
        if voice not in VOICES:
            self._send(b'{"error": "bad voice"}', 400, "application/json")
            return
        try:
            try:
                body = synth_wav(text, voice)
            except Exception:  # noqa: BLE001 — silero рвётся на экзотике
                body = piper_wav(text)
        except Exception as e:  # noqa: BLE001 — 500 с текстом без PII
            err = json.dumps({"error": f"synth failed: {str(e)[:100]}"}).encode()
            self._send(err, 500, "application/json")
            return
        self._send(body, 200, "audio/wav")


def main() -> None:
    port = int(os.environ.get("SYNTH_PORT", "8003"))
    load()
    ThreadingHTTPServer(("0.0.0.0", port), H).serve_forever()


if __name__ == "__main__":
    main()
