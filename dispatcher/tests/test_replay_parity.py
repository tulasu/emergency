# -*- coding: utf-8 -*-
"""Шаг 8. Паритет транспорта с эталоном: replay-файл через реестр Service
обязан дать то же, что Session.say (== cli replay --stream == replay)."""

from pathlib import Path

from dispatcher.service import Service

SID = "bilet04_call01"


def _lines() -> list[str]:
    return [l.strip() for l in
            Path("examples/dialog.txt").read_text(encoding="utf-8").splitlines()
            if l.strip() and not l.startswith("#")]


def test_replay_file_via_registry_matches_direct():
    svc = Service()
    a, opening_a = svc.open(SID, seed=7)
    b, opening_b = svc.open(SID, seed=7)
    assert opening_a == opening_b and opening_a
    for line in _lines():
        direct = svc.final(a, line).text
        for i in range(6, len(line), 5):  # --stream: частичные гипотезы
            svc.partial(b, line[:i])
        assert svc.final(b, line).text == direct, line
        assert direct  # заявитель не молчит ни на одной реплике
    assert len(svc.sessions[a].turns) == len(svc.sessions[b].turns) == len(_lines())
    svc.close(a)
    svc.close(b)
