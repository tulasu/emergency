# -*- coding: utf-8 -*-
"""
Корпус: 96 сценариев должны раскладываться по онтологии без остатка.

Этот файл — страховка при расширении. Новая ситуация или правка онтологии,
после которой хоть один факт остался без слота, валит сборку здесь.
"""

from collections import Counter

from dispatcher.data.loader import load_all
from dispatcher.types import Disclosure

LOADED = load_all()


CANON = 96  # билетов С-112: 32 билета по три вызова
CANON_FACTS = 810


def test_canon_corpus_intact():
    """Свои сценарии добавлять можно, исходные ломать нельзя.

    Система создана для расширения, поэтому счёт «ровно 96» был бы вредным
    требованием: он валил бы тесты у любого, кто завёл свою ситуацию.
    """
    own = {k: v for k, v in LOADED.scenarios.items() if k.startswith("bilet")}
    assert len(own) == CANON
    assert sum(len(s.facts) for s in own.values()) == CANON_FACTS


def test_all_facts_mapped_to_slots():
    unmapped = [i for i in LOADED.issues if i.kind == "unmapped"]
    assert not unmapped, "без слота остались: " + ", ".join(
        f"{i .scenario }:{i .key }" for i in unmapped[:10]
    )


def test_every_fact_links_existing_slot():
    from dispatcher.data.ontology import Ontology

    onto = Ontology.load()
    for sc in LOADED.scenarios.values():
        for fact in sc.facts.values():
            assert fact.slot in onto.slots, f"{sc .id }:{fact .key } -> {fact .slot }"


def test_slot_usually_holds_one_fact():
    sizes = Counter(
        len(v) for sc in LOADED.scenarios.values() for v in sc.by_slot.values()
    )
    # больше трёх фактов на слот — ответ рассыпается, оператор не запишет
    assert max(sizes) <= 3, sizes


def test_bank_richer_than_per_fact():
    # ради этого и заводилась онтология: медиана на слот должна быть заметно
    # больше девяти формулировок, которые автор писал руками на каждый факт
    import statistics

    per_slot = statistics.median(len(v) for v in LOADED.questions.values())
    assert per_slot >= 20, per_slot


def test_bank_phrasings_unique():
    for slot, qs in LOADED.questions.items():
        norm = [" ".join(q.lower().replace("ё", "е").split()).strip(" ?.!") for q in qs]
        assert len(norm) == len(set(norm)), slot


def test_bracketed_refinements_only_on_request():
    # главная учебная механика: 21 вызов из билетов, где сведение в скобках
    on_request = {
        (sid, f.key)
        for sid, sc in LOADED.scenarios.items()
        for f in sc.facts.values()
        if f.disclosure is Disclosure.ON_REQUEST
    }
    assert len(on_request) >= 25
    assert ("bilet04_call01", "building.floors") in on_request
    assert ("bilet04_call01", "fire.floor") not in on_request


def test_critical_keys_link_existing_facts():
    for sc in LOADED.scenarios.values():
        for key in sc.critical:
            assert key in sc.facts, f"{sc .id }: критичный {key }"


def test_prerequisites_link_existing_facts():
    for sc in LOADED.scenarios.values():
        for fact in sc.facts.values():
            for req in fact.requires:
                assert req in sc.facts, f"{sc .id }:{fact .key } требует {req }"
