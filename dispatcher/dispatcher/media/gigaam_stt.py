# -*- coding: utf-8 -*-
"""STT на GigaAM-v2 CTC (Сбер) через onnxruntime, CPU.

Не стриминг: копит pcm16 16 кГц, final() прогоняет всю фразу (~0.5 с
на CPU), partial() — раз в _PARTIAL_EVERY_S по накопленному буферу, чтобы
Session успела посчитать ответ заранее. Torch не нужен: лог-мел считается
на numpy, матрица мел-фильтров экспортирована рядом с моделью
(tools/export_gigaam.py) — один в один torchaudio.MelSpectrogram.
"""

from __future__ import annotations

import os
from pathlib import Path

import numpy as np

_DEFAULT_MODEL = Path(
    os.environ.get("GIGAAM_MODEL_PATH", "/app/models/gigaam-v2-ctc")
)
_RATE = 16000
_N_FFT = 400  # sample_rate // 40
_HOP = 160  # sample_rate // 100
_VOCAB = " абвгдежзийклмнопрстуфхцчшщъыьэюя"
_BLANK = len(_VOCAB)
_PARTIAL_EVERY_S = 1.0
_MAX_BUF_S = 30  # тишина до фразы копится — дальше хвост обрезаем


def _logmel(audio: np.ndarray, fbank: np.ndarray) -> np.ndarray:
    """torchaudio.MelSpectrogram(center, reflect, hann periodic, power=2) + log."""
    pad = _N_FFT // 2
    x = np.pad(audio, pad, mode="reflect")
    n = 1 + (len(x) - _N_FFT) // _HOP
    idx = np.arange(_N_FFT)[None, :] + _HOP * np.arange(n)[:, None]
    win = np.hanning(_N_FFT + 1)[:-1].astype(np.float32)  # periodic
    spec = np.abs(np.fft.rfft(x[idx] * win, axis=1)) ** 2  # [T, 201]
    mel = spec.astype(np.float32) @ fbank  # [T, 64]
    return np.log(np.clip(mel, 1e-9, 1e9)).T[None].astype(np.float32)  # [1, 64, T]


class GigaAMSTT:
    def __init__(self, model_path: str | Path | None = None, threads: int = 4):
        import onnxruntime as rt  # noqa: import здесь — медленный

        mp = Path(model_path) if model_path else _DEFAULT_MODEL
        onnx = next(iter(sorted(mp.glob("*.onnx"))), None)
        if onnx is None:
            raise FileNotFoundError(
                f"gigaam onnx не найден в {mp}: python tools/export_gigaam.py")
        opts = rt.SessionOptions()
        opts.intra_op_num_threads = threads
        self._sess = rt.InferenceSession(str(onnx), opts,
                                         providers=["CPUExecutionProvider"])
        self._in = [i.name for i in self._sess.get_inputs()]
        self._fbank = np.load(mp / "mel_fbank.npy")
        self._buf = bytearray()
        self._since_partial = 0

    def transcribe(self, pcm16: bytes) -> str:
        if len(pcm16) < _N_FFT * 2:
            return ""
        audio = np.frombuffer(pcm16, dtype=np.int16).astype(np.float32) / 32768.0
        feats = _logmel(audio, self._fbank)
        logp = self._sess.run(None, {
            self._in[0]: feats,
            self._in[1]: np.array([feats.shape[-1]], dtype=np.int64),
        })[0]
        out, prev = [], _BLANK
        for tok in logp[0].argmax(-1).tolist():
            if tok != _BLANK and tok != prev:
                out.append(_VOCAB[tok])
            prev = tok
        return " ".join("".join(out).split())

    def partial(self, pcm16: bytes) -> str:
        self._buf.extend(pcm16)
        if len(self._buf) > _MAX_BUF_S * _RATE * 2:
            del self._buf[:len(self._buf) - _MAX_BUF_S * _RATE * 2]
        self._since_partial += len(pcm16)
        if self._since_partial < _PARTIAL_EVERY_S * _RATE * 2:
            return ""
        self._since_partial = 0
        return self.transcribe(bytes(self._buf))

    def final(self) -> str:
        text = self.transcribe(bytes(self._buf))
        self._buf.clear()
        self._since_partial = 0
        return text
