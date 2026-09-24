# -*- coding: utf-8 -*-
"""
Ответ вне сценария: вопрос понят, но сведения у заявителя нет.

Вместо дежурного «не знаю» маленькая LLM (llama-server) отвечает от лица
заявителя, по-человечески и по ситуации: «вы собственник?» — «нет, я
соседка», «рядом кто-то есть?» — «да, соседи вышли на площадку».

Главный риск тренажёра — выдуманный факт в карточке обучающегося. Поэтому:
  * модель видит только вводную заявителя и то, что он УЖЕ сказал, —
    нераскрытые факты сценария ей не показываются, иначе она их выболтает;
  * ответ проверяется: без цифр и чисел словами, без имён и названий,
    которых нет в контексте, не длиннее двух коротких фраз;
  * не прошёл проверку или сервер молчит — остаётся дежурное «не знаю».
"""

from __future__ import annotations

import json
import os
import re
import urllib.request
from dataclasses import dataclass

from ..nlu.normalize import CARDINALS
from ..types import Mood, Scenario, Turn

LLM_URL = os.environ.get("DISPATCHER_LLM_URL", "http://127.0.0.1:8081")

# манера, а не слова: модель не должна говорить «я взволнован» вслух
_MOOD = {
    Mood.COMPOSED: "Манера: спокойно.",
    Mood.WORRIED: "Манера: встревоженно, но по делу. Не называй свои чувства.",
    Mood.TENSE: "Манера: отрывисто, на нервах. Не называй свои чувства.",
}

_SYSTEM = (
    "Ты — заявитель, который звонит в службу 112. Оператор спросил о том, "
    "чего в твоих сведениях НЕТ, — значит, ты этого не знаешь.\n"
    "Ответь одной короткой разговорной фразой, как по телефону:\n"
    "- если ответ прямо следует из того, что ты знаешь (кем ты приходишься, "
    "что уже рассказал), — ответь этим;\n"
    "- иначе скажи, что не знаешь, и коротко объясни почему, исходя из "
    "ситуации: не видел, не спрашивал, не заходил, не успел разглядеть.\n"
    "Нельзя: придумывать новые подробности о людях, месте и происшествии; "
    "называть адреса, имена, номера, числа, время и возраст.\n"
    "Если оператор не спрашивает, а сообщает, — коротко согласись.\n"
    "Не повторяй свои прошлые ответы — отвечай именно на этот вопрос.\n"
    "Только сама реплика, без кавычек и пояснений."
)

_NUMBER_WORDS = frozenset(CARDINALS) | {"тысяча", "тысячи", "тысяч", "сот", "полтора"}
# время и возраст — тоже факты: «около часа дня» заявитель знать не может
_TIME_WORDS = frozenset("""
час часа часов минут минуты минуту секунд утра утром вечера вечером ночи
ночью полдень полночь вчера позавчера неделю недели месяц месяца год года
лет летний летняя полчаса получаса пол
""".split())
_WORD = re.compile(r"[А-ЯЁA-Zа-яёa-z]+")


@dataclass
class Improviser:
    url: str = LLM_URL
    timeout: float = 3.0
    max_words: int = 22
    calls: int = 0
    rejected: int = 0
    failures: int = 0

    def reply(self, sc: Scenario, revealed: set[str], question: str,
              turns: list[Turn], mood: Mood) -> str | None:
        self.calls += 1
        known = [sc.opening] + [sc.facts[k].answers.get("plain", "")
                                for k in sorted(revealed) if k in sc.facts]
        history = []
        for t in turns[-3:]:
            history += [f"Оператор: {t.utterance}", f"Ты: {t.reply.text}"]
        user = (
            "Что ты знаешь и уже сказал:\n- " + "\n- ".join(k for k in known if k)
            + ("\n\nНачало разговора:\n" + "\n".join(history) if history else "")
            + f"\n\n{_MOOD.get(mood, '')}\nОператор: {question}\nТы:"
        )
        body = {
            "messages": [{"role": "system", "content": _SYSTEM},
                         {"role": "user", "content": user}],
            "max_tokens": 40,
            "temperature": 0.4,
            "chat_template_kwargs": {"enable_thinking": False},
        }
        try:
            req = urllib.request.Request(
                self.url + "/v1/chat/completions", data=json.dumps(body).encode(),
                headers={"Content-Type": "application/json"})
            with urllib.request.urlopen(req, timeout=self.timeout) as r:
                text = json.loads(r.read())["choices"][0]["message"]["content"]
        except Exception:  # noqa: BLE001 — нет сервера: остаётся «не знаю»
            self.failures += 1
            return None
        text = " ".join(text.replace("«", "").replace("»", "").replace('"', "").split())
        said = {" ".join(t.reply.text.lower().split()) for t in turns}
        if text.lower() in said or not self.grounded(text, " ".join(known + history)):
            self.rejected += 1
            return None
        return text

    def grounded(self, text: str, context: str) -> bool:
        """Нет ли в ответе того, чего заявитель знать не может."""
        if not text or re.search(r"\d", text):
            return False
        words = _WORD.findall(text)
        if not words or len(words) > self.max_words:
            return False
        if any(w.lower() in _NUMBER_WORDS or w.lower() in _TIME_WORDS for w in words):
            return False
        ctx = {w.lower() for w in _WORD.findall(context)}
        # имя или название с большой буквы не в начале фразы — только из контекста
        for sentence in re.split(r"[.!?…]+", text):
            for w in _WORD.findall(sentence)[1:]:
                if w[0].isupper() and w.lower() not in ctx:
                    return False
        return True
