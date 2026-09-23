# -*- coding: utf-8 -*-
"""STT на Vosk — стриминг, по факту готовности выдаёт partial/final.

vosk-model-small-ru-0.22 (~45 МБ), CPU-only. partial — гипотеза после
~0.5 с речи, final — после паузы > 0.6 с (AcceptWaveform возвращает True).
"""

from __future__ import annotations

import json
import os
from pathlib import Path

_DEFAULT_MODEL = Path(
    os.environ.get("VOSK_MODEL_PATH",
                   "/app/models/vosk-model-small-ru-0.22")
)


class VoskSTT:
    def __init__(self, model_path: str | Path | None = None,
                 sample_rate: int = 16000):
        from vosk import Model, KaldiRecognizer  # noqa: import здесь — медленный
        mp = Path(model_path) if model_path else _DEFAULT_MODEL
        if not mp.exists():
            raise FileNotFoundError(
                f"vosk модель не найдена: {mp}. "
                f"скачай vosk-model-small-ru-0.22 и положи в {mp}")
        self._rec = KaldiRecognizer(Model(str(mp)), sample_rate)
        self._rec.SetWords(True)
        self._rate = sample_rate
        # сегменты, которые vosk закрыл сам (своя пауза короче нашего VAD):
        # без накопления они терялись, и final() отдавал пустоту
        self._done: list[str] = []

    def partial(self, pcm16: bytes) -> str:
        if self._rec.AcceptWaveform(pcm16):
            t = json.loads(self._rec.Result()).get("text", "")
            if t:
                self._done.append(t)
            return " ".join(self._done)
        p = json.loads(self._rec.PartialResult()).get("partial", "")
        return " ".join([*self._done, p]).strip()

    def final(self) -> str:
        tail = json.loads(self._rec.FinalResult()).get("text", "")
        text = " ".join([*self._done, tail]).strip()
        self._done = []
        return text
