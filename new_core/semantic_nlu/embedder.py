# -*- coding: utf-8 -*-
"""CPU sentence embeddings: sergeyzh/rubert-tiny-sts."""
from __future__ import annotations

import time
from typing import Sequence

import numpy as np

DEFAULT_MODEL = "sergeyzh/rubert-tiny-sts"


class Embedder:
    """Ленивая загрузка модели. Векторы L2-нормированы → cosine = dot."""

    def __init__(self, model_name: str = DEFAULT_MODEL, device: str = "cpu"):
        self.model_name = model_name
        self.device = device
        self._model = None
        self.load_ms: float = 0.0

    @property
    def loaded(self) -> bool:
        return self._model is not None

    def load(self) -> float:
        """Загрузить модель. Возвращает cold-load latency в мс."""
        if self._model is not None:
            return self.load_ms
        from sentence_transformers import SentenceTransformer

        t0 = time.perf_counter()
        self._model = SentenceTransformer(self.model_name, device=self.device)
        self.load_ms = (time.perf_counter() - t0) * 1000
        return self.load_ms

    def encode(self, texts: Sequence[str], batch_size: int = 32) -> np.ndarray:
        self.load()
        assert self._model is not None
        if not texts:
            return np.zeros((0, 0), dtype=np.float32)
        vecs = self._model.encode(
            list(texts),
            batch_size=batch_size,
            normalize_embeddings=True,
            show_progress_bar=False,
            convert_to_numpy=True,
        )
        return np.asarray(vecs, dtype=np.float32)

    def encode_one(self, text: str) -> np.ndarray:
        return self.encode([text])[0]
