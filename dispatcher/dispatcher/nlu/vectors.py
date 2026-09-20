# -*- coding: utf-8 -*-
"""
Векторный банк слотов: реплика -> оценки слотов по косинусу.

Формулировки лежат отсортированными по слоту, поэтому максимум внутри слота
берётся одним `np.maximum.reduceat` — без словарей и циклов по 4599 строкам.
Оценка слота — максимум, а не центроид: формулировки внутри слота разнородны,
и усреднение «какой этаж горит» с «этаж повторите» размывает обе.

Векторы считаются один раз на сборке и кладутся в data/build.
"""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import numpy as np

from .encoder import Encoder


@dataclass(slots=True)
class VectorBank:
    slots: tuple[str, ...]  # имена слотов, порядок = порядок групп
    matrix: np.ndarray  # (n, dim), L2-нормированные, по слотам
    starts: np.ndarray  # (len(slots)+1,) границы групп в matrix
    sources: np.ndarray  # (n,) индекс сценария-источника
    scenario_ids: tuple[str, ...] = ()
    texts: tuple[str, ...] = ()
    model: str = ""
    digest: str = ""
    # Косинусы разных энкодеров живут в разных полосах: у e5 всё лежит выше
    # 0.75, и абсолютный порог перестаёт что-либо различать. Приводим к общей
    # шкале, вычитая фон — среднюю похожесть случайной пары.
    baseline: float = 0.0

    # ---------------------------------------------------------- построение

    @staticmethod
    def build(
        questions: dict[str, list[str]],
        sources: dict[str, list[str]],
        encoder: Encoder,
        digest: str = "",
    ) -> "VectorBank":
        slot_names = tuple(sorted(questions))
        texts: list[str] = []
        origins: list[str] = []
        starts = [0]
        for slot in slot_names:
            qs = questions[slot]
            src = sources.get(slot, [""] * len(qs))
            texts.extend(qs)
            origins.extend(src[i] if i < len(src) else "" for i in range(len(qs)))
            starts.append(len(texts))

        matrix = encoder.encode(texts, kind="passage")

        rng = np.random.default_rng(0)
        n = len(matrix)
        if n > 16:
            a = rng.integers(0, n, 4096)
            b = rng.integers(0, n, 4096)
            keep = a != b
            baseline = float(
                np.median(np.einsum("ij,ij->i", matrix[a[keep]], matrix[b[keep]]))
            )
        else:
            baseline = 0.0

        ids = {s: i for i, s in enumerate(sorted(set(origins)))}
        return VectorBank(
            slots=slot_names,
            matrix=matrix,
            starts=np.asarray(starts, dtype=np.int64),
            sources=np.asarray([ids[o] for o in origins], dtype=np.int32),
            scenario_ids=tuple(sorted(set(origins))),
            texts=tuple(texts),
            model=encoder.name,
            digest=digest,
            baseline=baseline,
        )

    # ------------------------------------------------------------- хранение

    def save(self, path: Path) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        np.savez_compressed(
            path,
            matrix=self.matrix,
            starts=self.starts,
            sources=self.sources,
            slots=np.array(self.slots),
            scenario_ids=np.array(self.scenario_ids),
            texts=np.array(self.texts),
            meta=np.array([self.model, self.digest, str(self.baseline)]),
        )

    @staticmethod
    def load(path: Path) -> "VectorBank":
        z = np.load(path, allow_pickle=False)
        meta = [str(x) for x in z["meta"]]
        model, digest = meta[0], meta[1]
        baseline = float(meta[2]) if len(meta) > 2 else 0.0
        return VectorBank(
            slots=tuple(str(s) for s in z["slots"]),
            matrix=z["matrix"],
            starts=z["starts"],
            sources=z["sources"],
            scenario_ids=tuple(str(s) for s in z["scenario_ids"]),
            texts=tuple(str(t) for t in z["texts"]),
            model=model,
            digest=digest,
            baseline=baseline,
        )

    # --------------------------------------------------------------- оценка

    def subset(self, slots: frozenset[str]) -> "VectorBank":
        """Срез банка под один сценарий.

        Заявитель отвечает о десятке слотов из восьмидесяти восьми, а поиск
        читает всю матрицу целиком — 6.7 МБ мимо кэша на каждую реплику.
        После энкодера эта матрица из кэша уже вытеснена, поэтому чтение идёт
        из памяти и стоит миллисекунды, хотя само умножение — микросекунды.
        Срез на десяток слотов укладывается в кэш и читается втрое быстрее.
        """
        keep = [i for i, sl in enumerate(self.slots) if sl in slots]
        if not keep:
            return VectorBank(
                slots=(),
                matrix=self.matrix[:0],
                starts=np.zeros(1, dtype=np.int64),
                sources=self.sources[:0],
                scenario_ids=self.scenario_ids,
                texts=(),
                model=self.model,
                digest=self.digest,
                baseline=self.baseline,
            )

        rows: list[np.ndarray] = []
        srcs: list[np.ndarray] = []
        starts = [0]
        names: list[str] = []
        for i in keep:
            a, b = int(self.starts[i]), int(self.starts[i + 1])
            rows.append(self.matrix[a:b])
            srcs.append(self.sources[a:b])
            starts.append(starts[-1] + (b - a))
            names.append(self.slots[i])

        return VectorBank(
            slots=tuple(names),
            matrix=np.ascontiguousarray(np.concatenate(rows)),
            starts=np.asarray(starts, dtype=np.int64),
            sources=np.concatenate(srcs),
            scenario_ids=self.scenario_ids,
            texts=(),
            model=self.model,
            digest=self.digest,
            baseline=self.baseline,
        )

    def _source_id(self, scenario: str) -> int:
        try:
            return self.scenario_ids.index(scenario)
        except ValueError:
            return -1

    def score(
        self, qvec: np.ndarray, allowed: frozenset[str] | None = None, without: str = ""
    ) -> dict[str, float]:
        """Косинус реплики с лучшей формулировкой каждого слота."""
        sims = self.matrix @ qvec
        if without:
            sid = self._source_id(without)
            if sid >= 0:
                sims = np.where(self.sources == sid, -1.0, sims)

        # границы групп: пустых слотов в банке нет, reduceat безопасен
        best = np.maximum.reduceat(sims, self.starts[:-1])
        span = max(1.0 - self.baseline, 1e-6)
        out = {}
        for i, slot in enumerate(self.slots):
            if allowed is not None and slot not in allowed:
                continue
            out[slot] = max(0.0, (float(best[i]) - self.baseline) / span)
        return out
