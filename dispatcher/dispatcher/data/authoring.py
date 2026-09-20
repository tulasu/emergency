# -*- coding: utf-8 -*-
"""
Заведение новой ситуации.

Главное обещание онтологии: автор пишет только содержание — факты, тексты
ответов, что критично и что выдаётся только по запросу. Формулировки вопросов
он не пишет вообще, их уже дали остальные сценарии корпуса.

Проверить это можно не собирая слепой набор: линтер вычёркивает сценарий из
банка и смотрит, дотянется ли до его фактов остальной корпус.
"""

from __future__ import annotations

from dataclasses import dataclass

from ..nlu.bank import LexicalBank
from ..types import Scenario
from .ontology import Ontology


@dataclass(slots=True)
class FactHealth:
    key: str
    slot: str
    bank_size: int  # формулировок слота без этого сценария
    reachable: float  # доля пробных вопросов, попавших в слот
    stolen_by: str = ""  # кто чаще всего перетягивает


def suggest_slot(
    text: str, bank: LexicalBank, onto: Ontology, k: int = 5
) -> list[tuple[str, str, float]]:
    """Подсказка слота для нового факта: по описанию и ответу.

    Нужна там, где автор не знает, куда положить факт, а заводить новый слот
    рано: сначала стоит посмотреть, нет ли готового.
    """
    out = []
    for slot, score in bank.top(text, k):
        label = onto.slots[slot].label if slot in onto.slots else ""
        out.append((slot, label, round(score, 3)))
    return out


def probe_questions(
    slots: set[str], bank: LexicalBank, without: str, limit: int = 12
) -> list[str]:
    """Чужие формулировки этих слотов — ими и проверяем достижимость."""
    seen: set[str] = set()
    out: list[str] = []
    for e in bank.entries:
        if e.slot not in slots or e.source == without:
            continue
        key = " ".join(sorted(e.lemmas))
        if key in seen:
            continue
        seen.add(key)
        out.append(e.text)
        if len(out) >= limit:
            break
    return out


def lint(sc: Scenario, bank: LexicalBank, onto: Ontology) -> list[FactHealth]:
    """Дотянется ли корпус до фактов этого сценария, не видя его самого.

    Для каждого факта берутся формулировки его слота из других сценариев и
    прогоняются через банк с вычеркнутым текущим сценарием. Если попаданий
    мало — слот беден или его перетягивает сосед, и автору стоит дописать
    несколько формулировок именно сюда.
    """
    scope = sc.slots
    report: list[FactHealth] = []

    for fact in sc.facts.values():
        # факт отвечает и за свой слот, и за те, что сценарий на него замещает:
        # «назовите адрес» при отсутствии addr.full уходит в addr.street
        own = {fact.slot} | {
            missing
            for missing, stand_in in sc.answers_for.items()
            if stand_in == fact.slot
        }
        probes = probe_questions(own, bank, sc.id)
        if not probes:
            report.append(FactHealth(fact.key, fact.slot, 0, 0.0, ""))
            continue

        hits = 0
        thieves: dict[str, int] = {}
        for q in probes:
            ranked = bank.top(q, 1, allowed=scope, without=sc.id)
            if ranked and ranked[0][0] in own:
                hits += 1
            elif ranked:
                thieves[ranked[0][0]] = thieves.get(ranked[0][0], 0) + 1

        worst = max(thieves, key=thieves.get) if thieves else ""
        report.append(
            FactHealth(
                fact.key, fact.slot, len(probes), round(hits / len(probes), 2), worst
            )
        )

    report.sort(key=lambda h: (h.reachable, h.bank_size))
    return report


def render_lint(sc: Scenario, report: list[FactHealth], onto: Ontology) -> str:
    lines = [
        f"{sc .id }: {len (report )} фактов",
        "",
        f"{'факт':<26} {'слот':<22} {'банк':>5} {'достижим':>9}  перетягивает",
    ]
    for h in report:
        mark = "  <-- допишите формулировки" if h.reachable < 0.5 else ""
        lines.append(
            f"{h .key :<26} {h .slot :<22} {h .bank_size :>5} "
            f"{h .reachable :>8.0%}  {h .stolen_by }{mark }"
        )

    weak = [h for h in report if h.reachable < 0.5]
    lines += ["", f"слабых фактов: {len (weak )} из {len (report )}"]
    if not weak:
        lines.append(
            "сценарий целиком покрыт формулировками корпуса — "
            "дописывать вопросы не нужно"
        )
    return "\n".join(lines)
