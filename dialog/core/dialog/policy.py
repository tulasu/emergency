# -*- coding: utf-8 -*-
"""
Поведение заявителя: что раскрыть и в какой манере.

Чистая функция состояния. Не знает ни текста реплики, ни того, как её
разобрали, ни текстов ответов — поэтому проверяется без моделей и без банка.
"""

from __future__ import annotations

from ..types import Act, Decision, Disclosure, Mood, Scenario, Style, Understanding
from .state import CallState


def decide(u: Understanding, st: CallState, sc: Scenario) -> Decision:
    pr = sc.profile
    st.turn += 1

    # 1. оператор не спрашивает, а сообщает: «бригада выехала», «ждите»
    if u.act is Act.SPEECH_ACT:
        st.mood = Mood(max(Mood.COMPOSED.value, st.mood.value - 1))
        st.idle_turns = 0
        return Decision(style=Style.ACK, mood=st.mood)

    # 2. ничего не разобрали
    if not u.keys:
        st.idle_turns += 1
        _drift(st, pr)
        # реплика без единого смыслового слова — плохая связь, а не незнание.
        # «а что?» — не расслышала; «какой у вас размер обуви» — не знаю
        garbled = u.topical == 0 and not u.slots
        return Decision(
            style=Style.MISHEAR if garbled else Style.DONT_KNOW,
            mood=st.mood,
            unprompted=_initiative(st, sc, pr),
        )

    # 3. предусловия не выполнены: заявитель ещё не может этого знать
    ready = [k for k in u.keys if all(r in st.revealed for r in sc.facts[k].requires)]
    if not ready:
        st.idle_turns += 1
        _drift(st, pr)
        return Decision(style=Style.DONT_KNOW, mood=st.mood)

    st.idle_turns = 0
    for key in ready:
        st.asked[key] = st.asked.get(key, 0) + 1

    # 4. оператор назвал не то число — заявитель поправляет
    if u.value_mismatch and u.act in (Act.ASSERT, Act.CONFIRM):
        st.revealed.update(ready[:1])
        st.last_reveal = tuple(ready[:1])
        return Decision(ready[:1], Style.CORRECT, st.mood)

    # 5. переспрос: «то есть тринадцатый, верно?»
    if u.act is Act.CONFIRM:
        st.revealed.update(ready[:1])
        st.last_reveal = tuple(ready[:1])
        return Decision(ready[:1], Style.CONFIRM, st.mood)

    # 6. залп вопросов разом либо повтор уже отвеченного
    overflow = len(ready) > pr.max_facts_per_turn
    take = ready[: pr.max_facts_per_turn]
    repeated = all(st.asked.get(k, 0) > 1 for k in take)
    st.revealed.update(take)
    st.last_reveal = tuple(take)

    style = Style.SLOW_DOWN if overflow else Style.SHORT if repeated else Style.PLAIN
    return Decision(take, style, st.mood, _initiative(st, sc, pr))


def _drift(st: CallState, pr) -> None:
    """Оператор буксует — заявитель начинает нервничать."""
    if (
        st.idle_turns >= pr.idle_turns_to_worry
        and st.mood.value < pr.mood_ceiling.value
    ):
        st.mood = Mood(st.mood.value + 1)
        st.idle_turns = 0


def _initiative(st: CallState, sc: Scenario, pr) -> str | None:
    """Критичный факт долго не спрашивают — заявитель напоминает сам.

    Уточнения «в скобках» сюда не попадают никогда: вся суть в том, что
    заявитель называет их только по прямому вопросу, и обучающийся обязан
    догадаться спросить.
    """
    if st.turn - st.last_initiative < pr.initiative_cooldown:
        return None
    if st.turn < pr.critical_wait:
        return None
    pending = [
        k
        for k in sc.critical
        if k not in st.revealed and sc.facts[k].disclosure is Disclosure.VOLUNTEERED
    ]
    if not pending:
        return None
    st.last_initiative = st.turn
    return pending[0]
