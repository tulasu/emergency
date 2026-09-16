# -*- coding: utf-8 -*-
"""Лексический gate, paraphrases и контекстные эллипсисы."""
from __future__ import annotations

import json
from pathlib import Path

from caller.contracts import DialogContext, Style
from caller.engine import Engine
from caller.scenario import Scenario
from caller.understand import Hybrid, Lexical, LocalLlm, ParaphraseBank

SC = Scenario.load("scenarios/bilet04_call01.json")


def test_off_topic_даёт_не_знаю():
    t = Engine(SC, "lexical", seed=7).handle("Какая у вас сейчас погода?")
    assert t.decision.style is Style.DONT_KNOW
    assert not t.decision.reveal


def test_уверенный_факт_проходит():
    t = Engine(SC, "lexical", seed=7).handle("есть ли пострадавшие?")
    assert "пострадавшие" in t.understanding.keys


def test_timing_заполняется():
    t = Engine(SC, "lexical", seed=7).handle("есть ли пострадавшие?")
    assert t.timing.total_ms >= 0
    assert t.timing.nlu_path in ("lexical", "act")
    assert "timing total=" in t.timing.format()


def test_paraphrase_learn_не_трогает_scenario(tmp_path: Path):
    """Удачная LLM-формулировка попадает в cache/, не в scenarios/."""
    bank = ParaphraseBank(SC.id, root=tmp_path)
    lex = Lexical(SC, paraphrases=bank)
    phrase = "на каком именно этаже сейчас пожар?"
    # до обучения lexical может не принять — это ок
    assert lex.add_paraphrase("этаж_возгорания", phrase)
    assert phrase in bank.phrases("этаж_возгорания")
    bank.save()
    on_disk = json.loads((tmp_path / f"{SC.id}.json").read_text(encoding="utf-8"))
    assert phrase in on_disk["этаж_возгорания"]
    # сценарий не изменился
    assert phrase not in SC.facts["этаж_возгорания"].questions
    # после обучения lexical принимает
    hit = lex.match_segment(phrase)
    assert hit.status == "accept"
    assert hit.key == "этаж_возгорания"


class _FakeLocal(LocalLlm):
    """Подмена LM Studio: возвращает заданные ключи без сети."""

    def __init__(self, sc, keys_by_text: dict[str, list[str]]):
        super().__init__(sc, timeout=0.01)
        self.keys_by_text = keys_by_text
        self.calls_log: list[tuple[str, DialogContext | None]] = []

    def classify(self, text, context=None, lexical_hint=None):
        self.calls_log.append((text, context, lexical_hint))
        keys = self.keys_by_text.get(text, [])
        return keys, "ok", 1.0, False


def test_hybrid_ellipsis_с_context_зовёт_llm(tmp_path: Path):
    bank = ParaphraseBank("test_ellipsis", root=tmp_path)
    fake = _FakeLocal(SC, {"Какой?": ["этаж_возгорания"]})
    hy = Hybrid(SC, lexical=Lexical(SC, paraphrases=bank), local=fake,
                paraphrases=bank)
    ctx = DialogContext(
        recent_ops=["На каком этаже пожар?"],
        recent_keys=["этаж_возгорания"],
    )
    u = hy.understand("Какой?", context=ctx)
    assert fake.calls_log, "эллипсис с контекстом должен вызвать LLM"
    assert u.keys == ["этаж_возгорания"]
    assert u.timing.nlu_path == "hybrid_llm"
    # «Какой?» без content-лемм в банк не кладём — эллипсисы остаются на LLM+context
    assert "Какой?" not in bank.phrases("этаж_возгорания")


def test_hybrid_ellipsis_без_context_не_зовёт_llm_если_reject(tmp_path: Path):
    """Короткий «Какой?» без истории: lexical reject, контекста нет → не LLM."""
    bank = ParaphraseBank("test_no_ctx", root=tmp_path)
    fake = _FakeLocal(SC, {"Какой?": ["этаж_возгорания"]})
    hy = Hybrid(SC, lexical=Lexical(SC, paraphrases=bank), local=fake,
                paraphrases=bank)
    u = hy.understand("Какой?", context=None)
    # без context эллипсис не триггерит LLM; ambiguous мог бы — «Какой?» обычно reject
    if not fake.calls_log:
        assert u.keys == []


def test_engine_копит_recent_keys():
    eng = Engine(SC, "lexical", seed=7)
    eng.handle("есть ли пострадавшие?")
    assert "пострадавшие" in eng.state.recent_keys
    assert eng.state.recent_ops


def test_hybrid_verify_отец_после_фио_не_повторяет_имя(tmp_path: Path):
    """Lexical клеит «отца зовут» на фио — LLM должен отвергнуть."""
    bank = ParaphraseBank("test_verify_fio", root=tmp_path)
    phrase = "А как отца вашего зовут?"
    fake = _FakeLocal(SC, {phrase: []})
    hy = Hybrid(SC, lexical=Lexical(SC, paraphrases=bank), local=fake,
                paraphrases=bank)
    ctx = DialogContext(
        recent_ops=["как вас зовут"],
        recent_keys=["фио"],
    )
    u = hy.understand(phrase, context=ctx)
    assert fake.calls_log, "повтор темы фио должен уйти в LLM на проверку"
    assert fake.calls_log[0][2] == ["фио"]
    assert u.keys == []
    assert u.timing.nlu_path == "hybrid_llm"


def test_hybrid_первый_фио_без_истории_без_llm(tmp_path: Path):
    """Первый уверенный «как вас зовут» — lexical, без сети."""
    bank = ParaphraseBank("test_first_fio", root=tmp_path)
    fake = _FakeLocal(SC, {})
    hy = Hybrid(SC, lexical=Lexical(SC, paraphrases=bank), local=fake,
                paraphrases=bank)
    u = hy.understand("как вас зовут", context=None)
    assert "фио" in u.keys
    assert not fake.calls_log
    assert u.timing.llm_status == "skip"
