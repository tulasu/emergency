# -*- coding: utf-8 -*-
"""Общее для бенчей: корпус из сценариев и разбор связки модель:бэкенд."""

from __future__ import annotations

from dispatcher.data.loader import load_all
from dispatcher.data.ontology import Ontology
from dispatcher.nlu.bank import LexicalBank


def load_bank():
    """Онтология, сценарии и лексический банк — преамбула каждого бенча."""
    onto = Ontology.load()
    loaded = load_all(onto)
    return onto, loaded, LexicalBank.build(loaded.questions, loaded.sources)


def make_vectors(spec, questions, sources):
    """Энкодер и векторный банк из строки вида `e5-small:onnx`.

    Устройство выбирается по бэкенду: у torch-cuda свой проход, остальным
    хватает CPU. Пустой backend — `auto`, как в `encoder.build`.
    """
    from dispatcher.nlu import encoder as enc
    from dispatcher.nlu.vectors import VectorBank

    name, _, backend = (spec or "").partition(":")
    backend = backend or "auto"
    device = "cuda" if backend == "torch-cuda" else "cpu"
    if backend.startswith("torch"):
        backend = "torch"
    encoder = enc.build(name, backend=backend, device=device)
    vectors = VectorBank.build(questions, sources, encoder)
    return encoder, vectors
