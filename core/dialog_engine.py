# -*- coding: utf-8 -*-
"""
Движок диалога заявителя для тренажёра оператора 112.

Всё, что тут есть, работает на CPU без нейросетей в рантайме.
Канал эмбеддингов подключается через протокол Embedder; если его нет,
движок работает на одном лексическом канале (для демо этого достаточно).
"""
from __future__ import annotations

import json
import re
from collections import Counter
from dataclasses import dataclass, field
from enum import Enum
from typing import Iterable, Optional, Protocol, Sequence

# ---------------------------------------------------------------- нормализация

FILLERS = {"ну", "вот", "короче", "алло", "пожалуйста", "слушайте", "эээ", "эм"}

# что Vosk делает с нашими словами: заполняется прогоном STT по банку перефразировок
STT_FIXES = {
    "газ и фицирован": "газифицирован",
    "эта жизнь": "этажность",
    "по страдавшие": "пострадавшие",
    "а дрес": "адрес",
}


def normalize(text: str) -> str:
    t = text.lower().replace("ё", "е")
    for bad, good in STT_FIXES.items():
        t = t.replace(bad, good)
    t = re.sub(r"[^а-яa-z0-9?,\s]", " ", t)
    tokens = [w for w in t.split() if w.strip(",") not in FILLERS]
    return " ".join(tokens).strip()


def segment(text: str) -> list[str]:
    """Режем многосоставный вопрос. Целую фразу тоже возвращаем."""
    parts = re.split(r"\s*(?:,|;|\?|\bа также\b|\bа еще\b|\bи еще\b|\bи\b)\s*", text)
    parts = [p.strip() for p in parts if len(p.strip()) >= 3]
    out = list(dict.fromkeys(parts))
    if text not in out:
        out.append(text)
    return out


# ------------------------------------------------------ лексический канал

def char_ngrams(s: str, n: int = 3) -> Counter:
    s = f"  {s} "
    return Counter(s[i:i + n] for i in range(len(s) - n + 1))


def dice(a: Counter, b: Counter) -> float:
    inter = sum((a & b).values())
    total = sum(a.values()) + sum(b.values())
    return 2 * inter / total if total else 0.0


def build_idf(phrases: Iterable[str]) -> dict[str, float]:
    """Вес триграммы = обратная частота по банку перефразировок сценария."""
    import math
    docs = [set(char_ngrams(p)) for p in phrases]
    n = len(docs) or 1
    df: Counter = Counter()
    for d in docs:
        df.update(d)
    return {g: math.log(1 + n / c) for g, c in df.items()}


MAX_IDF = 3.0


def lexical_sim(query: str, phrase: str, idf: dict[str, float] | None = None) -> float:
    a, b = char_ngrams(query), char_ngrams(phrase)
    if idf is None:
        return dice(a, b)
    w = lambda g: idf.get(g, MAX_IDF)
    inter = sum(w(g) * min(a[g], b[g]) for g in a.keys() & b.keys())
    total = sum(w(g) * c for g, c in a.items()) + sum(w(g) * c for g, c in b.items())
    return 2 * inter / total if total else 0.0


# ------------------------------------------------------ канал эмбеддингов

class Embedder(Protocol):
    def encode(self, texts: Sequence[str]) -> "list[list[float]]": ...


class NullEmbedder:
    """Заглушка: движок работает без эмбеддингов, только на лексике."""
    def encode(self, texts: Sequence[str]):
        return None


# ------------------------------------------------------------ речевые акты

class Act(str, Enum):
    ASK = "ask"          # спрашивает впервые
    REPEAT = "repeat"    # спрашивает повторно
    CONFIRM = "confirm"  # переспрашивает уже названное
    ASSERT = "assert"    # резюмирует, утверждает
    ACT = "act"          # речевой акт: успокоил, сообщил о выезде


QWORDS = {
    "какой", "какая", "какое", "какие", "каком", "какую", "какого",
    "сколько", "где", "куда", "откуда", "когда", "кто", "кого", "что",
    "чем", "почему", "зачем", "как", "чей", "скажите", "уточните", "назовите",
}
INVERSION_STARTERS = {
    "есть", "видите", "видно", "слышите", "можете", "знаете", "находитесь",
    "находится", "горит", "живете", "далеко", "рядом",
}
IMPERATIVE_ASK = {
    "представьтесь", "назовите", "скажите", "уточните", "сообщите",
    "опишите", "продиктуйте", "повторите", "перечислите",
}
CONFIRM_MARKERS = ("точно", "верно", "правильно", "значит", "то есть",
                   "я понял", "я поняла", "уточняю", "подтверд", "так")

# Естественные переспросы при unknown/ambiguous. Циклически меняются,
# чтобы не звучало как заевшая пластинка. См. RUNBOOK §3.
REPHRASES = [
    "Алло, не поняла, повторите, пожалуйста.",
    "Я вас не расслышала, скажите иначе.",
    "Повторите, пожалуйста, я не поняла.",
    "Что?.. Переспросите, я не расслышала.",
]
ACT_RE = re.compile(
    r"\b(оставайтесь|оставайся|ждите|не кладите|не вешайте|успокой|"
    r"выехал|выезжа|направил|направля|принято|записал|записала|"
    r"бригада|помощь идет|все понял[а]? спасибо|до свидания)\b"
)

PRONOUNS = {"он", "она", "оно", "они", "его", "ее", "её", "их",
            "это", "этот", "эта", "эти", "тот", "та", "те", "там",
            "тут", "сам", "сама", "само"}

CARDINAL_STEMS = {
    "один": 1, "два": 2, "две": 2, "три": 3, "четыре": 4, "пять": 5,
    "шесть": 6, "семь": 7, "восемь": 8, "девять": 9, "десять": 10,
    "двадцать": 20, "тридцать": 30, "сорок": 40, "пятьдесят": 50,
    "шестьдесят": 60, "семьдесят": 70, "восемьдесят": 80, "девяносто": 90,
    "сто": 100,
}

ORDINAL_STEMS = {
    "перв": 1, "втор": 2, "трет": 3, "четверт": 4, "пят": 5, "шест": 6,
    "седьм": 7, "восьм": 8, "девят": 9, "десят": 10, "одиннадцат": 11,
    "двенадцат": 12, "тринадцат": 13, "четырнадцат": 14, "пятнадцат": 15,
    "шестнадцат": 16, "семнадцат": 17, "восемнадцат": 18, "девятнадцат": 19,
}


def extract_numbers(text: str) -> set[int]:
    nums = {int(m) for m in re.findall(r"\b\d{1,3}\b", text)}
    for stem, val in ORDINAL_STEMS.items():
        if stem in text:
            nums.add(val)
    for word, val in CARDINAL_STEMS.items():
        if re.search(rf"\b{word}\b", text):
            nums.add(val)
    return nums


def detect_act(text: str, already_revealed: bool) -> Act:
    tokens = text.split()
    is_question = (
        "?" in text
        or bool(QWORDS & set(tokens))
        or " ли " in f" {text} "
        or (tokens and tokens[0] in INVERSION_STARTERS)
        or bool(IMPERATIVE_ASK & set(tokens))
    )
    # Речевой акт перебивает вопрос? Нет — «принято, как вас зовут?» это вопрос.
    if not is_question and ACT_RE.search(text):
        return Act.ACT
    has_confirm = any(m in text for m in CONFIRM_MARKERS)
    if is_question and has_confirm:
        return Act.CONFIRM
    if is_question:
        return Act.REPEAT if already_revealed else Act.ASK
    if already_revealed:
        return Act.CONFIRM if has_confirm else Act.ASSERT
    return Act.ASSERT


# ------------------------------------------------------------- сценарий

@dataclass
class Fact:
    key: str
    group: str
    value_numbers: set[int] = field(default_factory=set)
    paraphrases: list[str] = field(default_factory=list)
    replies: dict[str, str] = field(default_factory=dict)
    requires: list[str] = field(default_factory=list)

    def reply_for(self, act: Act) -> str:
        return self.replies.get(act.value) or self.replies.get("ask", "")


@dataclass(eq=False)
class Scenario:
    scenario_id: str
    facts: dict[str, Fact]
    disambig: dict[str, list[tuple[str, str]]] = field(default_factory=dict)
    tau_low: float = 0.34
    tau_high: float = 0.46
    delta: float = 0.05

    @staticmethod
    def from_json(path: str) -> "Scenario":
        return Scenario.from_dict(json.load(open(path, encoding="utf-8")))

    @staticmethod
    def load_many(path: str) -> "list[Scenario]":
        return [Scenario.from_dict(r) for r in json.load(open(path, encoding="utf-8"))]

    @staticmethod
    def from_dict(raw: dict) -> "Scenario":
        facts = {}
        for f in raw["facts"]:
            facts[f["key"]] = Fact(
                key=f["key"], group=f["group"],
                value_numbers=set(f.get("value_numbers", [])),
                paraphrases=f["paraphrases"], replies=f["replies"],
                requires=f.get("requires", []),
            )
        dis = {g: [(p, k) for p, k in rules]
               for g, rules in raw.get("disambig", {}).items()}
        return Scenario(raw["scenario_id"], facts, dis,
                        **raw.get("thresholds", {}))


# --------------------------------------------------------------- матчер

@dataclass
class Hit:
    fact_key: Optional[str]
    group: Optional[str]
    score: float
    margin: float
    segment: str
    status: str  # answered | ambiguous | unknown


class Matcher:
    def __init__(self, scenario: Scenario, embedder: Embedder | None = None,
                 w_emb: float = 0.7):
        self.sc = scenario
        self.emb = embedder
        self.w_emb = w_emb if embedder and not isinstance(embedder, NullEmbedder) else 0.0
        self.idf = build_idf([p for f in scenario.facts.values() for p in f.paraphrases])

    def _group_scores(self, seg: str) -> dict[str, float]:
        scores: dict[str, float] = {}
        for fact in self.sc.facts.values():
            best = max(lexical_sim(seg, p, self.idf) for p in fact.paraphrases)
            scores[fact.group] = max(scores.get(fact.group, 0.0), best)
        # сюда же подмешивается косинус эмбеддингов, если модель подключена
        return scores

    STOP = {"какой", "какая", "какое", "какие", "какого", "какую", "каком",
            "сколько", "где", "когда", "кто", "что", "чем", "как", "почему",
            "есть", "ли", "вы", "ваш", "ваша", "ваше", "там", "это", "сейчас",
            "назовите", "скажите", "уточните", "именно", "или", "не", "на",
            "сами", "сам", "сама", "тут", "вообще", "еще", "уже", "тоже",
            "пожалуйста", "значит", "верно", "точно", "правильно", "может",
            "можете", "будьте", "были", "было", "быть", "очень", "нам",
            "в", "с", "у", "и", "а", "по", "до", "от", "за", "мне", "вам",
            # Полярность и модальность: не несут тематической нагрузки,
            # не должны срабатывать как «чужое слово» в _unknown_content_word.
            "нет", "без", "отсутствуют", "отсутствует",
            "возможно", "кажется", "наверное", "видимо", "вероятно",
            "вообще", "вообще-то", "точно-то", "никак", "вроде",
            "никто", "ничего", "нигде", "никуда",
            # Модификаторы/уточнители: не несут содержания, не должны флаговать unknown.
            "точный", "полный", "примерно", "приблизительно", "ориентировочно"}

    def _has_group_content(self, seg: str, group: str) -> bool:
        """Есть ли в сегменте хотя бы одно content-слово из банка группы.

        Считаем баланс: точное совпадение слова в банке + морфологическое совпадение
        (падежи: «дома»/«дом», «квартиры»/«квартира»). Если из content-слов сегмента
        совпало хотя бы одно И промахов меньше половины — сегмент про это.
        Только полное отсутствие совпадений считаем off-topic.
        """
        bank = " ".join(normalize(p) for f in self.sc.facts.values()
                        if f.group == group for p in f.paraphrases)
        bank_words = set(w for w in bank.split() if len(w) >= 3)
        hits = 0
        misses = 0
        for tok in seg.split():
            tok = tok.rstrip("?,")
            if tok in self.STOP or len(tok) < 3 or tok.isdigit():
                continue
            if extract_numbers(tok):
                continue
            if self._is_verb(tok):  # императив/глагол не несёт тематической нагрузки
                continue
            # точное вхождение слова в банк (нормализованный)
            if tok in bank_words:
                hits += 1
            elif self._shares_root(tok, list(bank_words)):
                hits += 1
            else:
                misses += 1
        # Нет ни одного знакомого слова — явно off-topic.
        # Много промахов при наличии совпадений — тоже подозрительно.
        if hits == 0:
            return False
        if misses >= hits:
            return False
        return True

    @staticmethod
    def _shares_root(tok: str, bank_words: list[str]) -> bool:
        """«дома» ↔ «дом», «квартиры» ↔ «квартира», «улицы» ↔ «улица»."""
        for w in bank_words:
            common = 0
            for a, b in zip(tok, w):
                if a != b:
                    break
                common += 1
            if common >= 3 and common >= min(len(tok), len(w)) - 1:
                return True
        return False

    @staticmethod
    def _is_verb(tok: str) -> bool:
        # русские императивы 2 лица: -ите, -йте, -ете; инфинитив: -ть;
        # возвратные: -ться/-тся; прошедшее: -л-
        return tok.endswith(("ите", "йте", "ете", "ться", "тся", "ала", "али", "ать", "ять"))

    def _disambiguate(self, seg: str, group: str) -> Optional[str]:
        for pattern, key in self.sc.disambig.get(group, []):
            if re.search(pattern, seg):
                return key
        in_group = [f.key for f in self.sc.facts.values() if f.group == group]
        return in_group[0] if len(in_group) == 1 else None

    def match(self, seg: str) -> Hit:
        scores = self._group_scores(seg)
        ranked = sorted(scores.items(), key=lambda kv: -kv[1])
        (g1, s1), (g2, s2) = ranked[0], (ranked[1] if len(ranked) > 1 else ("", 0.0))
        margin = s1 - s2
        if s1 < self.sc.tau_low:
            return Hit(None, None, s1, margin, seg, "unknown")
        if s1 < self.sc.tau_high and margin < self.sc.delta:
            return Hit(None, g1, s1, margin, seg, "ambiguous")
        # Если в сегменте нет ни одного знакомого content-слова из банка группы,
        # оператор говорит про что-то совсем другое — это unknown.
        if not self._has_group_content(seg, g1):
            return Hit(None, g1, s1, margin, seg, "unknown")
        key = self._disambiguate(seg, g1)
        if key is None:
            return Hit(None, g1, s1, margin, seg, "ambiguous")
        return Hit(key, g1, s1, margin, seg, "answered")


# ------------------------------------------------------------- диалог

@dataclass
class Turn:
    utterance: str
    hits: list[Hit]
    acts: dict[str, Act]
    reply: str
    corrected: bool = False


class Dialog:
    MAX_FACTS_PER_TURN = 2

    def __init__(self, scenario: Scenario, matcher: Matcher):
        self.sc = scenario
        self.m = matcher
        self.revealed: set[str] = set()
        self.unresolved: list[str] = []
        self.speech_acts: list[str] = []
        self.last_fact_key: Optional[str] = None
        self.log: list[Turn] = []
        self._rephrase_idx = 0  # циклический индекс для переспросов

    ACT_REPLIES = {
        "успокоение": "Хорошо... хорошо, я жду",
        "выезд": "Спасибо! Скорее пожалуйста, там пламя уже в окна бьёт",
        "прощание": "Да, спасибо вам",
    }

    def _act_kind(self, text: str) -> Optional[str]:
        if re.search(r"\b(выехал|выезжа|направил|направля|бригада|помощь идет)", text):
            return "выезд"
        if re.search(r"\b(успокой|оставайтесь|оставайся|ждите|не кладите|не вешайте)", text):
            return "успокоение"
        if re.search(r"\b(до свидания|всего доброго)", text):
            return "прощание"
        return None

    def handle(self, utterance: str) -> Turn:
        text = normalize(utterance)

        # речевой акт перехватывается, только если в реплике нет вопроса.
        # Иначе «принято, как вас зовут?» уходит в акт и вопрос теряется.
        tokens = text.split()
        has_question = (
            "?" in utterance
            or bool(QWORDS & set(tokens))
            or " ли " in f" {text} "
            or (tokens and tokens[0] in INVERSION_STARTERS)
            or bool(IMPERATIVE_ASK & set(tokens))
        )
        kind = None if has_question else self._act_kind(text)
        if kind:
            self.speech_acts.append(kind)
            turn = Turn(utterance, [], {"__act__": Act.ACT}, self.ACT_REPLIES[kind])
            self.log.append(turn)
            return turn

        hits = [self.m.match(s) for s in segment(text)]

        # анафора: реплика без ключевых вопросов целится в раскрытый факт.
        # Триггеры: короткая/хвостовая фраза, либо озвученное значение с маркером
        # подтверждения («916-126-34-71, верно?»), либо фраза с местоимением
        # («Он в сознании?» после упоминания пострадавшего).
        if not any(h.status == "answered" for h in hits) and self.revealed:
            tokens = set(text.split())
            said = extract_numbers(text)
            confirm = CONFIRM_MARKERS
            has_confirm = bool({m for m in confirm if m in text})
            has_own_topic = bool(QWORDS & tokens) and len(tokens) > 2
            short_or_tail = len(tokens) <= 4 or text.startswith("а ")
            starts_with_pronoun = bool(tokens & PRONOUNS)
            value_statement = bool(said) and has_confirm and not short_or_tail
            if (short_or_tail and not has_own_topic) or value_statement or starts_with_pronoun:
                target = None
                # 1) привязка по значению: больше всего общих чисел с фактом
                scored = [(k, len(said & self.sc.facts[k].value_numbers))
                          for k in self.revealed
                          if self.sc.facts[k].value_numbers and (said & self.sc.facts[k].value_numbers)]
                if scored:
                    target = max(scored, key=lambda x: x[1])[0]
                # 2) иначе — последний факт, если есть сигнал (числа/маркер/местоимение).
                elif said or has_confirm or starts_with_pronoun:
                    target = self.last_fact_key
                if target:
                    hits = [Hit(target, self.sc.facts[target].group,
                                1.0, 1.0, text, "answered")]

        answered, seen = [], set()
        for h in hits:
            if h.status == "answered" and h.fact_key not in seen:
                fact = self.sc.facts[h.fact_key]
                if all(r in self.revealed for r in fact.requires):
                    answered.append(h)
                    seen.add(h.fact_key)

        acts, parts, corrected = {}, [], False
        for h in answered[: self.MAX_FACTS_PER_TURN]:
            fact = self.sc.facts[h.fact_key]
            act = detect_act(text, h.fact_key in self.revealed)
            # оператор резюмирует с неверным числом -> заявитель поправляет.
            # Сверяем каждую названную цифру: если хоть одна не из значений
            # факта — заявитель её не узнаёт, значит оператор ошибся.
            said = extract_numbers(text)
            if act in (Act.ASSERT, Act.CONFIRM) and fact.value_numbers and said:
                if said - fact.value_numbers:
                    act, corrected = Act.ASSERT, True
                    parts.append(fact.replies.get("correct", fact.reply_for(Act.ASK)))
                    acts[h.fact_key] = act
                    self.revealed.add(h.fact_key)
                    self.last_fact_key = h.fact_key
                    continue
            parts.append(fact.reply_for(act))
            acts[h.fact_key] = act
            self.revealed.add(h.fact_key)
            self.last_fact_key = h.fact_key

        if not parts:
            # Цикл из 4 естественных переспросов — чтобы не звучало как
            # баг «я вас не слышу» и не повторялось буквально. Оператор
            # должен понять, что проблема в формулировке, а не в связи.
            fallback = REPHRASES[self._rephrase_idx % len(REPHRASES)]
            self._rephrase_idx += 1
            reply = fallback
            self.unresolved.append(text)
        else:
            reply = ", ".join(parts)
            if len(answered) > self.MAX_FACTS_PER_TURN:
                reply += ". Подождите, вы слишком быстро!"

        turn = Turn(utterance, hits, acts, reply, corrected)
        self.log.append(turn)
        return turn

    def scorecard(self) -> dict:
        total = len(self.sc.facts)
        return {
            "добыто фактов": f"{len(self.revealed)}/{total}",
            "речевых актов": len(self.speech_acts),
            "не понято реплик": len(self.unresolved),
            "непонятое": self.unresolved,
        }
