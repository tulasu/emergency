# -*- coding: utf-8 -*-
"""STT на faster-whisper tiny (CPU, int8).

Не стриминг — копит pcm16 16кГц, по запросу даёт partial (если буфер > 1.5с)
и final (по требованию вызывающего, со сбросом). Дешевле Vosk и без модели
45 МБ на диске, но задержка выше — зато 'поставил и работает' через pip.
"""

from __future__ import annotations

import numpy as np

_PARTIAL_S = 1.5
_RATE = 16000


class WhisperSTT:
    def __init__(self, model: str = "tiny", language: str = "ru"):
        from faster_whisper import WhisperModel  # noqa: import here — медленный
        self._m = WhisperModel(model, device="cpu", compute_type="int8")
        self.lang = language
        self._buf = bytearray()

    def partial(self, pcm16: bytes) -> str:
        self._buf.extend(pcm16)
        if len(self._buf) < _PARTIAL_S * _RATE * 2:
            return ""
        return self._decode()[0]

    def final(self) -> str:
        if not self._buf:
            return ""
        text, _ = self._decode()
        self._buf.clear()
        return text

    def _decode(self) -> tuple[str, float]:
        audio = np.frombuffer(self._buf, dtype=np.int16).astype(np.float32) / 32768.0
        segs, _info = self._m.transcribe(audio, language=self.lang,
                                         beam_size=1, vad_filter=False)
        text = " ".join(s.text.strip() for s in segs).strip()
        return text, 0.0
