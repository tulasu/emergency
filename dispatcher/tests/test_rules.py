# -*- coding: utf-8 -*-
"""Правила: что оператор сделал репликой."""

import pytest

from dispatcher.nlu.rules import (
    detect_act,
    is_refinement,
    is_repeat_request,
    speech_act,
)
from dispatcher.types import Act


@pytest.mark.parametrize(
    "text,kind",
    [
        ("Бригада уже выехала к вам", "выезд"),
        ("Оставайтесь на линии, пожалуйста", "успокоение"),
        ("Не подходите близко к огню", "инструкция"),
        ("Служба 112, слушаю вас", "приветствие"),
    ],
)
def test_speech_acts(text, kind):
    assert speech_act(text) == kind
    assert detect_act(text) is Act.SPEECH_ACT


@pytest.mark.parametrize(
    "text",
    [
        "На каком этаже горит?",
        "Назовите адрес",
        "Пострадавшие есть",
        "Сколько этажей в доме",
        "Вы видите пламя",
        "Представьтесь, пожалуйста",
    ],
)
def test_questions_detected(text):
    assert detect_act(text) is Act.ASK


@pytest.mark.parametrize(
    "text",
    [
        "То есть тринадцатый этаж, верно?",
        "Я записал, улица Грина, правильно?",
        "Значит пострадавших нет",
    ],
)
def test_reask_detected(text):
    assert detect_act(text) is Act.CONFIRM


def test_assertion_not_a_question():
    assert detect_act("Записываю ваш адрес") is Act.CONFIRM
    assert detect_act("Пострадавших нет") is Act.ASSERT


@pytest.mark.parametrize(
    "text",
    ["хорошо", "я понял все спасибо большое", "ага понятно",
     "я понял спасибо служба уже в пути", "хорошо передаю службам вашу заявку",
     "сто двенадцать говорите"],
)
def test_ack_is_speech_act(text):
    # принято/сообщение — заявителю хватит «да, поняла», а не случайного факта
    assert detect_act(text) is Act.SPEECH_ACT


@pytest.mark.parametrize(
    "text",
    ["хорошо можете сказать адрес дома",
     "алло здравствуйте служба сто двенадцать что у вас случилось",
     "ага хорошо пострадавшие есть"],
)
def test_ack_lead_keeps_question(text):
    assert detect_act(text) is not Act.SPEECH_ACT


@pytest.mark.parametrize(
    "text",
    [
        "А корпус?",
        "а номер дома",
        "Точнее?",
        "поточнее скажите",
        "А какой именно?",
        "и подъезд",
    ],
)
def test_refinement_hangs_on_previous_answer(text):
    assert is_refinement(text)


@pytest.mark.parametrize(
    "text",
    [
        "А что случилось?",
        "Назовите адрес",
        "Где именно горит, на каком этаже",
        "А где вы находитесь сейчас",
        "",
    ],
)
def test_standalone_utterance_not_refinement(text):
    assert not is_refinement(text)


def test_repeat_request():
    assert is_repeat_request("Повторите, пожалуйста")
    assert is_repeat_request("Ещё раз, не расслышал")
    assert not is_repeat_request("Назовите адрес")


@pytest.mark.parametrize(
    "text",
    [
        "Наряд куда подъезжать?",
        "Куда бригаде ехать?",
        "Какая бригада приедет?",
    ],
)
def test_crew_question_not_dispatch_report(text):
    """«Наряд куда подъезжать?» — вопрос оператора, а не объявление."""
    assert speech_act(text) is None
    assert detect_act(text) is Act.ASK


@pytest.mark.parametrize(
    "text",
    [
        "Бригада выехала",
        "Наряд направлен",
        "Скорая выслана",
        "Бригада уже в пути",
    ],
)
def test_dispatch_report_detected(text):
    assert speech_act(text) == "выезд"
