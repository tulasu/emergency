# -*- coding: utf-8 -*-
"""Контракт встройки: реестр держит инварианты Session для транспорта."""

import pytest

from dispatcher.service import Service, reply_to_dict
from dispatcher.types import Style

SID = "bilet04_call01"
DIALOG = [
    "Служба 112, что у вас случилось?",
    "На каком этаже горит?",
    "Назовите адрес",
]


@pytest.fixture(scope="module")
def svc():
    return Service()  # лексический каскад: контракт не зависит от модели


def test_open_close_roundtrip(svc):
    sid, opening = svc.open(SID)
    assert opening
    svc.close(sid)
    svc.close(sid)  # идемпотентен
    with pytest.raises(KeyError):
        svc.final(sid, "алло")


def test_unknown_scenario_and_session(svc):
    with pytest.raises(KeyError):
        svc.open("нет-такого-сценария")
    with pytest.raises(KeyError):
        svc.partial("нет-сессии", "алло")
    with pytest.raises(KeyError):
        svc.final("нет-сессии", "алло")


def test_stream_matches_text_via_registry(svc):
    a, _ = svc.open(SID)
    b, _ = svc.open(SID)
    for phrase in DIALOG:
        t = svc.final(a, phrase).text
        for i in range(6, len(phrase), 5):
            svc.partial(b, phrase[:i])
        assert svc.final(b, phrase).text == t, phrase
    svc.close(a)
    svc.close(b)


def test_partial_creates_no_turn_and_cancel_safe(svc):
    sid, _ = svc.open(SID)
    svc.partial(sid, "на каком эта")
    assert not svc.sessions[sid].turns
    assert svc.cancel(sid) is None  # рвать нечего — безопасно
    svc.close(sid)


def test_reply_serializes(svc):
    sid, _ = svc.open(SID)
    d = reply_to_dict(svc.final(sid, "Какой госномер автомобиля?"))
    assert d["style"] == Style.DONT_KNOW.value
    assert set(d) == {"text", "audio_id", "style", "mood"}
    svc.close(sid)


def test_sessions_isolated(svc):
    a, _ = svc.open(SID)
    b, _ = svc.open(SID)
    svc.final(a, "Назовите адрес")
    assert not svc.sessions[b].turns  # состояние звонков не смешивается
    svc.close(a)
    svc.close(b)
