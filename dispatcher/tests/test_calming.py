# -*- coding: utf-8 -*-
"""Калибровочный край: «успокоение» с «?» и топом банка — вопрос, без — сообщение."""

from dispatcher.data.loader import load_all
from dispatcher.nlu.bank import LexicalBank
from dispatcher.nlu.cascade import Cascade
from dispatcher.nlu.reranker import RerankerVoter
from dispatcher.types import Act

LOADED = load_all()
BANK = LexicalBank.build(LOADED.questions, LOADED.sources)


def make_cascade():
    sc = LOADED.scenarios["bilet18_call03"]
    return Cascade(scenario=sc, lexical=BANK)


def test_calming_question_with_bank_top_is_ask():
    u = make_cascade()._understand("Не вешайте трубку, хорошо?")
    assert u.act is Act.ASK
    assert u.slots == ["meta.stay_on_line"]


def test_calming_imperative_without_question_stays_speech_act():
    u = make_cascade()._understand("Трубку не вешайте")
    assert u.act is Act.SPEECH_ACT and not u.slots


def test_calming_question_without_bank_top_stays_speech_act():
    # «не паникуйте» нет в банке: топ 0 < calming_min — сообщение, не вопрос
    u = make_cascade()._understand("Не паникуйте?")
    assert u.act is Act.SPEECH_ACT and not u.slots


class StubScorer:
    def __init__(self, scores):
        self.scores = scores

    def score_many(self, text, slots):
        return [self.scores[slot] for slot in slots]


def test_reranker_picks_best_above_thresholds():
    v = RerankerVoter(StubScorer({"a": 0.9, "b": 0.5}))
    assert v.choose("что-то", ["a", "b"]) == "a"
    assert v.calls == 1


def test_reranker_abstains_on_low_score_or_no_margin():
    low = RerankerVoter(StubScorer({"a": 0.1}))
    assert low.choose("что-то", ["a"]) is None
    assert low.refusals == 1
    tight = RerankerVoter(StubScorer({"a": 0.9, "b": 0.89}))
    assert tight.choose("что-то", ["a", "b"]) is None
    assert tight.refusals == 1


def test_reranker_abstains_on_empty_or_dead_scorer():
    assert RerankerVoter(StubScorer({})).choose("что-то", []) is None

    class Dead:
        def score_many(self, text, slots):
            raise RuntimeError("лёг")

    v = RerankerVoter(Dead())
    assert v.choose("что-то", ["a"]) is None
    assert v.failures == 1
