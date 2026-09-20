# -*- coding: utf-8 -*-
"""Шаг 2: audio_id резолвится из Fact.audio, композиты — через '+'."""

from dispatcher.data.loader import load_all
from dispatcher.dialog.render import Renderer
from dispatcher.types import Decision, Mood, Style

SID = "bilet04_call01"


def test_fact_audio_plain_and_fallback():
    loaded = load_all()
    sc = loaded.scenarios[SID]
    key = next(iter(sc.by_slot["fire.floor"]))
    sc.facts[key].audio["plain"] = "a/x.wav"
    r = Renderer(sc, seed=7)
    got = r.say(Decision([key], Style.PLAIN, Mood.COMPOSED))
    assert got.audio_id == "a/x.wav"
    short = r.say(Decision([key], Style.SHORT, Mood.COMPOSED))
    assert short.audio_id == "a/x.wav"  # fallback на plain


def test_generic_audio_matches_text_atomically():
    loaded = load_all()
    sc = loaded.scenarios[SID]
    r1 = Renderer(sc, seed=7)
    r2 = Renderer(sc, seed=7)
    a = r1.say(Decision([], Style.DONT_KNOW, Mood.COMPOSED))
    b = r2.say(Decision([], Style.DONT_KNOW, Mood.COMPOSED))
    assert a.text == b.text and a.audio_id == b.audio_id
    assert a.audio_id.startswith("common/dont_know/composed_")


def test_slow_down_and_urge_append():
    loaded = load_all()
    sc = loaded.scenarios[SID]
    key = next(iter(sc.by_slot["fire.floor"]))
    sc.facts[key].audio["plain"] = "a/x.wav"
    r = Renderer(sc, seed=7)
    slow = r.say(Decision([key, key], Style.SLOW_DOWN, Mood.COMPOSED))
    assert slow.audio_id.startswith("a/x.wav+a/x.wav+common/slow_down/")
    crit = sc.critical[0]
    urge = r.say(Decision([key], Style.PLAIN, Mood.COMPOSED, unprompted=crit))
    slot = sc.facts[crit].slot
    assert urge.audio_id == f"a/x.wav+common/urge/composed/{slot}.wav"


def test_no_audio_files_no_audio_id():
    loaded = load_all()  # без data/audio/index.json
    sc = loaded.scenarios[SID]
    r = Renderer(sc, seed=7)
    key = next(iter(sc.by_slot["fire.floor"]))
    assert r.say(Decision([key], Style.PLAIN, Mood.COMPOSED)).audio_id is None
