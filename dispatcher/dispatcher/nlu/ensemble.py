# -*- coding: utf-8 -*-
"""
Коллективное решение: e5-каскад + laya + маленькая LLM.

Правила (речевые акты, «повторите», уточнение «а корпус?») решают сами —
их голосованием не портим. На вопросе каскад отдаёт своё понимание и топ
кандидатов по всем слотам; laya и LLM выбирают из тех же кандидатов
(каждый в своём порядке). Решает правило `rule`:

  majority   слот, за который хотя бы двое из трёх; иначе — каскад
  strict     слот, за который хотя бы двое; иначе — отказ («не знаю»)
  cascade+   каскад, но если laya и LLM вдвоём против него — их слот
  llm-lead   последний голосующий (LLM) решает, если двое других не сошлись

Голос «ни о чём» (None) тоже голос: двое «ни о чём» против каскада — отказ.
"""

from __future__ import annotations

import time
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from typing import Protocol

from ..types import Act, Understanding
from . import rules
from .cascade import Cascade

_NONE = "<ни о чём>"


class Voter(Protocol):
    def choose(self, text: str, slots: list[str]) -> str | None: ...


@dataclass
class Ensemble:
    cascade: Cascade
    voters: list[Voter]
    rule: str = "majority"
    k: int = 5
    last_votes: dict = field(default_factory=dict)  # для журнала звонка
    _pool: ThreadPoolExecutor = field(init=False, repr=False)

    def __post_init__(self) -> None:
        # laya (GPU) и LLM (llama-server) голосуют параллельно
        self._pool = ThreadPoolExecutor(max_workers=max(1, len(self.voters)))

    @property
    def state(self):
        return self.cascade.state

    def understand(self, text: str) -> Understanding:
        t0 = time.perf_counter()
        self.state.turn += 1
        u = self._decide(text, self.cascade._understand(text))
        u.latency_ms = round(1000 * (time.perf_counter() - t0), 3)
        self.state.observe(u.slots)
        return u

    def preview(self, text: str) -> Understanding | None:
        try:
            return self._decide(text, self.cascade._understand(text))
        except Exception:  # noqa: BLE001
            return None

    def save(self) -> None:
        self.cascade.save()

    def _decide(self, text: str, u: Understanding) -> Understanding:
        self.last_votes = {}
        if u.act is Act.SPEECH_ACT or u.source == "rules" or not u.topical:
            return u
        if len(u.slots) > 1:  # два вопроса в реплике — голосование их схлопнет
            return u
        cands = self.cascade.candidates(text, self.k)
        own = u.slots[0] if u.slots else None
        if own and own not in cands:
            cands = [own, *cands[: self.k - 1]]
        if not cands:
            return u
        # голосующим — без «алло, служба сто двенадцать»: иначе «сто
        # двенадцать» тянет их к адресу и номерам, как тянуло каскад
        clean = rules.strip_ritual(text)
        picks = list(self._pool.map(lambda v: v.choose(clean, cands), self.voters))
        votes = [own or _NONE, *(p or _NONE for p in picks)]
        self.last_votes = {"cascade": votes[0],
                           **{f"v{i}": p for i, p in enumerate(votes[1:], 1)}}
        for i, v in enumerate(self.voters, 1):
            if getattr(v, "last", ""):
                self.last_votes[f"v{i}_raw"] = v.last
        top, n = Counter(votes).most_common(1)[0]
        if self.rule == "majority":
            final = top if n >= 2 else votes[0]
        elif self.rule == "strict":
            final = top if n >= 2 else _NONE
        elif self.rule == "cascade+":
            others = votes[1:]
            final = others[0] if len(others) >= 2 and len(set(others)) == 1 \
                and others[0] != votes[0] else votes[0]
        elif self.rule == "llm-lead":
            final = top if n >= 2 else votes[-1]
        else:
            raise ValueError(f"нет правила {self.rule}")
        slots = [] if final == _NONE else [final]
        if slots == u.slots[:1] and len(u.slots) <= 1:
            return u
        return self.cascade.rewrap(text, u, slots, "ensemble")
