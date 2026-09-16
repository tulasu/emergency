# -*- coding: utf-8 -*-
"""
Этап 2: понимание. Реплика -> ключи фактов + речевой акт.

Реализации за одним интерфейсом:
  Lexical — леммы + IDF; margin и min_confidence; runtime-paraphrases;
  Llm     — Anthropic-классификатор с кэшем;
  Hybrid  — lexical, а на ambiguous/эллипсис/reject+context — LM Studio.

Политика: лучше «не знаю», чем ответ не на тот факт.
Сценарии преподавателя не трогаем — удачные формулировки копятся в cache/paraphrases/.
"""
from __future__ import annotations

import json
import math
import os
import re
import time
import urllib.request
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import Protocol

from . import morph
from .contracts import Act, DialogContext, Timing, Understanding
from .scenario import Scenario

# --------------------------------------------------------------- речевые акты

ACT_PATTERNS = {
    "выезд": r"\b(выехал|выезжа|направ|бригад|помощь ид|уже ед)",
    "успокоение": r"\b(успоко|не паник|оставайтесь|остаться на месте|остаться на линии|не кладите|не вешайте|ждите)",
    "инструкция": r"\b(не подходите|не пытайтесь|не включайте|не курите|"
                  r"отойдите|выйдите|перекройте|не тушите|встречайте|встреть|уходите|"
                  r"не пользуйтесь|пользуйтесь лифтом|предупредите)",
    "приветствие": r"(служба 112|единая служба|служба спасения|слушаю)",
    "прощание": r"\b(до свидания|всего доброго|спасибо, ждите)",
}
CONFIRM_RE = re.compile(r"\b(верно|правильно|значит|то есть|уточняю|"
                        r"подтвержда|подтвердите|точно)\b")
QUESTION_STARTERS = {"есть", "видеть", "слышать", "мочь", "знать",
                     "находиться", "гореть", "далеко", "живой"}
IMPERATIVE_ASK = {"представиться", "назвать", "сказать", "уточнить",
                  "сообщить", "описать", "продиктовать", "повторить"}

DEFAULT_THRESHOLD = 0.34
DEFAULT_MARGIN = 0.10
DEFAULT_LM_URL = "http://127.0.0.1:1234/v1/chat/completions"
DEFAULT_LM_TIMEOUT = 0.20
ELLIPSIS_TOPICAL_MAX = 2


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
    if CONFIRM_RE.search(low):
        return Act.CONFIRM
    return Act.ASK if is_q else Act.ASSERT


def is_ellipsis(text: str) -> bool:
    """Короткое уточнение без самостоятельной темы («какой?», «марка?»)."""
    return len(morph.topical_lemmas(text)) <= ELLIPSIS_TOPICAL_MAX


# ------------------------------------------------------------------ интерфейс

class Understander(Protocol):
    name: str

    def understand(self, text: str,
                   context: DialogContext | None = None) -> Understanding: ...


# ------------------------------------------------------------- paraphrases

class ParaphraseBank:
    """Runtime-банк формулировок. Не пишет в scenarios/ преподавателя."""

    def __init__(self, scenario_id: str, root: str | Path = "cache/paraphrases"):
        self.path = Path(root) / f"{scenario_id}.json"
        self.data: dict[str, list[str]] = {}
        if self.path.exists():
            raw = json.loads(self.path.read_text(encoding="utf-8"))
            if isinstance(raw, dict):
                self.data = {k: list(v) for k, v in raw.items() if isinstance(v, list)}

    def phrases(self, key: str) -> list[str]:
        return list(self.data.get(key, []))

    def learn(self, text: str, keys: list[str]) -> list[tuple[str, str]]:
        """Добавить фразу к ключам. Учим только при ровно одном ключе.

        Возвращает список (key, phrase) реально добавленных.
        """
        phrase = text.strip()
        if not phrase or len(keys) != 1:
            return []
        key = keys[0]
        bucket = self.data.setdefault(key, [])
        # не дублировать без учёта регистра/пробелов
        low = phrase.lower()
        if any(p.lower() == low for p in bucket):
            return []
        bucket.append(phrase)
        return [(key, phrase)]

    def save(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.path.write_text(
            json.dumps(self.data, ensure_ascii=False, indent=1), encoding="utf-8")


# ------------------------------------------------------------- лексический

@dataclass
class MatchHit:
    """Результат матчинга одного сегмента."""
    key: str | None
    score: float
    second: float
    status: str   # accept | ambiguous | reject


class Lexical:
    """Пересечение лемм, взвешенное обратной частотой по банку сценария.

    Принимает ключ только если score >= min_confidence и отрыв от
    второго места >= margin. Иначе reject (или ambiguous для Hybrid).
    """
    name = "lexical"

    def __init__(self, sc: Scenario,
                 threshold: float = DEFAULT_THRESHOLD,
                 margin: float = DEFAULT_MARGIN,
                 paraphrases: ParaphraseBank | None = None):
        self.sc = sc
        self.threshold = threshold
        self.margin = margin
        self.min_confidence = sc.profile.min_confidence
        self.paraphrases = paraphrases or ParaphraseBank(sc.id)
        self.index: dict[str, list[set[str]]] = {}
        self._rebuild_index()

    def _rebuild_index(self) -> None:
        self.index = {}
        for k, f in self.sc.facts.items():
            variants = list(f.questions) + self.paraphrases.phrases(k)
            self.index[k] = [morph.content_lemmas(q) for q in variants]
        self._rebuild_idf()

    def _rebuild_idf(self) -> None:
        df = Counter()
        n = 0
        for variants in self.index.values():
            for s in variants:
                df.update(s)
                n += 1
        self.idf = {w: math.log(1 + n / c) for w, c in df.items()}
        self.default = max(self.idf.values(), default=1.0)

    def add_paraphrase(self, key: str, phrase: str) -> bool:
        """Добавить формулировку в runtime-индекс (и в bank)."""
        if key not in self.sc.facts:
            return False
        lemmas = morph.content_lemmas(phrase)
        if not lemmas:
            # эллипсисы вроде «какой?» в lexical бесполезны — не засоряем банк
            return False
        added = self.paraphrases.learn(phrase, [key])
        if not added:
            return False
        self.index.setdefault(key, []).append(lemmas)
        self._rebuild_idf()
        return True

    def _w(self, s: set[str]) -> float:
        return sum(self.idf.get(x, self.default) for x in s)

    def _score(self, q: set[str], key: str) -> float:
        best = 0.0
        for variant in self.index.get(key, []):
            inter = self._w(q & variant)
            total = self._w(q) + self._w(variant)
            if total:
                best = max(best, 2 * inter / total)
        return best

    def _rank(self, q: set[str], said: set[int]) -> list[tuple[float, str]]:
        ranked = sorted(((self._score(q, k), k) for k in self.index),
                        reverse=True)
        cands = [(s, k) for s, k in ranked if s >= self.threshold]
        if said:
            hit = [(s, k) for s, k in cands
                   if said & self.sc.facts[k].value_numbers]
            if hit:
                cands = hit
        return cands

    def match_segment(self, seg: str, said: set[int] | None = None) -> MatchHit:
        said = said if said is not None else set()
        q = morph.content_lemmas(seg)
        if not q or not morph.topical_lemmas(seg):
            return MatchHit(None, 0.0, 0.0, "reject")
        cands = self._rank(q, said)
        if not cands:
            return MatchHit(None, 0.0, 0.0, "reject")
        best, key = cands[0]
        second = cands[1][0] if len(cands) > 1 else 0.0
        gap = best - second
        if best < self.min_confidence:
            return MatchHit(key, best, second, "ambiguous")
        if gap < self.margin and second > 0:
            return MatchHit(key, best, second, "ambiguous")
        return MatchHit(key, best, second, "accept")

    def understand(self, text: str,
                   context: DialogContext | None = None) -> Understanding:
        del context  # lexical контекст не использует
        t0 = time.perf_counter()
        act = detect_act(text)
        ws = morph.words(text)
        timing = Timing(nlu_path="lexical", llm_status="skip")
        if act is Act.SPEECH_ACT:
            timing.lexical_ms = (time.perf_counter() - t0) * 1000
            timing.nlu_path = "act"
            return Understanding([], act, 1.0, len(ws), source=self.name,
                                 topical=len(morph.topical_lemmas(text)),
                                 timing=timing)

        found, conf = [], 0.0
        said = morph.numbers(text)
        for seg in morph.segments(text):
            hit = self.match_segment(seg, said)
            if hit.status == "accept" and hit.key:
                conf = max(conf, hit.score)
                if hit.key not in found:
                    found.append(hit.key)

        timing.lexical_ms = (time.perf_counter() - t0) * 1000
        mismatch = self._mismatch(text, found)
        return Understanding(found, act, conf, len(ws), mismatch, self.name,
                             len(morph.topical_lemmas(text)), timing)

    def _mismatch(self, text: str, keys: list[str]) -> bool:
        said = morph.numbers(text)
        if not said:
            return False
        for k in keys:
            vn = self.sc.facts[k].value_numbers
            if vn and not (said & vn):
                return True
        return False

    def save(self) -> None:
        self.paraphrases.save()


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
- не выдумывай ключей, которых нет в списке;
- уточнение без полного смысла («какой?», «марка?», «автомобиль?») относи
  к факту из контекста диалога, если подходит; иначе верни пустой список."""


def _parse_keys(raw: str, facts: dict) -> list[str]:
    text = re.sub(r"```json|```", "", raw).strip()
    m = re.search(r"\{.*\}", text, re.DOTALL)
    if m:
        text = m.group(0)
    keys = json.loads(text)["keys"]
    return [k for k in keys if k in facts]


def _format_user(text: str, context: DialogContext | None) -> str:
    if not context or not context.nonempty():
        return text
    ops = "\n".join(f"- {o}" for o in context.recent_ops) or "—"
    keys = ", ".join(context.recent_keys) or "—"
    return (
        f"Недавние реплики оператора:\n{ops}\n"
        f"Недавно обсуждавшиеся факты: {keys}\n"
        f"Текущая реплика: {text}"
    )


class Llm:
    """Классификатор ключей через Anthropic. Текст ответа не генерирует."""
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

    def _norm(self, text: str, context: DialogContext | None = None) -> str:
        base = " ".join(sorted(morph.content_lemmas(text))) or text.lower().strip()
        if context and context.recent_keys:
            return base + "||" + ",".join(context.recent_keys[-3:])
        return base

    def _ask_model(self, text: str,
                   context: DialogContext | None = None
                   ) -> tuple[list[str] | None, str, float]:
        if not self.key:
            return None, "error", 0.0
        body = json.dumps({
            "model": self.MODEL, "max_tokens": 200, "temperature": 0,
            "system": SYSTEM.format(catalog=self.catalog),
            "messages": [{"role": "user", "content": _format_user(text, context)}],
        }).encode()
        req = urllib.request.Request(self.URL, data=body, headers={
            "content-type": "application/json",
            "anthropic-version": "2023-06-01",
            "x-api-key": self.key,
        })
        t0 = time.perf_counter()
        try:
            with urllib.request.urlopen(req, timeout=20) as r:
                data = json.loads(r.read())
            raw = "".join(c.get("text", "") for c in data.get("content", []))
            keys = _parse_keys(raw, self.sc.facts)
            self.calls += 1
            return keys, "ok", (time.perf_counter() - t0) * 1000
        except TimeoutError:
            return None, "timeout", (time.perf_counter() - t0) * 1000
        except Exception:
            return None, "error", (time.perf_counter() - t0) * 1000

    def understand(self, text: str,
                   context: DialogContext | None = None) -> Understanding:
        act = detect_act(text)
        ws = morph.words(text)
        timing = Timing(nlu_path="llm", llm_status="skip")
        if act is Act.SPEECH_ACT:
            timing.nlu_path = "act"
            return Understanding([], act, 1.0, len(ws), source="act", timing=timing)

        norm = self._norm(text, context)
        if norm in self.cache:
            self.hits += 1
            keys, src = self.cache[norm], "cache"
            timing.llm_status = "cache"
            timing.nlu_path = "cache"
        else:
            keys, status, llm_ms = self._ask_model(text, context)
            timing.llm_called = True
            timing.llm_ms = llm_ms
            timing.llm_status = status
            if keys is None:
                fb = self.fallback.understand(text, context)
                fb.timing.llm_called = True
                fb.timing.llm_ms = llm_ms
                fb.timing.llm_status = status
                return fb
            self.cache[norm] = keys
            src = "llm"
            timing.nlu_path = "llm"

        mismatch = Lexical._mismatch(self, text, keys)
        return Understanding(keys, act, 1.0 if keys else 0.0, len(ws),
                             mismatch, src,
                             len(morph.topical_lemmas(text)), timing)

    def save(self) -> None:
        self.cache_path.parent.mkdir(parents=True, exist_ok=True)
        self.cache_path.write_text(
            json.dumps(self.cache, ensure_ascii=False, indent=1), encoding="utf-8")
        save = getattr(self.fallback, "save", None)
        if save:
            save()


# -------------------------------------------------------- LM Studio / Hybrid

class LocalLlm:
    """OpenAI-compatible классификатор (LM Studio на localhost)."""
    name = "local_llm"

    def __init__(self, sc: Scenario,
                 url: str | None = None,
                 model: str | None = None,
                 timeout: float | None = None,
                 cache_path: str = "cache/understand_local.json"):
        self.sc = sc
        self.url = url or os.environ.get("LM_STUDIO_URL", DEFAULT_LM_URL)
        self.model = model or os.environ.get("LM_STUDIO_MODEL", "local")
        self.timeout = float(
            timeout if timeout is not None
            else os.environ.get("LM_STUDIO_TIMEOUT", DEFAULT_LM_TIMEOUT))
        self.cache_path = Path(cache_path)
        self.cache: dict[str, list[str]] = {}
        if self.cache_path.exists():
            self.cache = json.loads(self.cache_path.read_text(encoding="utf-8"))
        self.catalog = "\n".join(
            f"- {k}: {f.questions[0]}" for k, f in sc.facts.items())
        self.system = SYSTEM.format(catalog=self.catalog)
        self.calls = 0
        self.hits = 0

    def _norm(self, text: str, context: DialogContext | None = None) -> str:
        base = " ".join(sorted(morph.content_lemmas(text))) or text.lower().strip()
        if context and context.recent_keys:
            return base + "||" + ",".join(context.recent_keys[-3:])
        return base

    def classify(self, text: str,
                 context: DialogContext | None = None
                 ) -> tuple[list[str], str, float, bool]:
        """(keys, status, ms, from_cache)."""
        norm = self._norm(text, context)
        if norm in self.cache:
            self.hits += 1
            return self.cache[norm], "cache", 0.0, True

        body = json.dumps({
            "model": self.model,
            "temperature": 0,
            "max_tokens": 200,
            "messages": [
                {"role": "system", "content": self.system},
                {"role": "user", "content": _format_user(text, context)},
            ],
        }).encode()
        req = urllib.request.Request(self.url, data=body, headers={
            "content-type": "application/json",
        })
        t0 = time.perf_counter()
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as r:
                data = json.loads(r.read())
            raw = data["choices"][0]["message"]["content"]
            keys = _parse_keys(raw, self.sc.facts)
            self.calls += 1
            self.cache[norm] = keys
            return keys, "ok", (time.perf_counter() - t0) * 1000, False
        except Exception as e:
            ms = (time.perf_counter() - t0) * 1000
            msg = str(e).lower()
            timed = (
                isinstance(e, TimeoutError)
                or isinstance(getattr(e, "reason", None), TimeoutError)
                or "timed out" in msg
                or "timeout" in type(e).__name__.lower()
            )
            return [], "timeout" if timed else "error", ms, False

    def save(self) -> None:
        self.cache_path.parent.mkdir(parents=True, exist_ok=True)
        self.cache_path.write_text(
            json.dumps(self.cache, ensure_ascii=False, indent=1), encoding="utf-8")


class Hybrid:
    """Lexical на уверенных матчах; LM Studio на ambiguous / эллипсис / reject+context."""
    name = "hybrid"

    def __init__(self, sc: Scenario,
                 lexical: Lexical | None = None,
                 local: LocalLlm | None = None,
                 paraphrases: ParaphraseBank | None = None):
        self.sc = sc
        bank = paraphrases or ParaphraseBank(sc.id)
        self.lexical = lexical or Lexical(sc, paraphrases=bank)
        self.local = local or LocalLlm(sc)

    def _need_llm(self, text: str, found: list[str], need_ambiguous: bool,
                  had_reject: bool, context: DialogContext | None) -> bool:
        if found:
            return False
        ctx_ok = bool(context and context.nonempty())
        if need_ambiguous:
            return True
        if is_ellipsis(text) and ctx_ok:
            return True
        if had_reject and ctx_ok:
            return True
        return False

    def _learn(self, text: str, keys: list[str], status: str) -> None:
        if status not in ("ok", "cache") or len(keys) != 1:
            return
        self.lexical.add_paraphrase(keys[0], text)

    def understand(self, text: str,
                   context: DialogContext | None = None) -> Understanding:
        act = detect_act(text)
        ws = morph.words(text)
        timing = Timing(nlu_path="lexical", llm_status="skip")
        if act is Act.SPEECH_ACT:
            timing.nlu_path = "act"
            return Understanding([], act, 1.0, len(ws), source="act",
                                 topical=len(morph.topical_lemmas(text)),
                                 timing=timing)

        found, conf = [], 0.0
        said = morph.numbers(text)
        need_ambiguous = False
        had_reject = False
        t_lex = time.perf_counter()
        segs = list(morph.segments(text)) or [text]
        for seg in segs:
            hit = self.lexical.match_segment(seg, said)
            if hit.status == "accept" and hit.key:
                conf = max(conf, hit.score)
                if hit.key not in found:
                    found.append(hit.key)
            elif hit.status == "ambiguous":
                need_ambiguous = True
            else:
                had_reject = True
        timing.lexical_ms = (time.perf_counter() - t_lex) * 1000

        src = "lexical"
        if self._need_llm(text, found, need_ambiguous, had_reject, context):
            keys, status, llm_ms, from_cache = self.local.classify(text, context)
            timing.llm_called = not from_cache
            timing.llm_ms = llm_ms
            timing.llm_status = status if not from_cache else "cache"
            if from_cache:
                timing.nlu_path = "cache"
                src = "cache"
            elif status == "ok":
                timing.nlu_path = "hybrid_llm"
                src = "hybrid"
            else:
                timing.nlu_path = "lexical"
                keys = []
                src = "hybrid"
            found = keys
            conf = 1.0 if keys else 0.0
            self._learn(text, found, "cache" if from_cache else status)

        mismatch = Lexical._mismatch(self.lexical, text, found)
        return Understanding(found, act, conf, len(ws), mismatch, src,
                             len(morph.topical_lemmas(text)), timing)

    def save(self) -> None:
        self.local.save()
        self.lexical.save()


def build(sc: Scenario, kind: str = "lexical") -> Understander:
    if kind == "llm":
        return Llm(sc)
    if kind == "hybrid":
        return Hybrid(sc)
    return Lexical(sc)
