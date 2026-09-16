# -*- coding: utf-8 -*-
"""
Этап 2: понимание. Реплика -> ключи фактов + речевой акт.

Две взаимозаменяемые реализации за одним интерфейсом:
  Lexical — леммы и веса, работает офлайн, потолок около 25%;
  Llm     — классификатор с кэшем, около 90%, нужен доступ к модели.

Llm при недоступности модели молча падает на Lexical, поэтому демо
никогда не встаёт колом.
"""
from __future__ import annotations

import json
import math
import os
import re
import urllib.request
from collections import Counter
from pathlib import Path
from typing import Protocol

from . import morph
from .contracts import Act, Understanding
from .scenario import Scenario

# --------------------------------------------------------------- речевые акты

ACT_PATTERNS = {
    "выезд": r"\b(выехал|выезжа|направ|бригад|помощь ид|уже ед)",
    "успокоение": r"\b(успокой|не паник|оставайтесь|не кладите|не вешайте|ждите)",
    "инструкция": r"\b(не подходите|не пытайтесь|не включайте|не курите|"
                  r"отойдите|выйдите|перекройте|не тушите|встречайте|уходите)",
    "приветствие": r"(служба 112|единая служба|служба спасения|слушаю)",
    "прощание": r"\b(до свидания|всего доброго|спасибо, ждите)",
}
CONFIRM_MARKERS = ("верно", "правильно", "значит", "то есть", "уточняю",
                   "подтверд", "точно")
QUESTION_STARTERS = {"есть", "видеть", "слышать", "мочь", "знать",
                     "находиться", "гореть", "далеко", "живой"}
IMPERATIVE_ASK = {"представиться", "назвать", "сказать", "уточнить",
                  "сообщить", "описать", "продиктовать", "повторить"}


def speech_act(text: str) -> str | None:
    low = text.lower()
    for kind, pat in ACT_PATTERNS.items():
        if re.search(pat, low):
            return kind
    return None


def detect_act(text: str) -> Act:
    low = text.lower()
    if speech_act(low):
        return Act.SPEECH_ACT
    ws = morph.words(low)
    lemmas = {morph.lemma(w) for w in ws}
    is_q = ("?" in text
            or bool(lemmas & {"что", "где", "когда", "как", "какой", "сколько",
                              "кто", "чей"})
            or " ли " in f" {low} "
            or bool(lemmas & IMPERATIVE_ASK)
            or (ws and morph.lemma(ws[0]) in QUESTION_STARTERS))
    if any(m in low for m in CONFIRM_MARKERS):
        return Act.CONFIRM
    return Act.ASK if is_q else Act.ASSERT


# ------------------------------------------------------------------ интерфейс

class Understander(Protocol):
    name: str

    def understand(self, text: str) -> Understanding: ...


# ------------------------------------------------------------- лексический

class Lexical:
    """Пересечение лемм, взвешенное обратной частотой по банку сценария."""
    name = "lexical"

    def __init__(self, sc: Scenario, threshold: float = 0.34):
        self.sc = sc
        self.threshold = threshold
        self.index = {k: [morph.content_lemmas(q) for q in f.questions]
                      for k, f in sc.facts.items()}
        df = Counter()
        n = 0
        for variants in self.index.values():
            for s in variants:
                df.update(s)
                n += 1
        self.idf = {w: math.log(1 + n / c) for w, c in df.items()}
        self.default = max(self.idf.values(), default=1.0)

    def _w(self, s: set[str]) -> float:
        return sum(self.idf.get(x, self.default) for x in s)

    def _score(self, q: set[str], key: str) -> float:
        best = 0.0
        for variant in self.index[key]:
            inter = self._w(q & variant)
            total = self._w(q) + self._w(variant)
            if total:
                best = max(best, 2 * inter / total)
        return best

    def understand(self, text: str) -> Understanding:
        act = detect_act(text)
        ws = morph.words(text)
        if act is Act.SPEECH_ACT:
            return Understanding([], act, 1.0, len(ws), source=self.name)

        found, conf = [], 0.0
        for seg in morph.segments(text):
            q = morph.content_lemmas(seg)
            if not q:
                continue
            ranked = sorted(((self._score(q, k), k) for k in self.index),
                            reverse=True)
            if ranked and ranked[0][0] >= self.threshold:
                score, key = ranked[0]
                conf = max(conf, score)
                if key not in found:
                    found.append(key)

        mismatch = self._mismatch(text, found)
        return Understanding(found, act, conf, len(ws), mismatch, self.name)

    def _mismatch(self, text: str, keys: list[str]) -> bool:
        said = morph.numbers(text)
        if not said:
            return False
        for k in keys:
            vn = self.sc.facts[k].value_numbers
            if vn and not (said & vn):
                return True
        return False


# -------------------------------------------------------- модель + кэш

SYSTEM = """Ты классифицируешь реплику оператора службы 112.

Заявитель знает только перечисленные ниже сведения. Определи, о каких из них
спрашивает оператор.

Сведения:
{catalog}

Верни ТОЛЬКО JSON, без пояснений:
{{"keys": ["ключ", ...]}}

Правила:
- если оператор спрашивает о том, чего в списке нет, верни пустой список;
- если в одной реплике несколько вопросов, перечисли все подходящие ключи;
- не выдумывай ключей, которых нет в списке."""


class Llm:
    """Классификатор ключей. Ответ модели — список ключей, не текст реплики.

    Кэш по нормализованной фразе: каждая новая формулировка стоит один
    вызов за всё время жизни системы.
    """
    name = "llm"
    URL = "https://api.anthropic.com/v1/messages"
    MODEL = "claude-sonnet-4-6"

    def __init__(self, sc: Scenario, cache_path: str = "cache/understand.json",
                 fallback: Understander | None = None):
        self.sc = sc
        self.fallback = fallback or Lexical(sc)
        self.key = os.environ.get("ANTHROPIC_API_KEY")
        self.cache_path = Path(cache_path)
        self.cache: dict[str, list[str]] = {}
        if self.cache_path.exists():
            self.cache = json.loads(self.cache_path.read_text(encoding="utf-8"))
        self.catalog = "\n".join(
            f"- {k}: {f.questions[0]}" for k, f in sc.facts.items())
        self.calls = 0
        self.hits = 0

    def _norm(self, text: str) -> str:
        return " ".join(sorted(morph.content_lemmas(text))) or text.lower().strip()

    def _ask_model(self, text: str) -> list[str] | None:
        if not self.key:
            return None
        body = json.dumps({
            "model": self.MODEL, "max_tokens": 200, "temperature": 0,
            "system": SYSTEM.format(catalog=self.catalog),
            "messages": [{"role": "user", "content": text}],
        }).encode()
        req = urllib.request.Request(self.URL, data=body, headers={
            "content-type": "application/json",
            "anthropic-version": "2023-06-01",
            "x-api-key": self.key,
        })
        try:
            with urllib.request.urlopen(req, timeout=20) as r:
                data = json.loads(r.read())
            raw = "".join(c.get("text", "") for c in data.get("content", []))
            keys = json.loads(re.sub(r"```json|```", "", raw).strip())["keys"]
            self.calls += 1
            return [k for k in keys if k in self.sc.facts]
        except Exception:
            return None

    def understand(self, text: str) -> Understanding:
        act = detect_act(text)
        ws = morph.words(text)
        if act is Act.SPEECH_ACT:
            return Understanding([], act, 1.0, len(ws), source="act")

        norm = self._norm(text)
        if norm in self.cache:
            self.hits += 1
            keys, src = self.cache[norm], "cache"
        else:
            keys = self._ask_model(text)
            if keys is None:
                return self.fallback.understand(text)
            self.cache[norm] = keys
            src = "llm"

        mismatch = Lexical._mismatch(self, text, keys)
        return Understanding(keys, act, 1.0 if keys else 0.0, len(ws),
                             mismatch, src)

    def save(self) -> None:
        self.cache_path.parent.mkdir(parents=True, exist_ok=True)
        self.cache_path.write_text(
            json.dumps(self.cache, ensure_ascii=False, indent=1), encoding="utf-8")


def build(sc: Scenario, kind: str = "lexical") -> Understander:
    return Llm(sc) if kind == "llm" else Lexical(sc)
