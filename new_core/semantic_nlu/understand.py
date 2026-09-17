# -*- coding: utf-8 -*-
"""
Semantic matching: max cosine к любому question факта (не mean-прототип).

Accept при score >= min_confidence и (margin от 2-го ИЛИ strong score).
Отрицания: лёгкий heuristic (частицы «не»/«ни») → demote/reject.
"""
from __future__ import annotations

import re
import time
from dataclasses import dataclass, field

import numpy as np

try:
    from .embedder import DEFAULT_MODEL, Embedder
except ImportError:  # python -m bench / script mode
    from embedder import DEFAULT_MODEL, Embedder

# Cosine STS у близких пожарных фактов часто gap < 0.05 при score > 0.85.
DEFAULT_THRESHOLD = 0.45
DEFAULT_MARGIN = 0.03
# Выше этого — принимаем top-1 даже при узком margin (иначе «что горит» → dont_know).
DEFAULT_STRONG = 0.85
NEGATION_RE = re.compile(
    r"(?:\bне\b|\bни\b|\bнет\b|\bнельзя\b|\bненужн)",
    re.IGNORECASE,
)


@dataclass
class MatchResult:
    keys: list[str] = field(default_factory=list)
    score: float = 0.0
    second: float = 0.0
    status: str = "reject"  # accept | ambiguous | reject
    negated: bool = False
    encode_ms: float = 0.0
    rank_ms: float = 0.0

    @property
    def total_ms(self) -> float:
        return self.encode_ms + self.rank_ms


class SemanticUnderstander:
    name = "semantic"

    def __init__(
        self,
        sc,
        embedder: Embedder | None = None,
        threshold: float = DEFAULT_THRESHOLD,
        margin: float = DEFAULT_MARGIN,
        min_confidence: float | None = None,
        strong: float = DEFAULT_STRONG,
    ):
        self.sc = sc
        self.embedder = embedder or Embedder(DEFAULT_MODEL)
        self.threshold = threshold
        self.margin = margin
        self.strong = strong
        self.min_confidence = (
            min_confidence if min_confidence is not None
            else getattr(sc.profile, "min_confidence", 0.45)
        )
        self.keys: list[str] = []
        # per-key matrix of question embeddings (n_q, dim)
        self.qvecs: list[np.ndarray] = []
        self.build_ms: float = 0.0

    def build(self) -> float:
        """Индекс: все questions каждого факта. Score = max cosine."""
        self.embedder.load()
        t0 = time.perf_counter()
        keys = list(self.sc.facts.keys())
        qvecs = []
        for k in keys:
            qs = list(self.sc.facts[k].questions)
            if not qs:
                qs = [k]
            qvecs.append(self.embedder.encode(qs))
        self.keys = keys
        self.qvecs = qvecs
        self.build_ms = (time.perf_counter() - t0) * 1000
        return self.build_ms

    def _ensure_built(self) -> None:
        if not self.qvecs:
            self.build()

    @staticmethod
    def has_negation(text: str) -> bool:
        return bool(NEGATION_RE.search(text.lower()))

    def _scores(self, q: np.ndarray) -> np.ndarray:
        out = np.empty(len(self.keys), dtype=np.float32)
        for i, mat in enumerate(self.qvecs):
            out[i] = float((mat @ q).max()) if len(mat) else 0.0
        return out

    def match(self, text: str) -> MatchResult:
        self._ensure_built()
        if not text.strip() or not self.keys:
            return MatchResult()

        t0 = time.perf_counter()
        q = self.embedder.encode_one(text.strip())
        encode_ms = (time.perf_counter() - t0) * 1000

        t1 = time.perf_counter()
        scores = self._scores(q)
        order = np.argsort(-scores)
        best_i = int(order[0])
        best = float(scores[best_i])
        second_i = int(order[1]) if len(order) > 1 else None
        second = float(scores[second_i]) if second_i is not None else 0.0
        key = self.keys[best_i]
        gap = best - second
        negated = self.has_negation(text)

        if best < self.threshold:
            status, keys = "reject", []
        elif best < self.min_confidence:
            status, keys = "ambiguous", []
        elif gap >= self.margin or best >= self.strong:
            # сильный top-1 принимаем даже при узком отрыве
            keys = [key]
            # второй тоже сильный и почти рядом — оба (что+где горит)
            if (second_i is not None
                    and second >= self.strong
                    and gap < self.margin
                    and self.keys[second_i] not in keys):
                keys.append(self.keys[second_i])
            status = "accept"
        else:
            status, keys = "ambiguous", []

        if negated and status == "accept":
            low = text.lower()
            questionish = (
                "?" in text
                or any(w in low for w in ("ли ", "какой", "какая", "какие",
                                          "где", "что", "сколько", "как вас",
                                          "назовите", "скажите", "есть ли"))
            )
            if not questionish:
                status, keys = "reject", []

        rank_ms = (time.perf_counter() - t1) * 1000
        return MatchResult(
            keys=keys,
            score=best,
            second=second,
            status=status,
            negated=negated,
            encode_ms=encode_ms,
            rank_ms=rank_ms,
        )

    def understand(self, text: str, context=None):
        """Контракт trainer Engine: -> Understanding."""
        from caller.contracts import Act, Timing, Understanding
        from caller.understand import detect_act

        del context  # прототип контекст/эллипсис не использует
        act = detect_act(text)
        timing = Timing(nlu_path="semantic", llm_status="skip")
        words = len(text.split())

        if act is Act.SPEECH_ACT:
            timing.lexical_ms = 0.0
            timing.nlu_path = "act"
            return Understanding(
                [], act, 1.0, words, source=self.name, topical=0, timing=timing,
            )

        m = self.match(text)
        timing.lexical_ms = m.total_ms
        timing.nlu_path = "semantic"
        return Understanding(
            list(m.keys), act, m.score, words,
            source=self.name, topical=words, timing=timing,
        )
