# -*- coding: utf-8 -*-
"""Живой TTS: Silero v4_ru для ответов без предрендера.

Тот же голос и тракт, что у tools/synth_audio.py: синтез на 48 кГц,
децимация до 8 кГц, цифры словами (Silero их молча выбрасывает).
Только GPU в звонке: на CPU под нагрузкой звонка (GigaAM, энкодер) фраза
синтезировалась 3–7 с, и всё это время поток чтения AudioSocket стоял —
звонок «глох». На GPU — десятки миллисекунд, модель занимает ~0.2 ГБ.
"""

from __future__ import annotations

import audioop
import threading

import numpy as np

_RATE = 48000


class SileroTTS:
    def __init__(self, speaker: str = "kseniya", device: str = "cuda"):
        import torch

        self._torch = torch
        self.speaker = speaker
        self.device = torch.device(device if torch.cuda.is_available() else "cpu")
        self._model, _ = torch.hub.load(
            repo_or_dir="snakers4/silero-models", model="silero_tts",
            language="ru", speaker="v4_ru", trust_repo=True,
        )
        self._model.to(self.device)
        self._lock = threading.Lock()  # модель одна на процесс, звонков много
        # Прогрев: Silero — TorchScript, и первые два вызова уходят на
        # профилирование (~2 с каждый). Фразы разной длины и дважды — после
        # этого любая новая фраза 15–30 мс, а не 2 с посреди звонка.
        for text in ("Алло.", "Да, поняла вас.", "Не знаю, не видела, я там не была.",
                     "Москва, улица Красный Казанец, дом девятнадцать Б, на стоянке.") * 2:
            self(text)

    def __call__(self, text: str) -> bytes:
        from tools.synth_audio import speakable

        with self._lock, self._torch.inference_mode():
            audio = self._model.apply_tts(
                text=speakable(text), speaker=self.speaker, sample_rate=_RATE,
                put_accent=True, put_yo=True)
            wide = audio.cpu().numpy()
        pcm48 = np.clip(wide * 32767, -32768, 32767).astype("<i2").tobytes()
        pcm8, _ = audioop.ratecv(pcm48, 2, 1, _RATE, 8000, None)
        return pcm8
