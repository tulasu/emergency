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
import json
import struct
import time
import wave
from dataclasses import dataclass
from pathlib import Path
from typing import Protocol

AUDIO = Path(__file__).resolve().parents[2] / "data" / "audio"

# AudioSocket: 1 байт тип + 2 байта длина (big-endian) + payload.
# Типы — из app_audiosocket: 0x00 hangup, 0x01 UUID (первый фрейм),
# 0x03 DTMF, 0x10 SLIN-аудио, 0xff ошибка.
_KIND_HANGUP = 0x00
_KIND_UUID = 0x01
_KIND_DTMF = 0x03
_KIND_AUDIO = 0x10
_KIND_ERROR = 0xFF
_FRAME = 160  # 20 мс SLIN 8 кГц
_PREROLL = 4800  # 300 мс SLIN 8 кГц


class Transcriber(Protocol):
    """Стриминговый STT. partial — гипотезы, final — итог фразы."""

    def partial(self, pcm16: bytes) -> str: ...
    def final(self) -> str: ...


@dataclass
class MediaConfig:
    vad_threshold: int = 400  # rms порог голоса, калибровать под линию
    silence_ms: int = 700  # тишины = конец фразы (600–800)
    audio_root: Path = AUDIO
    log_root: Path | None = None  # лог фраз оператора: wav + turns.jsonl


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
                 send, cfg: MediaConfig | None = None, tts=None):
        self.session = session
        self.tr = transcriber
        self.send = send  # send(slin8_bytes)
        self.cfg = cfg or MediaConfig()
        self.tts = tts  # tts(text) -> slin8: ответы без предрендера (LLM вне сценария)
        self._silent_ms = 0
        self._heard = False
        self.playing: bytes | None = None
        self._utt = bytearray()  # звук текущей фразы оператора — в лог
        self._pre = bytearray()  # ~300 мс до срабатывания VAD, чтобы не резать начало
        self._n = 0
        self.log_dir: Path | None = None
        if self.cfg.log_root is not None:
            stamp = time.strftime("%Y%m%d-%H%M%S")
            sc = getattr(getattr(session, "scenario", None), "id", "") or "call"
            self.log_dir = self.cfg.log_root / f"{stamp}_{sc}"
            self.log_dir.mkdir(parents=True, exist_ok=True)

    def on_mic(self, slin8: bytes) -> None:
        if self.playing is not None:
            return  # half-duplex окно: свой playback в VAD не кормим
        pcm16 = upsample8to16(slin8)
        voiced = audioop.rms(pcm16, 2) >= self.cfg.vad_threshold
        # шаг 4: инкремент по длине буфера, а не фикс 20 мс — транспорт
        # может слать чанки любого размера
        ms = len(slin8) // 16  # 8 кГц mono s16: 16 байт = 1 мс
        self._silent_ms = 0 if voiced else self._silent_ms + ms
        if voiced and not self._heard:
            self._utt = bytearray(self._pre)
        if voiced:
            self._heard = True
        if self._heard:
            self._utt += slin8
        else:
            self._pre = (self._pre + slin8)[-_PREROLL:]
        hyp = self.tr.partial(pcm16)
        if hyp:
            self.session.on_partial(hyp)
        if self._heard and self._silent_ms >= self.cfg.silence_ms:
            self._heard = False
            self._silent_ms = 0
            utt, self._utt = bytes(self._utt), bytearray()
            self._answer(self.tr.final(), utt)

    def _answer(self, text: str, utt: bytes = b"") -> None:
        if not text.strip():
            self._log(text, utt, None)  # голос был, STT не разобрал — важно для дообучения
            return
        reply = self.session.on_final(text)
        self._log(text, utt, getattr(self.session, "turns", [None])[-1])
        pcm = load_reply_pcm(reply.audio_id, self.cfg.audio_root) \
            if reply.audio_id else None
        if pcm is None and self.tts is not None and reply.text:
            pcm = self.tts(reply.text)
        if pcm is None:
            self.on_tts_text(reply.text)  # живого TTS нет — хук для шага 6
            return
        self.playing = pcm
        for i in range(0, len(pcm), _FRAME * 2):
            if self.playing is None:
                break  # barge-in оборвал
            self.send(pcm[i:i + _FRAME * 2])
        self.playing = None

    def flush_log(self) -> None:
        """Звонок оборвался посреди фразы — сохранить недоговорённое."""
        if self._heard and self._utt:
            self._log("<оборвано> " + self.tr.final(), bytes(self._utt), None)

    def _log(self, text: str, utt: bytes, turn) -> None:
        if self.log_dir is None:
            return
        self._n += 1
        wav_name = f"{self._n:03d}.wav"
        with wave.open(str(self.log_dir / wav_name), "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(2)
            w.setframerate(8000)
            w.writeframes(utt)
        rec = {"n": self._n, "ts": time.strftime("%Y-%m-%dT%H:%M:%S"),
               "wav": wav_name, "dur_s": round(len(utt) / 16000, 2),
               "stt": text, "ref": ""}  # ref — сюда руками правильный текст
        if turn is not None:
            u, d, r = turn.understanding, turn.decision, turn.reply
            rec.update(
                slots=u.slots, keys=u.keys, act=str(u.act),
                score=round(u.score, 3), source=u.source,
                reveal=d.reveal, style=str(d.style),
                reply=r.text, audio_id=r.audio_id,
                generated=r.audio_id is None and bool(r.text),
                ms=round(u.latency_ms),
            )
            votes = getattr(getattr(self.session, "cascade", None), "last_votes", None)
            if votes:
                rec["votes"] = votes
        with open(self.log_dir / "turns.jsonl", "a", encoding="utf-8") as f:
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")
        print(f"[turn {self._n}] stt={text!r} -> "
              f"{rec.get('keys')} {rec.get('source', '')} {rec.get('votes', '')} "
              f"{'[LLM] ' if rec.get('generated') else ''}{rec.get('reply', '')!r}",
              flush=True)

    def on_tts_text(self, text: str) -> None:
        """Нет предрендера — переопределить живым TTS в проде."""

    def on_barge_in(self) -> None:
        """Голос во время playback (детектит транспорт): стоп + cancel."""
        if self.playing is not None:
            self.playing = None
            self.session.cancel()
