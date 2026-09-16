# -*- coding: utf-8 -*-
"""
Этап 4: речь. Решение -> конкретная реплика и ссылка на готовое аудио.

Текст ответа берётся из сценария, поэтому заявитель не может ничего выдумать.
Настроение меняет только обрамление, не содержание.
"""
from __future__ import annotations

import random

from .contracts import Decision, Mood, Reply, Style
from .scenario import Scenario

# Реплики, не привязанные к конкретному факту. Индекс — настроение.
GENERIC: dict[Style, dict[Mood, list[str]]] = {
    Style.DONT_KNOW: {
        Mood.COMPOSED: ["Не знаю, не видела.", "Затрудняюсь сказать.",
                        "Этого я не знаю."],
        Mood.WORRIED: ["Не знаю я! Не смотрела.", "Откуда мне знать?"],
        Mood.TENSE: ["Да не знаю я!", "Я не разглядывала, мне не до того!"],
    },
    Style.MISHEAR: {
        Mood.COMPOSED: ["Что? Повторите, пожалуйста.", "Не расслышала."],
        Mood.WORRIED: ["Что вы говорите? Повторите!", "Плохо слышно, ещё раз."],
        Mood.TENSE: ["Не слышу вас! Громче!"],
    },
    Style.ACK: {
        Mood.COMPOSED: ["Хорошо, поняла.", "Да, поняла вас."],
        Mood.WORRIED: ["Хорошо... я жду.", "Поняла, только быстрее."],
        Mood.TENSE: ["Ладно, только быстрее, пожалуйста!"],
    },
}

SLOW_DOWN = {
    Mood.COMPOSED: "Подождите, не так быстро.",
    Mood.WORRIED: "Помедленнее, я не успеваю!",
    Mood.TENSE: "Да подождите вы, сразу всё спрашиваете!",
}

URGE = {
    Mood.COMPOSED: "Вы {what} записали?",
    Mood.WORRIED: "Вы {what}-то запишите!",
    Mood.TENSE: "Вы {what} вообще записали?!",
}

# как назвать критичный факт, когда заявитель напоминает о нём сам
URGE_NAMES = {"адрес": "адрес", "пострадавшие": "про людей",
              "телефон": "телефон", "фио": "мою фамилию"}


class Speaker:
    def __init__(self, sc: Scenario, seed: int | None = None):
        self.sc = sc
        self.rnd = random.Random(seed)

    def say(self, d: Decision) -> Reply:
        parts, audio = [], None

        if d.style in GENERIC:
            parts.append(self.rnd.choice(GENERIC[d.style][d.mood]))
        else:
            for key in d.reveal:
                fact = self.sc.facts[key]
                parts.append(fact.answer(d.style))
                audio = audio or fact.audio.get(d.style.value)
            if d.style is Style.SLOW_DOWN:
                parts.append(SLOW_DOWN[d.mood])

        if d.unprompted:
            name = next((v for k, v in URGE_NAMES.items() if k in d.unprompted),
                        "это")
            parts.append(URGE[d.mood].format(what=name))

        return Reply(" ".join(p for p in parts if p), audio)
