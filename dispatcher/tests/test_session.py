# -*- coding: utf-8 -*-
"""
Сессия: потоковый и текстовый вход обязаны давать одно и то же.

Если эти два режима расходятся, текстовые тесты перестают что-либо значить
для реального звонка.
"""

from dispatcher.data.loader import load_all
from dispatcher.nlu.bank import LexicalBank
from dispatcher.nlu.cascade import Cascade
from dispatcher.session import Session
from dispatcher.types import Style

LOADED = load_all()
BANK = LexicalBank.build(LOADED.questions, LOADED.sources)
SID = "bilet04_call01"

DIALOG = [
    "Служба 112, что у вас случилось?",
    "На каком этаже горит?",
    "Пострадавшие есть?",
    "Назовите адрес",
    "Как вас зовут?",
]


def make_session(**kw):
    sc = LOADED.scenarios[SID]
    return Session.open(sc, Cascade(scenario=sc, lexical=BANK, **kw), seed=7)


def test_greeting_keeps_question():
    s = make_session()
    reply = s.say("Служба 112, что у вас случилось?")
    assert reply.style is not Style.ACK
    assert "горит" in reply.text.lower()


def test_stream_matches_text():
    by_text = make_session()
    by_stream = make_session()
    for phrase in DIALOG:
        a = by_text.say(phrase).text
        for i in range(6, len(phrase), 5):  # частичные гипотезы STT
            by_stream.on_partial(phrase[:i])
        b = by_stream.on_final(phrase).text
        assert a == b, f"разошлись на «{phrase }»: {a !r } != {b !r }"


def test_hypothesis_keeps_dialog_state():
    s = make_session()
    s.on_partial("на каком эта")
    s.on_partial("на каком этаже гор")
    assert not s.turns, "частичная гипотеза не должна порождать ход"
    assert s.cascade.state.last_slots == () and s.cascade.state.turn == 0
    s.on_final("на каком этаже горит")
    assert len(s.turns) == 1


def test_barge_in_drops_speculation():
    s = make_session()
    s.say("На каком этаже горит?")
    s.on_partial("а сколько всего эта")
    s.cancel()
    assert s._draft is None and s._draft_text == ""


def test_unknown_question_invents_no_fact():
    s = make_session()
    reply = s.say("Какой государственный номер автомобиля?")
    assert reply.style is Style.DONT_KNOW
    assert not s.turns[-1].decision.reveal


def test_contextual_refinement_uses_previous_answer():
    s = make_session()
    s.say("Назовите адрес")
    reply = s.say("А что рядом находится?")
    assert "библиотек" in reply.text.lower()


def test_reask_with_foreign_number_corrected():
    s = make_session()
    s.say("На каком этаже горит?")
    s.say("А сколько всего этажей в доме?")
    reply = s.say("То есть на четырнадцатом горит?")
    assert reply.style is Style.CORRECT
    assert "тринадцат" in reply.text.lower()


def test_answer_never_leaves_scenario():
    """Тексты берутся только из билета — заявитель не может ничего сочинить."""
    s = make_session()
    from dispatcher.dialog import render

    allowed_texts = {
        a for f in LOADED.scenarios[SID].facts.values() for a in f.answers.values()
    }
    allowed_texts |= {t for d in render.GENERIC.values() for v in d.values() for t in v}
    allowed_texts |= set(render.SLOW_DOWN.values())
    for phrase in DIALOG + ["Какой госномер?", "Бригада выехала"]:
        reply = s.say(phrase)
        chunks = [k.strip() for k in reply.text.split(". ") if k.strip()]
        for chunk in chunks:
            assert any(
                chunk.rstrip(".!?") in variant
                or variant.rstrip(".!?") in chunk
                or "записали" in chunk
                for variant in allowed_texts
            ), f"чужой текст: {chunk !r }"


def test_journal_recorded():
    s = make_session()
    for p in DIALOG:
        s.say(p)
    assert len(s.turns) == len(DIALOG)
    assert all(t.understanding.latency_ms >= 0 for t in s.turns)
