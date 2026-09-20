# -*- coding: utf-8 -*-
"""
Речь заявителя: решение -> конкретная реплика.

Текст ответа всегда берётся из сценария. Ни модель, ни рендер ничего не
сочиняют — заявитель физически не может выдать сведение, которого в билете
нет. Настроение меняет обрамление, но не содержание.
"""

from __future__ import annotations

import random

from ..data.ontology import Ontology
from ..types import Decision, Mood, Reply, Scenario, Style

# Реплики, не привязанные к факту. Индекс — настроение.
GENERIC: dict[Style, dict[Mood, tuple[str, ...]]] = {
    Style.DONT_KNOW: {
        Mood.COMPOSED: (
            "Не знаю, не видела.",
            "Затрудняюсь сказать.",
            "Этого я не знаю.",
        ),
        Mood.WORRIED: ("Не знаю я! Не смотрела.", "Откуда мне знать?"),
        Mood.TENSE: ("Да не знаю я!", "Я не разглядывала, мне не до того!"),
    },
    Style.MISHEAR: {
        Mood.COMPOSED: ("Что? Повторите, пожалуйста.", "Не расслышала."),
        Mood.WORRIED: ("Что вы говорите? Повторите!", "Плохо слышно, ещё раз."),
        Mood.TENSE: ("Не слышу вас! Громче!",),
    },
    Style.ACK: {
        Mood.COMPOSED: ("Хорошо, поняла.", "Да, поняла вас."),
        Mood.WORRIED: ("Хорошо... я жду.", "Поняла, только быстрее."),
        Mood.TENSE: ("Ладно, только быстрее, пожалуйста!",),
    },
}

SLOW_DOWN: dict[Mood, str] = {
    Mood.COMPOSED: "Подождите, не так быстро.",
    Mood.WORRIED: "Помедленнее, я не успеваю!",
    Mood.TENSE: "Да подождите вы, сразу всё спрашиваете!",
}

# Метка слота подставляется после запятой: она бывает длинной («есть ли
# пострадавшие»), и во фразе «Вы ... записали?» такая метка ломает строй
URGE: dict[Mood, str] = {
    Mood.COMPOSED: "Вы записали, {what}?",
    Mood.WORRIED: "Вы запишите, {what}!",
    Mood.TENSE: "Вы вообще записали, {what}?!",
}


class Renderer:
    def __init__(
        self, sc: Scenario, onto: Ontology | None = None, seed: int | None = None
    ):
        self.sc = sc
        self.onto = onto or Ontology.load()
        self.rnd = random.Random(seed)

    def say(self, d: Decision) -> Reply:
        parts: list[str] = []
        ids: list[str] = []

        if d.style in GENERIC:
            # один бросок на пару (текст, файл): иначе текст скажет одно,
            # а предрендер сыграет другое
            variants = GENERIC[d.style][d.mood]
            i = self.rnd.randrange(len(variants))
            parts.append(variants[i])
            ids.append(f"common/{d.style.value}/{d.mood.name.lower()}_{i}.wav")
        else:
            for key in d.reveal:
                fact = self.sc.facts[key]
                parts.append(fact.answer(d.style))
                hit = fact.audio.get(d.style.value) or fact.audio.get("plain")
                if hit:
                    ids.append(hit)
            if d.style is Style.SLOW_DOWN:
                parts.append(SLOW_DOWN[d.mood])
                ids.append(f"common/slow_down/{d.mood.name.lower()}.wav")

        if d.unprompted:
            parts.append(URGE[d.mood].format(what=self._name(d.unprompted)))
            fact = self.sc.facts.get(d.unprompted)
            slot = fact.slot if fact else ""
            if slot:
                ids.append(f"common/urge/{d.mood.name.lower()}/{slot}.wav")

        # композит клеит плеер конкатенацией (8 кГц mono, шаг 4); без файлов
        # в индексе — None, плеер идёт через живой TTS по тексту
        return Reply(
            " ".join(p for p in parts if p),
            "+".join(ids) or None,
            d.style,
            d.mood,
        )

    def _name(self, key: str) -> str:
        """Как заявитель называет сведение, о котором сам напоминает.

        Берётся из онтологии, а не из захардкоженного списка: новый слот
        получает имя автоматически.
        """
        fact = self.sc.facts.get(key)
        if not fact:
            return "это"
        slot = self.onto.slots.get(fact.slot)
        if not slot:
            return "это"
        return slot.urge or slot.label
