# -*- coding: utf-8 -*-
"""
Банк формулировок: реплика -> оценки слотов.

Формулировки принадлежат слоту, а не факту, поэтому банк один на весь корпус:
за слот «адрес» проголосовали десятки сценариев. Новый сценарий получает эти
формулировки даром — ради этого и заводилась онтология.

Лексическая оценка здесь самостоятельна и работает офлайн без единой модели.
Векторная надстройка появится рядом и будет с ней сплавляться.
"""

from __future__ import annotations

import math
import random
from collections import defaultdict
from dataclasses import dataclass, field

from . import normalize


@dataclass(slots=True)
class Entry:
    slot: str
    lemmas: frozenset[str]
    weight: float  # суммарный вес лемм, считается один раз
    text: str
    source: str = ""  # сценарий-источник — для честной оценки


@dataclass(slots=True)
class LexicalBank:
    """Пересечение лемм, взвешенное обратной частотой.

    Обратный индекс: лемма -> номера формулировок. Реплика сравнивается
    только с теми, у кого есть хоть одна общая лемма, а не со всеми 4599.
    """

    entries: list[Entry] = field(default_factory=list)
    postings: dict[str, list[int]] = field(default_factory=dict)
    idf: dict[str, float] = field(default_factory=dict)
    default_idf: float = 1.0
    slots: tuple[str, ...] = ()
    # Фон слота: его средняя оценка на посторонних репликах. У слота с 466
    # формулировками максимум завышен просто потому, что попыток больше —
    # эта фора вычитается, иначе редкий слот не выигрывает никогда.
    floor: dict[str, float] = field(default_factory=dict)
    # сколько формулировок накопил слот: на бедных банк даёт 6-13%, на
    # населённых 83%, и это решает, кому отдавать спорный случай
    population: dict[str, int] = field(default_factory=dict)

    @staticmethod
    def build(
        questions: dict[str, list[str]], sources: dict[str, list[str]] | None = None
    ) -> "LexicalBank":
        raw: list[tuple[str, frozenset[str], str, str]] = []
        df: dict[str, int] = defaultdict(int)
        for slot, texts in questions.items():
            origin = (sources or {}).get(slot, ())
            for i, text in enumerate(texts):
                lemmas = frozenset(normalize.content_lemmas(text))
                if not lemmas:
                    continue
                raw.append((slot, lemmas, text, origin[i] if i < len(origin) else ""))
                for lm in lemmas:
                    df[lm] += 1

        n = len(raw) or 1
        idf = {lm: math.log(1 + n / c) for lm, c in df.items()}
        default = max(idf.values(), default=1.0)

        entries: list[Entry] = []
        postings: dict[str, list[int]] = defaultdict(list)
        for i, (slot, lemmas, text, origin) in enumerate(raw):
            weight = sum(idf.get(lm, default) for lm in lemmas)
            entries.append(Entry(slot, lemmas, weight, text, origin))
            for lm in lemmas:
                postings[lm].append(i)

        bank = LexicalBank(entries, dict(postings), idf, default, tuple(questions))
        bank.population = {s: len(v) for s, v in questions.items()}
        bank.floor = bank._background()
        return bank

    def _background(self, sample: int = 400) -> dict[str, float]:
        rnd = random.Random(0)
        texts = [e.text for e in self.entries]
        if len(texts) > sample:
            texts = rnd.sample(texts, sample)
        total: dict[str, float] = {}
        for t in texts:
            for slot, v in self.score(t, _raw=True).items():
                total[slot] = total.get(slot, 0.0) + v
        return {s: v / len(texts) for s, v in total.items()}

        # ------------------------------------------------------------- оценка

    def _w(self, lemmas) -> float:
        return sum(self.idf.get(lm, self.default_idf) for lm in lemmas)

    def score(
        self,
        text: str,
        allowed: frozenset[str] | None = None,
        without: str = "",
        _raw: bool = False,
    ) -> dict[str, float]:
        """Оценка каждого слота, у которого есть общая лемма с репликой.

        `without` убирает формулировки одного сценария: так меряется то, что
        случится с новой ситуацией, которой банк ещё не видел.
        """
        query = frozenset(normalize.content_lemmas(text))
        if not query:
            return {}
        qw = self._w(query)

        touched: set[int] = set()
        for lm in query:
            touched.update(self.postings.get(lm, ()))

        best: dict[str, float] = {}
        for i in touched:
            e = self.entries[i]
            if allowed is not None and e.slot not in allowed:
                continue
            if without and e.source == without:
                continue
            total = qw + e.weight
            if not total:
                continue
                # мера Дайса по весам: длина формулировки не наказывается
            s = 2 * self._w(query & e.lemmas) / total
            if s > best.get(e.slot, 0.0):
                best[e.slot] = s
        if _raw or not self.floor:
            return best
        return {k: v - self.floor.get(k, 0.0) for k, v in best.items()}

    def top(
        self,
        text: str,
        k: int = 5,
        allowed: frozenset[str] | None = None,
        without: str = "",
    ) -> list[tuple[str, float]]:
        scored = self.score(text, allowed, without)
        return sorted(scored.items(), key=lambda kv: -kv[1])[:k]
