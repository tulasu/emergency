# -*- coding: utf-8 -*-
"""
Этап 3: решение. Что раскрыть и в какой манере.

Не знает ни текста реплики, ни того, как её распознали, ни текстов ответов.
Чистая функция состояния — тестируется без модели и без банка.
"""
from __future__ import annotations

from dataclasses import dataclass, field

from .contracts import Act, Decision, Mood, Style, Understanding
from .scenario import Profile, Scenario


@dataclass
class CallerState:
    revealed: set[str] = field(default_factory=set)
    turn: int = 0
    idle_turns: int = 0
    mood: Mood = Mood.COMPOSED
    last_initiative: int = -99


def decide(u: Understanding, st: CallerState, sc: Scenario) -> Decision:
    pr: Profile = sc.profile
    st.turn += 1

    # 1. оператор успокоил или сообщил о выезде
    if u.act is Act.SPEECH_ACT:
        st.mood = Mood(max(Mood.COMPOSED.value, st.mood.value - 1))
        st.idle_turns = 0
        return Decision(style=Style.ACK, mood=st.mood)

    # 2. ничего не поняли: искажение речи или вопрос вне сценария
    if not u.keys:
        st.idle_turns += 1
        _drift(st, pr)
        # искажение = смысловых слов нет вовсе. «как грыжа?» — это «не знаю»,
        # а не «не расслышала»
        garbled = u.topical == 0
        return Decision(style=Style.MISHEAR if garbled else Style.DONT_KNOW,
                        mood=st.mood, unprompted=_initiative(st, sc, pr))

    # 3. предусловия ещё не выполнены — заявитель не может это знать
    ready = [k for k in u.keys
             if all(r in st.revealed for r in sc.facts[k].requires)]
    if not ready:
        st.idle_turns += 1
        _drift(st, pr)
        return Decision(style=Style.DONT_KNOW, mood=st.mood)

    st.idle_turns = 0

    # 4. оператор назвал неверное значение — поправляем
    if u.value_mismatch and u.act in (Act.ASSERT, Act.CONFIRM):
        st.revealed.update(ready[:1])
        return Decision(ready[:1], Style.CORRECT, st.mood)

    # 5. переспрос
    if u.act is Act.CONFIRM:
        st.revealed.update(ready[:1])
        return Decision(ready[:1], Style.CONFIRM, st.mood)

    # 6. слишком много вопросов разом / уже отвечали
    overflow = len(ready) > pr.max_facts_per_turn
    take = ready[:pr.max_facts_per_turn]
    repeated = all(k in st.revealed for k in take)
    st.revealed.update(take)

    style = (Style.SLOW_DOWN if overflow else
             Style.SHORT if repeated else Style.PLAIN)
    return Decision(take, style, st.mood, _initiative(st, sc, pr))


def _drift(st: CallerState, pr: Profile) -> None:
    if st.idle_turns >= pr.idle_turns_to_worry and st.mood.value < pr.mood_ceiling.value:
        st.mood = Mood(st.mood.value + 1)
        st.idle_turns = 0


def _initiative(st: CallerState, sc: Scenario, pr: Profile) -> str | None:
    """Заявитель сам напоминает о критичном факте, который долго не спрашивают."""
    if st.turn - st.last_initiative < pr.initiative_cooldown:
        return None
    if st.turn < pr.critical_wait:
        return None
    pending = [k for k in sc.critical if k not in st.revealed]
    if not pending:
        return None
    st.last_initiative = st.turn
    return pending[0]
