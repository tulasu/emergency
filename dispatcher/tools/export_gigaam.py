# -*- coding: utf-8 -*-
"""Экспорт GigaAM-v2 CTC в onnx + матрица мел-фильтров для numpy-фронтенда.

Нужны torch, torchaudio и pip-пакет gigaam — только здесь, в рантайме
(dispatcher/media/gigaam_stt.py) хватает onnxruntime и numpy.

  python3 tools/export_gigaam.py            # -> models/gigaam-v2-ctc/
"""

from __future__ import annotations

from pathlib import Path

import gigaam
import numpy as np
import torchaudio

OUT = Path(__file__).resolve().parents[1] / "models" / "gigaam-v2-ctc"


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    m = gigaam.load_model("v2_ctc", device="cpu", fp16_encoder=False)
    m.to_onnx(str(OUT))
    # те же параметры, что у gigaam.preprocess.FeatureExtractor(16000, 64)
    fb = torchaudio.functional.melscale_fbanks(201, 0.0, 8000.0, 64, 16000, None, "htk")
    np.save(OUT / "mel_fbank.npy", fb.numpy().astype(np.float32))
    print(f"готово: {OUT}")


if __name__ == "__main__":
    main()
