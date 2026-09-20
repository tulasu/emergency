# -*- coding: utf-8 -*-
"""Шаг 4. Медиа-тракт: Asterisk AudioSocket <-> Session.

Граница ядра — только on_partial/on_final/cancel. Внутрь Session/Cascade
не лезем. STT — инжектируемый Transcriber (движок выберет прод),
VAD — наивный энергетический (ponytail: порог+тишина калибруются наружу,
Silero/WebRTC — когда этот начнёт рвать фразы на шумах линии).
Пока играет ответ — микрофон игнорируется (half-duplex окно против
самопрерывания; эхоподавление — когда понадобится full-duplex).
"""

from __future__ import annotations

import audioop
import struct
import wave
from dataclasses import dataclass
from pathlib import Path
from typing import Protocol

AUDIO = Path(__file__).resolve().parents[2] / "data" / "audio"

# AudioSocket: 1 байт тип + 2 байта длина (big-endian) + payload
_KIND_AUDIO = 0x00
_KIND_HANGUP = 0x01
_FRAME = 160  # 20 мс SLIN 8 кГц


class Transcriber(Protocol):
    """Стриминговый STT. partial — гипотезы, final — итог фразы."""

    def partial(self, pcm16: bytes) -> str: ...
    def final(self) -> str: ...


@dataclass
class MediaConfig:
    vad_threshold: int = 400  # rms порог голоса, калибровать под линию
    silence_ms: int = 700  # тишины = конец фразы (600–800)
    audio_root: Path = AUDIO


def pack_audio(slin8: bytes) -> bytes:
    return struct.pack(">BH", _KIND_AUDIO, len(slin8)) + slin8


def unpack(stream) -> tuple[int, bytes]:
    """Читает один фрейм AudioSocket. (kind, payload)."""
    hdr = stream.read(3)
    if len(hdr) < 3:
        return _KIND_HANGUP, b""
    kind, ln = struct.unpack(">BH", hdr)
    data = b""
    while len(data) < ln:
        chunk = stream.read(ln - len(data))
        if not chunk:
            break
        data += chunk
    return kind, data


def upsample8to16(slin8: bytes) -> bytes:
    """8 кГц -> 16 кГц для VAD/STT. Линейная интерполяция сойдёт до ~50
    потоков (ponytail: дальше Speex/soxr)."""
    return audioop.ratecv(slin8, 2, 1, 8000, 16000, None)[0]


def load_reply_pcm(audio_id: str, root: Path = AUDIO) -> bytes | None:
    """Композит 'a+b' клеится конкатенацией. Нет файлов — None, плеер
    идёт через живой TTS по Reply.text."""
    out = bytearray()
    for part in audio_id.split("+"):
        p = root / part
        if not p.exists():
            return None
        with wave.open(str(p), "rb") as w:
            assert (w.getframerate(), w.getnchannels()) == (8000, 1), p
            out += w.readframes(w.getnframes())
    return bytes(out)


class Call:
    """Один звонок: фреймы из сокета -> транскрибер -> Session -> звук назад."""

    def __init__(self, session, transcriber: Transcriber,
                 send, cfg: MediaConfig | None = None):
        self.session = session
        self.tr = transcriber
        self.send = send  # send(slin8_bytes)
        self.cfg = cfg or MediaConfig()
        self._silent_ms = 0
        self._heard = False
        self.playing: bytes | None = None

    def on_mic(self, slin8: bytes) -> None:
        if self.playing is not None:
            return  # half-duplex окно: свой playback в VAD не кормим
        pcm16 = upsample8to16(slin8)
        voiced = audioop.rms(pcm16, 2) >= self.cfg.vad_threshold
        # шаг 4: инкремент по длине буфера, а не фикс 20 мс — транспорт
        # может слать чанки любого размера
        ms = len(slin8) // 16  # 8 кГц mono s16: 16 байт = 1 мс
        self._silent_ms = 0 if voiced else self._silent_ms + ms
        if voiced:
            self._heard = True
        hyp = self.tr.partial(pcm16)
        if hyp:
            self.session.on_partial(hyp)
        if self._heard and self._silent_ms >= self.cfg.silence_ms:
            self._heard = False
            self._silent_ms = 0
            self._answer(self.tr.final())

    def _answer(self, text: str) -> None:
        if not text.strip():
            return
        reply = self.session.on_final(text)
        pcm = load_reply_pcm(reply.audio_id, self.cfg.audio_root) \
            if reply.audio_id else None
        if pcm is None:
            self.on_tts_text(reply.text)  # живого TTS нет — хук для шага 6
            return
        self.playing = pcm
        for i in range(0, len(pcm), _FRAME * 2):
            if self.playing is None:
                break  # barge-in оборвал
            self.send(pcm[i:i + _FRAME * 2])
        self.playing = None

    def on_tts_text(self, text: str) -> None:
        """Нет предрендера — переопределить живым TTS в проде."""

    def on_barge_in(self) -> None:
        """Голос во время playback (детектит транспорт): стоп + cancel."""
        if self.playing is not None:
            self.playing = None
            self.session.cancel()
