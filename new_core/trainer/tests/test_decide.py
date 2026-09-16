# -*- coding: utf-8 -*-
"""Поведение заявителя тестируется без модели и без банка вопросов."""
from caller.contracts import Act, Mood, Style, Understanding
from caller.decide import CallerState, decide
from caller.scenario import Scenario

SC = Scenario.load("scenarios/bilet04_call01.json")
U = lambda keys=(), act=Act.ASK, words=4, mm=False, topical=2: Understanding(
    list(keys), act, 0.9, words, mm, topical=topical)


def test_вопрос_вне_сценария_даёт_не_знаю():
    assert decide(U(words=4), CallerState(), SC).style is Style.DONT_KNOW


def test_короткая_искажённая_реплика_даёт_повторите():
    assert decide(U(words=1, topical=0), CallerState(), SC).style is Style.MISHEAR


def test_много_вопросов_разом_урезается():
    d = decide(U(["адрес_улица", "пострадавшие", "фио"]), CallerState(), SC)
    assert d.style is Style.SLOW_DOWN and len(d.reveal) == 2


def test_повторный_вопрос_отвечается_суше():
    st = CallerState(revealed={"фио"})
    assert decide(U(["фио"]), st, SC).style is Style.SHORT


def test_неверное_число_исправляется():
    d = decide(U(["этаж_возгорания"], Act.ASSERT, mm=True), CallerState(), SC)
    assert d.style is Style.CORRECT


def test_предусловие_блокирует_факт():
    d = decide(U(["ориентир"]), CallerState(), SC)
    assert d.style is Style.DONT_KNOW and not d.reveal


def test_настроение_растёт_от_буксования_и_падает_от_успокоения():
    st = CallerState()
    for _ in range(3):
        decide(U(words=4), st, SC)
    assert st.mood is Mood.WORRIED
    decide(U(act=Act.SPEECH_ACT), st, SC)
    assert st.mood is Mood.COMPOSED


def test_настроение_не_превышает_потолок_профиля():
    st = CallerState()
    for _ in range(20):
        decide(U(words=4), st, SC)
    assert st.mood.value <= SC.profile.mood_ceiling.value


def test_заявитель_напоминает_о_критичном_факте():
    st = CallerState()
    outs = [decide(U(["газификация"]), st, SC) for _ in range(8)]
    assert any(o.unprompted for o in outs)
