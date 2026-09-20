# -*- coding: utf-8 -*-
"""Арбитр: выбор из пяти кандидатов и устойчивость к недоступной модели."""

from dispatcher.data.loader import load_all
from dispatcher.nlu.arbiter import NullArbiter
from dispatcher.nlu.bank import LexicalBank
from dispatcher.nlu.cascade import Cascade, Thresholds

LOADED = load_all()
BANK = LexicalBank.build(LOADED.questions, LOADED.sources)


class StubArbiter:
    """Всегда выбирает первого кандидата и запоминает, о чём спросили."""

    def __init__(self):
        self.seen = []

    def choose(self, text, slots):
        self.seen.append((text, list(slots)))
        return slots[0] if slots else None


def make_cascade(arbiter=None, **kw):
    sc = LOADED.scenarios["bilet04_call01"]
    return sc, Cascade(scenario=sc, lexical=BANK, arbiter=arbiter, without=sc.id, **kw)


def test_stub_turns_grey_zone_to_dont_know():
    _, c = make_cascade(
        NullArbiter(), thresholds=Thresholds(floor=0.0, gap_high=0.99, gap_low=0.0)
    )
    u = c.understand("на каком этаже горит")
    assert u.source == "arbiter" and not u.keys


def test_arbiter_gets_at_most_five_candidates():
    stub = StubArbiter()
    _, c = make_cascade(
        stub, thresholds=Thresholds(floor=0.0, gap_high=0.99, gap_low=0.0)
    )
    c.understand("на каком этаже горит")
    assert stub.seen and len(stub.seen[0][1]) <= 5


def test_arbiter_pick_becomes_scenario_fact():
    stub = StubArbiter()
    sc, c = make_cascade(
        stub, thresholds=Thresholds(floor=0.0, gap_high=0.99, gap_low=0.0)
    )
    u = c.understand("на каком этаже горит")
    assert u.source == "arbiter"
    assert u.keys and all(k in sc.facts for k in u.keys)


def test_arbiter_skips_confident_answers():
    """Уверенный ответ идёт мимо арбитра: платить за модель незачем."""
    stub = StubArbiter()
    _, c = make_cascade(stub)
    u = c.understand("назовите ваш номер телефона")
    assert u.source == "bank" and not stub.seen
    assert u.keys == ["caller.phone"]

    # ---------------------------------------------- спасение бедных слотов


def thin(sc):
    return {s for s in sc.slots if BANK.population.get(s, 0) <= 40}

    # банк на этой реплике колеблется: разрыв до второго кандидата мал


DISPUTED = "что с домом"
# а на этой уверен
CLEAR = "назовите ваш номер телефона"


def test_arbiter_asked_on_disputed_utterance():
    stub = StubArbiter()
    sc, c = make_cascade(stub, thin_rescue=True)
    assert thin(sc), "в сценарии нет бедных слотов, тест бессмыслен"
    c.understand(DISPUTED)
    assert stub.seen, "арбитра не спросили"


def test_confident_bank_not_given_to_arbiter():
    """В живом сеансе арбитр трижды перебил верный ответ банка про подъезд,
    отвечая про этаж, причём уверенно. Уверенность банка — единственное,
    что от этого защищает."""
    stub = StubArbiter()
    _, c = make_cascade(stub, thin_rescue=True)
    c.understand(CLEAR)
    assert not stub.seen, "уверенный ответ отдали арбитру на перепроверку"


def test_arbiter_sees_rich_slots_too():
    """Иначе он лишён возможности согласиться с банком и вынужден спорить."""
    stub = StubArbiter()
    sc, c = make_cascade(stub, thin_rescue=True)
    c.understand(DISPUTED)
    _text, offered = stub.seen[0]
    rich = [s for s in offered if BANK.population.get(s, 0) > 40]
    assert rich, "показали только бедные — арбитру некуда согласиться"


def test_takeover_only_on_thin_slot():
    """Ответ арбитра принимается, лишь если он указал на бедный слот."""
    sc = LOADED.scenarios["bilet04_call01"]
    rich_slot = max(sc.slots, key=lambda s: BANK.population.get(s, 0))

    class Stubborn:
        def choose(self, text, slots):
            return rich_slot  # всегда тянет на богатый слот

    c = Cascade(
        scenario=sc, lexical=BANK, arbiter=Stubborn(), thin_rescue=True, without=sc.id
    )
    u = c.understand(DISPUTED)
    assert u.source == "bank", "арбитру позволили увести на населённый слот"


def test_no_flag_no_arbiter_call():
    stub = StubArbiter()
    _, c = make_cascade(stub)  # thin_rescue выключен
    c.understand(DISPUTED)
    assert not stub.seen
