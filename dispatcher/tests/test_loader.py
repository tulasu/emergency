# -*- coding: utf-8 -*-
"""Чтение одного сценария."""

from dispatcher.data.loader import DATA, load_scenario
from dispatcher.data.ontology import Ontology
from dispatcher.types import Disclosure, Style

ONTO = Ontology.load()
SC = load_scenario(DATA / "scenarios" / "bilet04_call01.json", ONTO)


def test_scenario_loads():
    assert SC.id == "bilet04_call01"
    assert SC.opening.startswith("Алло")
    assert SC.profile.name == "calm"


def test_fact_found_by_slot():
    facts = SC.facts_for("fire.floor")
    assert [f.key for f in facts] == ["fire.floor"]
    assert "тринадцат" in facts[0].answer(Style.PLAIN)


def test_missing_slot_gives_empty():
    assert SC.facts_for("vehicle.plate") == []


def test_missing_style_falls_back_to_plain():
    fact = SC.facts_for("building.gas")[0]
    assert fact.answer(Style.CORRECT) == fact.answers["plain"]


def test_fact_numbers_parsed():
    assert SC.facts_for("fire.floor")[0].numbers == {13}
    assert SC.facts_for("building.floors")[0].numbers == {14}


def test_volunteered_first():
    fact = SC.facts_for("building.floors")[0]
    assert fact.disclosure is Disclosure.ON_REQUEST


def test_prerequisite_kept():
    assert SC.facts_for("addr.landmark")[0].requires == ("addr.street",)
