# -*- coding: utf-8 -*-
"""
Энкодер: тексты -> нормированные векторы.

Один интерфейс, несколько реализаций. Сравниваются они бенчмарком, а ядро
о разнице не знает: банк получает матрицу векторов и ищет по косинусу.

Векторы всегда L2-нормированы, поэтому косинус — это обычное скалярное
произведение, и поиск по банку сводится к одному умножению матриц.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path
from typing import Literal, Protocol

import numpy as np

Kind = Literal["query", "passage"]
MODELS_DIR = Path(
    os.environ.get("DISPATCHER_MODELS", Path(__file__).resolve().parents[2] / "models")
)


@dataclass(frozen=True, slots=True)
class Spec:
    """Как обращаться с конкретной моделью."""

    repo: str
    dim: int
    pooling: Literal["mean", "cls"] = "mean"
    query_prefix: str = ""
    passage_prefix: str = ""
    max_len: int = 64  # реплика оператора короткая, длинное окно не нужно
    onnx_dir: str = ""  # папка с готовым экспортом в репозитории, если есть


# Модели русского языка, помещающиеся в бюджет реального времени.
REGISTRY: dict[str, Spec] = {
    "rubert-tiny2": Spec("cointegrated/rubert-tiny2", 312),
    "rubert-tiny-turbo": Spec("sergeyzh/rubert-tiny-turbo", 312),
    # e5 без префиксов молча теряет качество — самая частая ошибка интеграции
    "e5-small": Spec(
        "intfloat/multilingual-e5-small",
        384,
        query_prefix="query: ",
        passage_prefix="passage: ",
        onnx_dir="onnx",
    ),
    "labse-ru-turbo": Spec("sergeyzh/LaBSE-ru-turbo", 768),
    # та же модель, но веса для torch лежат отдельно от экспорта в onnx
    "e5-small-torch": Spec(
        "intfloat/multilingual-e5-small",
        384,
        query_prefix="query: ",
        passage_prefix="passage: ",
    ),
    # дообученный на формулировках корпуса; веса только локальные,
    # собираются через tools/train_encoder.py
    "e5-small-tuned": Spec("", 384, query_prefix="query: ", passage_prefix="passage: "),
}


class Encoder(Protocol):
    name: str
    dim: int

    def encode(self, texts: list[str], kind: Kind = "query") -> np.ndarray: ...


def _l2(x: np.ndarray) -> np.ndarray:
    n = np.linalg.norm(x, axis=1, keepdims=True)
    return x / np.maximum(n, 1e-12)


class TorchEncoder:
    """transformers на CPU или CUDA. Эталон качества, с ним сверяется ONNX."""

    def __init__(self, name: str, device: str = "cpu", batch: int = 64):
        import torch
        from transformers import AutoModel, AutoTokenizer

        if name not in REGISTRY:
            raise KeyError(f"нет модели {name}; есть: {', '.join(REGISTRY)}")
        self.spec = REGISTRY[name]
        self.name = f"{name}/torch-{device}"
        self.device = device
        self.batch = batch
        self._torch = torch

        # Положенная рядом папка важнее Хаба: рантайму нужна работа без сети
        local = MODELS_DIR / name
        if (local / "config.json").exists():
            source = str(local)
        elif not self.spec.repo:
            raise FileNotFoundError(
                f"{name}: весов нет ни рядом ({local}), ни на Хабе — "
                f"соберите их через tools/train_encoder.py"
            )
        else:
            source = self.spec.repo
        self.tok = AutoTokenizer.from_pretrained(source, cache_dir=MODELS_DIR)
        self.model = AutoModel.from_pretrained(source, cache_dir=MODELS_DIR)
        self.model.eval().to(device)
        self.dim = int(self.model.config.hidden_size)
        if self.dim != self.spec.dim:
            raise ValueError(
                f"{name}: ожидали {self.spec.dim} измерений, " f"модель даёт {self.dim}"
            )

    def encode(self, texts: list[str], kind: Kind = "query") -> np.ndarray:
        if not texts:
            return np.zeros((0, self.dim), dtype=np.float32)
        prefix = self.spec.query_prefix if kind == "query" else self.spec.passage_prefix
        torch = self._torch
        out: list[np.ndarray] = []

        with torch.inference_mode():
            for i in range(0, len(texts), self.batch):
                chunk = [prefix + t for t in texts[i : i + self.batch]]
                enc = self.tok(
                    chunk,
                    padding=True,
                    truncation=True,
                    max_length=self.spec.max_len,
                    return_tensors="pt",
                ).to(self.device)
                hidden = self.model(**enc).last_hidden_state
                if self.spec.pooling == "cls":
                    vec = hidden[:, 0]
                else:
                    mask = enc["attention_mask"].unsqueeze(-1).to(hidden.dtype)
                    vec = (hidden * mask).sum(1) / mask.sum(1).clamp(min=1e-9)
                out.append(vec.float().cpu().numpy())

        return _l2(np.concatenate(out)).astype(np.float32)


class OnnxEncoder:
    """onnxruntime на CPU. Целевой рантайм: ни torch, ни CUDA не нужны.

    Квантованная в int8 модель весит вчетверо меньше и считается быстрее,
    расходясь с эталоном на сотые доли косинуса — на выбор слота это не влияет.
    """

    def __init__(
        self,
        name: str,
        quantized: bool = True,
        threads: int | None = None,
        batch: int = 64,
    ):
        import onnxruntime as ort
        from tokenizers import Tokenizer

        from huggingface_hub import hf_hub_download

        if name not in REGISTRY:
            raise KeyError(f"нет модели {name}; есть: {', '.join(REGISTRY)}")
        self.spec = REGISTRY[name]
        if not self.spec.onnx_dir:
            raise ValueError(
                f"{name}: готового экспорта в onnx нет, " f"нужен бэкенд torch"
            )
        self.name = f"{name}/onnx{'-int8' if quantized else ''}"
        self.batch = batch
        self.dim = self.spec.dim

        d = self.spec.onnx_dir
        weights = "model_qint8_avx512_vnni.onnx" if quantized else "model.onnx"

        # Положенная рядом модель важнее скачанной: рантайму нужна работа
        # без сети, да и качать по сто мегабайт на каждой машине незачем
        local = MODELS_DIR / name
        if (local / weights).exists() and (local / "tokenizer.json").exists():
            model_path = str(local / weights)
            tok_path = str(local / "tokenizer.json")
        else:
            model_path = hf_hub_download(
                self.spec.repo, f"{d}/{weights}", cache_dir=MODELS_DIR
            )
            tok_path = hf_hub_download(
                self.spec.repo, f"{d}/tokenizer.json", cache_dir=MODELS_DIR
            )

        self.tok = Tokenizer.from_file(tok_path)
        self.tok.enable_truncation(self.spec.max_len)
        self.tok.enable_padding()

        opts = ort.SessionOptions()
        opts.intra_op_num_threads = threads or min(4, os.cpu_count() or 1)
        opts.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
        self.sess = ort.InferenceSession(
            model_path, opts, providers=["CPUExecutionProvider"]
        )
        self.wants = {i.name for i in self.sess.get_inputs()}

    def encode(self, texts: list[str], kind: Kind = "query") -> np.ndarray:
        if not texts:
            return np.zeros((0, self.dim), dtype=np.float32)
        prefix = self.spec.query_prefix if kind == "query" else self.spec.passage_prefix
        out: list[np.ndarray] = []

        for i in range(0, len(texts), self.batch):
            batch = self.tok.encode_batch(
                [prefix + t for t in texts[i : i + self.batch]]
            )
            ids = np.array([b.ids for b in batch], dtype=np.int64)
            mask = np.array([b.attention_mask for b in batch], dtype=np.int64)
            feed = {"input_ids": ids, "attention_mask": mask}
            if "token_type_ids" in self.wants:
                feed["token_type_ids"] = np.zeros_like(ids)
            feed = {k: v for k, v in feed.items() if k in self.wants}

            hidden = self.sess.run(None, feed)[0]  # (b, len, dim)
            if self.spec.pooling == "cls":
                vec = hidden[:, 0]
            else:
                m = mask[..., None].astype(np.float32)
                vec = (hidden * m).sum(1) / np.maximum(m.sum(1), 1e-9)
            out.append(vec.astype(np.float32))

        return _l2(np.concatenate(out))


def build(
    name: str = "rubert-tiny2", backend: str = "auto", device: str = "cpu"
) -> Encoder:
    """Собрать энкодер. `auto` берёт onnx, если он есть, иначе torch."""
    if backend == "auto":
        backend = "onnx" if (REGISTRY[name].onnx_dir and device == "cpu") else "torch"
    if backend == "onnx":
        return OnnxEncoder(name)
    if backend == "onnx-fp32":
        return OnnxEncoder(name, quantized=False)
    if backend == "torch":
        return TorchEncoder(name, device)
    raise ValueError(f"нет бэкенда {backend}")


def available_device() -> str:
    """cuda, если видеокарта поднята, иначе cpu."""
    try:
        import torch

        return "cuda" if torch.cuda.is_available() else "cpu"
    except Exception:
        return "cpu"
