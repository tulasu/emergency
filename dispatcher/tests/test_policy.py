# -*- coding: utf-8 -*-
"""
Поведение заявителя проверяется без моделей и без банка.

Политике на вход подаётся готовое понимание, поэтому тесты не зависят ни от
распознавания, ни от текстов ответов.
"""

import pytest

from dispatcher.data.loader import DATA, load_scenario
from dispatcher.data.ontology import Ontology
from dispatcher.dialog.policy import decide
from dispatcher.dialog.state import CallState
from dispatcher.types import Act, Mood, Style, Understanding

ONTO = Ontology.load()
SC = load_scenario(DATA / "scenarios" / "bilet04_call01.json", ONTO)


def U(keys=(), slots=None, act=Act.ASK, words=4, mismatch=False, topical=2):
    return Understanding(
        slots=list(slots if slots is not None else keys),
        keys=list(keys),
        act=act,
        score=0.9,
        words=words,
        topical=topical,
        value_mismatch=mismatch,
    )


def test_speech_act_acknowledged():
    d = decide(U(act=Act.SPEECH_ACT), CallState(), SC)
    assert d.style is Style.ACK and not d.reveal


def test_clear_unknown_question_means_dont_know():
    # слот распознан, но у этого заявителя такого факта нет
    d = decide(U(keys=(), slots=["vehicle.plate"]), CallState(), SC)
    assert d.style is Style.DONT_KNOW


def test_gibberish_means_repeat():
    d = decide(U(words=1, topical=0), CallState(), SC)
    assert d.style is Style.MISHEAR


def test_dont_know_differs_from_mishear():
    confused = decide(U(words=1, topical=0), CallState(), SC).style
    unknown = decide(U(keys=(), slots=["vehicle.plate"]), CallState(), SC).style
    assert confused is not unknown


def test_burst_of_questions_trimmed():
    d = decide(U(["addr.street", "victim.present", "caller.name"]), CallState(), SC)
    assert d.style is Style.SLOW_DOWN
    assert len(d.reveal) == SC.profile.max_facts_per_turn


def test_repeated_question_answered_dryer():
    st = CallState()
    decide(U(["caller.name"]), st, SC)
    assert decide(U(["caller.name"]), st, SC).style is Style.SHORT


def test_wrong_number_corrected():
    d = decide(U(["fire.floor"], act=Act.CONFIRM, mismatch=True), CallState(), SC)
    assert d.style is Style.CORRECT


def test_reask_confirmed():
    d = decide(U(["fire.floor"], act=Act.CONFIRM), CallState(), SC)
    assert d.style is Style.CONFIRM


def test_prerequisite_blocks_fact():
    # ориентир называется только после того, как названа улица
    d = decide(U(["addr.landmark"]), CallState(), SC)
    assert d.style is Style.DONT_KNOW and not d.reveal


def test_prerequisite_lifted_after_answer():
    st = CallState()
    decide(U(["addr.street"]), st, SC)
    assert decide(U(["addr.landmark"]), st, SC).style is Style.PLAIN


def test_mood_rises_when_stuck():
    st = CallState()
    for _ in range(SC.profile.idle_turns_to_worry):
        decide(U(words=1, topical=0), st, SC)
    assert st.mood is Mood.WORRIED


def test_mood_drops_after_ack():
    st = CallState(mood=Mood.WORRIED)
    decide(U(act=Act.SPEECH_ACT), st, SC)
    assert st.mood is Mood.COMPOSED


def test_mood_capped_by_profile():
    st = CallState()
    for _ in range(40):
        decide(U(words=1, topical=0), st, SC)
    assert st.mood.value <= SC.profile.mood_ceiling.value


def test_caller_reminds_of_critical_fact():
    st = CallState()
    seen = [decide(U(words=1, topical=0), st, SC).unprompted for _ in range(12)]
    assert any(seen), "о критичном факте так и не напомнили"


def test_bracketed_refinement_not_volunteered():
    # этажность дома — уточнение из билета, обучающийся обязан спросить сам
    st = CallState()
    reminded = {decide(U(words=1, topical=0), st, SC).unprompted for _ in range(30)}
    assert "building.floors" not in reminded


@pytest.mark.parametrize("profile,expected", [("calm", 2), ("hard", 1)])
def test_profile_changes_facts_per_turn(profile, expected):
    from dispatcher.types import Profile

    sc = load_scenario(DATA / "scenarios" / "bilet04_call01.json", ONTO)
    sc.profile = Profile.preset(profile)
    d = decide(U(["addr.street", "victim.present", "caller.name"]), CallState(), sc)
    assert len(d.reveal) == expected
