# -*- coding: utf-8 -*-
"""Шаг 4: endpointing по тишине, проигрывание, barge-in — без Asterisk."""

import io
import struct
import wave

from dispatcher.media.asterisk import (
    Call, MediaConfig, load_reply_pcm, pack_audio, unpack, upsample8to16,
)


def tone(ms: int, loud: bool) -> bytes:
    import audioop
    import math
    n = 8000 * ms // 1000
    amp = 8000 if loud else 50
    pcm = struct.pack(f"<{n}h", *[int(amp * math.sin(i * 0.3)) for i in range(n)])
    return audioop.tomono(pcm, 2, 1, 1) if False else pcm


class FakeSTT:
    def __init__(self):
        self.phrase = "на каком этаже горит"
        self.calls = 0

    def partial(self, pcm16: bytes) -> str:
        self.calls += 1
        return self.phrase[: self.calls]

    def final(self) -> str:
        return self.phrase


class StubSession:
    def __init__(self):
        self.partials: list[str] = []
        self.finals: list[str] = []
        self.cancelled = 0

    def on_partial(self, t: str) -> None:
        self.partials.append(t)

    def on_final(self, t: str):
        self.finals.append(t)
        from dispatcher.types import Reply
        return Reply("На тринадцатом.", "a/x.wav")

    def cancel(self):
        self.cancelled += 1


def test_framing_roundtrip():
    assert unpack(io.BytesIO(pack_audio(b"abc"))) == (0x00, b"abc")
    assert unpack(io.BytesIO(b""))[0] == 0x01  # обрыв = hangup


def test_resample_doubles():
    out = upsample8to16(b"\x00\00" * 160)
    assert 630 <= len(out) <= 650  # ratecv-фильтр: ±пары семплов


def test_silence_endpoints_phrase(tmp_path):
    sess, sent = StubSession(), []
    cfg = MediaConfig(vad_threshold=400, silence_ms=100, audio_root=tmp_path)
    # reply без файлов -> TTS-хук, без отправки
    c = Call(sess, FakeSTT(), sent.append, cfg)
    c.on_mic(tone(100, True))
    assert sess.partials  # гипотезы грели сессию
    assert not sess.finals
    c.on_mic(tone(200, False))  # тишина -> финал
    assert sess.finals == ["на каком этаже горит"]


def test_playback_and_barge_in(tmp_path):
    (tmp_path / "a").mkdir()
    with wave.open(str(tmp_path / "a/x.wav"), "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(8000)
        w.writeframes(b"\x01\x02" * 800)
    assert load_reply_pcm("a/x.wav+a/x.wav", tmp_path) == b"\x01\x02" * 1600
    assert load_reply_pcm("a/missing.wav", tmp_path) is None
    sess, sent = StubSession(), []
    c = Call(sess, FakeSTT(), sent.append, MediaConfig(audio_root=tmp_path))
    c._answer("на каком этаже горит")
    assert b"".join(sent) == b"\x01\x02" * 800  # проигралось целиком
    c.playing = b"\x00" * 100
    c.on_barge_in()
    assert c.playing is None and sess.cancelled == 1
