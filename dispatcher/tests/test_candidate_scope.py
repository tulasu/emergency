# -*- coding: utf-8 -*-
"""Продовый top-5 ограничен сценарием, но сохраняет голос каскада."""

from dispatcher.data.loader import load_all
from dispatcher.nlu.bank import LexicalBank
from dispatcher.nlu.cascade import Cascade
from dispatcher.nlu.ensemble import Ensemble


def test_candidates_scope_and_cascade_own_vote():
    loaded = load_all()
    sc = loaded.scenarios["bilet04_call01"]
    bank = LexicalBank.build(loaded.questions, loaded.sources)
    cascade = Cascade(sc, bank)
    text = "Какая сейчас погода?"

    assert "env.weather" not in sc.slots
    assert cascade.candidates(text)[0] == "env.weather"
    scoped = cascade.candidates(text, scope="scenario")
    assert scoped and len(scoped) <= 5 and all(s in sc.slots for s in scoped)

    class Capture:
        def choose(self, text, slots):
            self.slots = slots
            return None

    voter = Capture()
    Ensemble(cascade, [voter], candidate_scope="scenario").understand(text)
    assert voter.slots[0] == "env.weather"  # прежний голос каскада возвращён
    assert all(s in sc.slots for s in voter.slots[1:])

    Ensemble(cascade, [voter]).understand(text)
    assert voter.slots == cascade.candidates(text)  # Service по умолчанию не изменён
