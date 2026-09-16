# -*- coding: utf-8 -*-
"""
Этап 5: оценка. Журнал ходов и разбор занятия.

Главная метрика — не «сколько реплик система поняла», а сколько
обязательных фактов обучающийся сумел добыть. «Не знаю» и «повторите»
обучающемуся не в минус: заявитель имеет право не знать.
"""
from __future__ import annotations

from dataclasses import dataclass, field

from .contracts import Style, Turn
from .scenario import Scenario


@dataclass
class Report:
    turns: int = 0
    obtained: list[str] = field(default_factory=list)
    missed_critical: list[str] = field(default_factory=list)
    prompted_by_caller: list[str] = field(default_factory=list)
    dont_know: int = 0
    mishear: int = 0
    slow_down: int = 0
    corrections: int = 0
    unrecognized: list[str] = field(default_factory=list)

    def score(self, sc: Scenario) -> int:
        """0..100. Обязательные факты весят всё, штрафы мелкие."""
        need = sc.critical or list(sc.facts)
        got = len([k for k in need if k in self.obtained])
        base = 100 * got / len(need)
        base -= 10 * len(set(self.prompted_by_caller))   # напомнил заявитель
        base -= 3 * self.corrections                # записал неверно
        base -= 2 * self.slow_down                  # частил вопросами
        return max(0, min(100, round(base)))


class Journal:
    def __init__(self, sc: Scenario):
        self.sc = sc
        self.turns: list[Turn] = []

    def add(self, turn: Turn) -> None:
        self.turns.append(turn)

    def report(self) -> Report:
        r = Report(turns=len(self.turns))
        for t in self.turns:
            r.obtained += [k for k in t.decision.reveal if k not in r.obtained]
            if t.decision.unprompted:
                r.prompted_by_caller.append(t.decision.unprompted)
            st = t.decision.style
            r.dont_know += st is Style.DONT_KNOW
            r.mishear += st is Style.MISHEAR
            r.slow_down += st is Style.SLOW_DOWN
            r.corrections += st is Style.CORRECT
            if st in (Style.DONT_KNOW, Style.MISHEAR):
                r.unrecognized.append(t.utterance)
        r.missed_critical = [k for k in self.sc.critical if k not in r.obtained]
        return r

    def render(self) -> str:
        r = self.report()
        need = self.sc.critical or list(self.sc.facts)
        lines = [
            f"Ходов: {r.turns}   Оценка: {r.score(self.sc)}/100",
            f"Обязательные факты: {len(need) - len(r.missed_critical)}/{len(need)}",
        ]
        if r.missed_critical:
            lines.append("  не выяснено: " + ", ".join(r.missed_critical))
        if r.prompted_by_caller:
            lines.append("  напомнил заявитель: " + ", ".join(r.prompted_by_caller))
        if r.corrections:
            lines.append(f"  неверно записано и исправлено заявителем: {r.corrections}")
        lines.append(f"Всего фактов получено: {len(r.obtained)}/{len(self.sc.facts)}")
        lines.append(f"«не знаю»: {r.dont_know}   «повторите»: {r.mishear}"
                     f"   «не так быстро»: {r.slow_down}")
        return "\n".join(lines)
