# -*- coding: utf-8 -*-
"""
Векторный банк на поддельном энкодере.

Модель для этих проверок не нужна: важно, что банк правильно группирует
формулировки по слотам, вычёркивает сценарий и переживает запись на диск.
"""

import hashlib

import numpy as np
import pytest

from dispatcher.nlu.vectors import VectorBank


class StubEncoder:
    """Детерминированные векторы из хэша слова. Похожие строки — похожие векторы."""

    name = "stub"
    dim = 16

    def encode(self, texts, kind="query"):
        out = np.zeros((len(texts), self.dim), dtype=np.float32)
        for i, t in enumerate(texts):
            for word in t.lower().split():
                h = int(hashlib.blake2b(word.encode(), digest_size=4).hexdigest(), 16)
                out[i, h % self.dim] += 1.0
        n = np.linalg.norm(out, axis=1, keepdims=True)
        return out / np.maximum(n, 1e-12)


QUESTIONS = {
    "addr.full": ["какой адрес", "адрес происшествия", "где это"],
    "fire.floor": ["какой этаж горит", "этаж возгорания"],
}
SOURCES = {
    "addr.full": ["bilet01_call01", "bilet02_call01", "bilet01_call01"],
    "fire.floor": ["bilet04_call01", "bilet04_call01"],
}


@pytest.fixture(scope="module")
def bank():
    return VectorBank.build(QUESTIONS, SOURCES, StubEncoder(), digest="тест")


def test_bank_grouped_by_slot(bank):
    assert bank.slots == ("addr.full", "fire.floor")
    assert bank.matrix.shape == (5, 16)
    assert list(bank.starts) == [0, 3, 5]


def test_vectors_normalized(bank):
    assert np.allclose(np.linalg.norm(bank.matrix, axis=1), 1.0, atol=1e-5)


def test_utterance_hits_own_slot(bank):
    q = StubEncoder().encode(["какой этаж горит"])[0]
    scores = bank.score(q)
    assert scores["fire.floor"] > scores["addr.full"]


def test_excluded_scenario_absent(bank):
    q = StubEncoder().encode(["этаж возгорания"])[0]
    full = bank.score(q)["fire.floor"]
    excluded = bank.score(q, without="bilet04_call01")["fire.floor"]
    assert full == pytest.approx(1.0, abs=1e-5)
    assert excluded < full  # все формулировки слота были из этого сценария


def test_foreign_slots_cut_off(bank):
    q = StubEncoder().encode(["какой адрес"])[0]
    assert set(bank.score(q, allowed=frozenset({"addr.full"}))) == {"addr.full"}


def test_save_and_load(bank, tmp_path):
    path = tmp_path / "bank.npz"
    bank.save(path)
    back = VectorBank.load(path)
    assert back.slots == bank.slots
    assert back.digest == "тест"
    assert np.allclose(back.matrix, bank.matrix)
    assert back.texts == bank.texts
