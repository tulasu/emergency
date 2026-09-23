# -*- coding: utf-8 -*-
"""
Реестр звонков: session_id -> Session.

Исполняемая фиксация контракта из docs/session-contract.md:
один SIP-звонок = один Session.open. Тяжёлое (онтология, банк, энкодер,
векторы, арбитр) строится раз на процесс, Cascade/Session — на каждый звонок
(NluState мутабелен, шарить нельзя). Поверх лягут gRPC/HTTP (шаг 6) и
Go-оркестратор (шаг 5) — они будут только проксировать эти пять методов.
"""

from __future__ import annotations

import uuid
from dataclasses import dataclass, field

from .data.loader import Loaded, load_all
from .data.ontology import Ontology
from .nlu.bank import LexicalBank
from .session import Session
from .types import Reply


@dataclass
class Service:
    """Тонкая обёртка над Session для транспорта. Параметры — как cli._build."""

    model: str | None = None
    backend: str = "auto"
    device: str | None = None
    arbiter_kind: str | None = None
    thin_rescue: bool = False
    ensemble: str | None = None  # majority | llm-lead … — голосование e5 + laya + LLM
    improv: bool = False  # ответы вне сценария от LLM

    onto: Ontology = field(init=False)
    loaded: Loaded = field(init=False)
    bank: LexicalBank = field(init=False)
    encoder: object = field(init=False, default=None)
    vectors: object = field(init=False, default=None)
    arbiter: object = field(init=False, default=None)
    voters: list = field(init=False, default_factory=list)
    improviser: object = field(init=False, default=None)
    sessions: dict[str, Session] = field(init=False, default_factory=dict)

    def __post_init__(self) -> None:
        from .nlu.cascade import Cascade  # noqa: F401  (тип каскада на звонок)

        self.onto = Ontology.load()
        self.loaded = load_all(self.onto)
        self.bank = LexicalBank.build(self.loaded.questions, self.loaded.sources)

        if self.model:
            from .nlu import encoder as enc
            from .nlu.vectors import VectorBank

            self.encoder = enc.build(
                self.model,
                backend=self.backend,
                device=self.device
                or (
                    "cuda"
                    if self.backend == "torch" and enc.available_device() == "cuda"
                    else "cpu"
                ),
            )
            self.vectors = VectorBank.build(
                self.loaded.questions, self.loaded.sources, self.encoder
            )

        if self.arbiter_kind == "laya":
            from .nlu.laya_arbiter import LayaArbiter

            self.arbiter = LayaArbiter(
                ontology=self.onto,
                device=self.device or "cpu",
                min_confidence=0.80,
            )

        if self.ensemble:
            from .nlu.laya_arbiter import LayaArbiter
            from .nlu.llm_arbiter import LlmArbiter

            # у голосующего laya порог ниже, чем у арбитра: её «не уверена»
            # превращается в воздержание, а решает большинство
            self.voters = [
                self.arbiter if self.arbiter is not None else LayaArbiter(
                    ontology=self.onto, device=self.device or "cpu",
                    min_confidence=0.5),
                LlmArbiter(self.onto),
            ]
        if self.improv:
            from .dialog.improv import Improviser

            self.improviser = Improviser()

    # ------------------------------------------------------------- 5 методов

    def open(self, scenario_id: str, session_id: str | None = None,
             seed: int | None = None) -> tuple[str, str]:
        """Ответ на звонок. Возвращает (session_id, opening)."""
        from .nlu.cascade import Cascade

        if scenario_id not in self.loaded.scenarios:
            raise KeyError(f"нет сценария {scenario_id}")
        sc = self.loaded.scenarios[scenario_id]
        cascade = Cascade(
            scenario=sc,
            lexical=self.bank,
            ontology=self.onto,
            encoder=self.encoder,
            vectors=self.vectors,
            arbiter=self.arbiter,
            thin_rescue=self.thin_rescue and self.arbiter is not None,
        )
        understander = cascade
        if self.voters:
            from .nlu.ensemble import Ensemble

            understander = Ensemble(cascade, self.voters, rule=self.ensemble)
        sid = session_id or uuid.uuid4().hex
        self.sessions[sid] = Session.open(sc, understander, self.onto, seed)
        self.sessions[sid].improv = self.improviser
        return sid, self.sessions[sid].opening()

    def _get(self, session_id: str) -> Session:
        try:
            return self.sessions[session_id]
        except KeyError:
            raise KeyError(f"нет сессии {session_id}") from None

    def partial(self, session_id: str, text: str) -> None:
        self._get(session_id).on_partial(text)

    def final(self, session_id: str, text: str) -> Reply:
        return self._get(session_id).on_final(text)

    def cancel(self, session_id: str) -> str | None:
        return self._get(session_id).cancel()

    def close(self, session_id: str) -> None:
        s = self.sessions.pop(session_id, None)
        if s is not None:
            s.close()


def reply_to_dict(r: Reply) -> dict:
    """Сериализация Reply для gRPC/HTTP — формат из контракта."""
    return {
        "text": r.text,
        "audio_id": r.audio_id,
        "style": r.style.value,
        "mood": int(r.mood.value),
    }
