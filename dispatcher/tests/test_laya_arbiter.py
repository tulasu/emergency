# -*- coding: utf-8 -*-
"""
Арбитр на laya проверяется без самой модели.

Важно не то, что модель угадывает, а что обёртка правильно переводит метки
обратно в слоты, уважает порог уверенности и не роняет звонок, когда модель
недоступна.
"""

import pytest

from dispatcher.data.ontology import Ontology
from dispatcher.nlu.laya import NOTHING
from dispatcher.nlu.laya_arbiter import LayaArbiter

ONTO = Ontology.load()
CANDIDATES = ["fire.floor", "building.floors", "caller.name"]


class StubRouter:
    """Отдаёт заранее заданный выбор и запоминает, что ему передали."""

    def __init__(self, choice, confidence=0.9, explode=False):
        self.choice, self.confidence, self.explode = choice, confidence, explode
        self.last = None

    def predict(self, state, questions):
        if self.explode:
            raise RuntimeError("модель недоступна")
        self.last = (state, questions)
        return {
            "answers": {"слот": {"choice": self.choice, "confidence": self.confidence}}
        }


def make_arbiter(**kw):
    return LayaArbiter(ontology=ONTO, router=StubRouter(**kw))


def test_label_maps_back_to_slot():
    a = make_arbiter(choice=ONTO.slots["fire.floor"].label)
    assert a.choose("на каком этаже горит", CANDIDATES) == "fire.floor"
    assert a.calls == 1


def test_nothing_option_means_refusal():
    a = make_arbiter(choice=NOTHING)
    assert a.choose("какая марка машины", CANDIDATES) is None
    assert a.refusals == 1


def test_low_confidence_means_refusal():
    a = make_arbiter(choice=ONTO.slots["fire.floor"].label, confidence=0.2)
    assert a.choose("что-то невнятное", CANDIDATES) is None
    assert a.refusals == 1


def test_dead_model_keeps_call_alive():
    a = make_arbiter(choice="неважно", explode=True)
    assert a.choose("на каком этаже", CANDIDATES) is None
    assert a.failures == 1 and a.calls == 0


def test_no_candidates_no_model_call():
    a = make_arbiter(choice="неважно")
    assert a.choose("что угодно", []) is None
    assert a.router.last is None


def test_nothing_option_always_offered():
    a = make_arbiter(choice=NOTHING)
    a.choose("вопрос", CANDIDATES)
    _state, questions = a.router.last
    assert NOTHING in questions["слот"]["criteria"]
    assert len(questions["слот"]["criteria"]) == len(CANDIDATES) + 1


def test_criterion_carries_sample_phrasings():
    a = make_arbiter(choice=NOTHING)
    a.choose("вопрос", CANDIDATES)
    _state, questions = a.router.last
    text = questions["слот"]["criteria"][ONTO.slots["fire.floor"].label]
    assert ONTO.slots["fire.floor"].label in text


def test_clashing_labels_disambiguated():
    """Два слота с одинаковой меткой не должны затирать друг друга."""
    one_of_two = ["fire.floor", "fire.floor"]
    a = make_arbiter(choice=ONTO.slots["fire.floor"].label)
    assert a.choose("вопрос", one_of_two) == "fire.floor"
    _state, questions = a.router.last
    assert len(questions["слот"]["criteria"]) == 3  # два слота плюс «ни о чём»


@pytest.mark.parametrize("threshold,expected", [(0.5, "fire.floor"), (0.95, None)])
def test_confidence_threshold_tunable(threshold, expected):
    a = LayaArbiter(
        ontology=ONTO,
        min_confidence=threshold,
        router=StubRouter(ONTO.slots["fire.floor"].label, 0.9),
    )
    assert a.choose("на каком этаже", CANDIDATES) == expected
