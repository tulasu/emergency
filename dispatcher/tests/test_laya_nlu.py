# -*- coding: utf-8 -*-
"""
Понимание на laya проверяется без модели.

Важно не то, что модель угадывает, а что обёртка собирает варианты из
сценария, переводит выбор обратно в факт и не выдаёт ничего сверх билета.
"""

from dispatcher.data.loader import DATA, load_scenario
from dispatcher.data.ontology import Ontology
from dispatcher.nlu.laya import NOTHING
from dispatcher.nlu.laya_nlu import LayaUnderstander
from dispatcher.types import Act

ONTO = Ontology.load()
SC = load_scenario(DATA / "scenarios" / "bilet04_call01.json", ONTO)


class StubRouter:
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


def make_nlu(**kw):
    return LayaUnderstander(scenario=SC, ontology=ONTO, router=StubRouter(**kw))


def test_options_built_from_scenario():
    n = make_nlu(choice=NOTHING)
    n.understand("вопрос какой-то")
    _state, questions = n.router.last
    criteria = questions["слот"]["criteria"]
    assert len(criteria) == len(SC.by_slot) + 1  # слоты плюс «ни о чём»
    assert NOTHING in criteria


def test_option_described_by_caller_answer():
    """Формулировки вопросов не нужны — в этом весь смысл режима."""
    n = make_nlu(choice=NOTHING)
    n.understand("вопрос")
    _state, questions = n.router.last
    label = ONTO.slots["fire.floor"].label
    assert "тринадцат" in questions["слот"]["criteria"][label].lower()


def test_pick_becomes_scenario_fact():
    n = make_nlu(choice=ONTO.slots["fire.floor"].label)
    u = n.understand("на каком этаже горит")
    assert u.slots == ["fire.floor"]
    assert u.keys == ["fire.floor"]
    assert u.source == "laya"


def test_nothing_option_means_dont_know():
    n = make_nlu(choice=NOTHING)
    u = n.understand("какой у вас размер обуви")
    assert not u.keys and n.refusals == 1


def test_low_confidence_means_dont_know():
    n = make_nlu(choice=ONTO.slots["fire.floor"].label, confidence=0.1)
    u = n.understand("невнятица")
    assert not u.keys and n.refusals == 1


def test_speech_act_by_rules_without_model():
    n = make_nlu(choice="неважно")
    u = n.understand("Бригада уже выехала, оставайтесь на линии")
    assert u.act is Act.SPEECH_ACT and u.source == "rules"
    assert n.router.last is None, "модель дёргать незачем"


def test_wrong_number_flagged():
    n = make_nlu(choice=ONTO.slots["fire.floor"].label)
    u = n.understand("то есть на четырнадцатом горит")
    assert u.value_mismatch, "четырнадцать — это этажность, а не этаж пожара"


def test_right_number_not_an_error():
    n = make_nlu(choice=ONTO.slots["fire.floor"].label)
    assert not n.understand("то есть тринадцатый этаж").value_mismatch


def test_dead_model_keeps_call_alive():
    n = make_nlu(choice="неважно", explode=True)
    u = n.understand("на каком этаже горит")
    assert not u.keys and n.failures == 1


def test_latency_logged():
    n = make_nlu(choice=ONTO.slots["fire.floor"].label)
    assert n.understand("на каком этаже горит").latency_ms >= 0
