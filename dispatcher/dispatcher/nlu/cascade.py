# -*- coding: utf-8 -*-
"""
Каскад понимания: реплика -> слоты и ключи фактов.

Три слоя, каждый следующий дороже предыдущего и вызывается реже:

  1. правила   речевой акт, числа, уточнение «а корпус?»      микросекунды
  2. банк      лексика, сплавленная с векторами                 ~1-15 мс
  3. арбитр    локальная модель выбирает из пяти кандидатов   сотни мс, редко

Кандидаты всегда ограничены слотами сценария: заявитель физически не может
ответить о том, чего не знает. Поэтому «не знаю» — не отказ системы, а
правильный ответ, и он отличается от «не расслышала».
"""

from __future__ import annotations

import time
from dataclasses import dataclass, field
from typing import Protocol


from ..data.ontology import Ontology
from ..types import Act, Scenario, SlotKind, Understanding
from . import normalize, rules
from .bank import LexicalBank
from .encoder import Encoder
from .vectors import VectorBank


@dataclass(slots=True)
class Thresholds:
    """Пороги решения. Подбираются бенчмарком и замораживаются.

    Главный признак уверенности — не оценка лидера, а **разрыв до второго
    кандидата**. Абсолютная оценка зависит от энкодера и от того, сколько
    формулировок накопил слот; разрыв не зависит ни от того, ни от другого.
    На корпусе разрыв 0.08 держит долю ложных ответов на 1.7%, тогда как
    абсолютный порог не опускается ниже 6% ни в одной точке.

    Между `gap_low` и `gap_high` лежит серая зона: слишком похоже, чтобы
    отвечать наугад, и слишком осмысленно, чтобы отмахнуться.
    """

    floor: float = 0.05  # ниже этой оценки лидера не отвечаем вовсе
    gap_high: float = 0.06  # разрыв, при котором отвечаем уверенно
    gap_low: float = 0.02  # ниже — «не знаю»
    # Насколько чужой слот должен обойти свой, чтобы счесть вопрос не по
    # адресу. Мало ставить нельзя: общие формулировки вроде «что случилось»
    # раскиданы по десятку слотов, и любой из них перебьёт свой на ровном
    # месте. При 0.08 «что случилось» уходило в отказ, при 0.20 не уходит.
    margin: float = 0.20
    refine_relief: float = 0.10  # насколько опускается порог при уточнении
    numeric_match: float = 0.15  # надбавка слоту, чьё значение совпало с числом
    form_penalty: float = 0.12  # штраф за несовпадение формы вопроса и вида слота
    thin_limit: int = 40  # слот беднее этого банк почти не находит
    echo_weight: float = 0.9  # вес отклика на зачитанное значение

    @staticmethod
    def preset(kind: str) -> "Thresholds":
        """Пороги откалиброваны отдельно для каждой сборки каскада.

        Калибровать надо по слепым наборам, а не по формулировкам корпуса:
        на них другая шкала, и подмена роняет сквозной прогон вдвое.

        ВНИМАНИЕ: `fused` здесь нет намеренно. Он был откалиброван ДО того,
        как банк стал вычитать фон слота, и числа протухли. Перед тем как
        включать векторы в рантайм, прогнать `bench.run -m e5-small:onnx`
        с перебором `--gap` и `--floor` и вписать свежие числа сюда.
        """
        if kind == "tuned":
            # Дообученный энкодер разводит классы: свой слот около 0.9,
            # посторонний вопрос около 0.4. Поэтому отказ строится на
            # абсолютной оценке, а не на сравнении с чужим слотом, и разрыв
            # нужен маленький. Подобрано по слепым наборам.
            return Thresholds(floor=0.60, gap_high=0.03, gap_low=0.01)
        if kind == "fused":
            raise ValueError(
                "пресет fused протух после вычета фона — "
                "прогони bench.run --gap/--floor и впиши свежие числа"
            )
        if kind == "lexical":
            # После вычета фона слота шкала сжалась примерно вдесятеро:
            # оценка лидера может быть отрицательной, а разрывы измеряются
            # сотыми. Подобрано по сквозному прогону слепых наборов, а не по
            # формулировкам корпуса — на них шкала другая, проверено.
            return Thresholds(floor=0.05, gap_high=0.03, gap_low=0.01)
        raise ValueError(f"нет пресета {kind }")


class Arbiter(Protocol):
    """Локальная модель: выбрать из слотов-кандидатов или отказаться."""

    def choose(self, text: str, slots: list[str]) -> str | None: ...


@dataclass(slots=True)
class NluState:
    """Контекст диалога, который нужен пониманию, а не политике.

    Уточняющая реплика «а корпус?» не имеет собственной темы: она держится
    на слоте, о котором только что шла речь.
    """

    last_slots: tuple[str, ...] = ()
    last_family: str = ""
    turn: int = 0

    def observe(self, slots: list[str]) -> None:
        if slots:
            self.last_slots = tuple(slots)
            self.last_family = slots[0].split(".", 1)[0]


@dataclass
class Cascade:
    scenario: Scenario
    lexical: LexicalBank
    ontology: Ontology | None = None
    vectors: VectorBank | None = None
    encoder: Encoder | None = None
    arbiter: Arbiter | None = None
    thresholds: Thresholds | None = None
    alpha: float = 0.85
    without: str = ""  # для честной оценки на своём же корпусе
    thin_rescue: bool = False  # отдавать бедные слоты арбитру

    state: NluState = field(default_factory=NluState)

    def __post_init__(self) -> None:
        if self.thresholds is None:
            self.thresholds = Thresholds.preset(self.kind())
        self.allowed = self.scenario.slots
        # срез банка под слоты этого заявителя: в горячем пути читается он,
        # полный банк — только когда надо понять, не спросили ли о чужом
        self.scoped = (
            self.vectors.subset(self.allowed) if self.vectors is not None else None
        )
        onto = self.ontology or Ontology.load()
        self.kinds = {sid: sl.kind for sid, sl in onto.slots.items()}

        # Слова самих ответов. Оператор зачитывает записанное обратно —
        # «Иванов Петр Иванович, верно?» — и в банке вопросов таких слов нет
        # ни одного: фамилии принадлежат конкретному заявителю, а не слоту.
        self.values: dict[str, frozenset[str]] = {}
        # Числа значения — и объявленные в сценарии, и вытащенные из текста
        # ответа. Оператор зачитывает «тридцать семь», а в ответе стоит «37»:
        # по словам они не сходятся, по числам сходятся.
        self.value_numbers: dict[str, frozenset[int]] = {}
        for slot, keys in self.scenario.by_slot.items():
            lemmas: set[str] = set()
            fact_numbers: set[int] = set()
            for key in keys:
                fact = self.scenario.facts[key]
                lemmas |= normalize.content_lemmas(fact.answers["plain"])
                fact_numbers |= set(fact.numbers) or normalize.numbers(
                    fact.answers["plain"]
                )
            if lemmas:
                self.values[slot] = frozenset(lemmas)
            if fact_numbers:
                self.value_numbers[slot] = frozenset(fact_numbers)

    def kind(self) -> str:
        """Какая это сборка каскада — от неё зависят пороги.

        Дообученный энкодер разводит классы куда сильнее стокового, и пороги
        у него свои. Определять их по одному лишь наличию векторов нельзя:
        дообученная модель с порогами стокового перестаёт отказываться вовсе.
        """
        if self.vectors is None:
            return "lexical"
        model_name = getattr(self.encoder, "name", "")
        return "tuned" if "tuned" in model_name else "fused"

        # ------------------------------------------------------------ основное

    def understand(self, text: str) -> Understanding:
        t0 = time.perf_counter()
        self.state.turn += 1
        u = self._understand(text)
        u.latency_ms = round(1000 * (time.perf_counter() - t0), 3)
        self.state.observe(u.slots)
        return u

    def preview(self, text: str) -> Understanding | None:
        """Прикидка по частичной гипотезе STT: без учёта в состоянии.

        Спекуляция обязана быть чистой — иначе недоговорённая фраза сдвинет
        last_slots/turn, и уточнение «а корпус?» повиснет не на том слоте.
        """
        try:
            return self._understand(text)
        except Exception:  # noqa: BLE001
            return None

    def _understand(self, text: str) -> Understanding:
        words = normalize.words(text)
        topical = normalize.topical_lemmas(text)
        act = rules.detect_act(text)

        # слой 1: речевой акт — оператор не спрашивает, а сообщает
        if act is Act.SPEECH_ACT:
            return Understanding(
                act=act,
                score=1.0,
                words=len(words),
                topical=len(topical),
                source="rules",
            )

            # слой 1: в реплике нет ни одного значимого слова — это шум связи
            # или обрывок, а не вопрос. У лексики такое отсеивается само (искать
            # нечего), а энкодер закодирует и односимвольное «ы», и найдёт ему
            # похожий слот. Заслон нужен до всякого поиска.
        if not normalize.content_lemmas(text):
            return Understanding(act=act, words=len(words), topical=0, source="rules")

            # слой 1: «повторите», «ещё раз» — просьба назвать то же самое.
            # Своей темы у такой реплики нет, её несёт предыдущий ход.
        if rules.repeat_without_topic(text):
            if self.state.last_slots:
                return self._wrap(
                    text, list(self.state.last_slots), act, 1.0, words, topical, "rules"
                )

            # слой 1: уточнение без собственной темы держится на прошлом слоте
        scope, relief = self._scope(text)
        th = self.thresholds
        floor = th.floor - relief
        gap_high = max(th.gap_high - relief, 0.0)
        gap_low = max(th.gap_low - relief, 0.0)

        # слой 2: банк. Считаем по всем 88 слотам, а не только по слотам
        # сценария: иначе не отличить «не расслышала» от «поняла вопрос, но
        # такого заявитель знать не может» — а это разные ответы.
        per_segment = self._score(text, act)
        if not per_segment:
            return self._wrap(text, [], act, 0.0, words, topical, "bank")

        overall: dict[str, float] = {}
        for scores in per_segment:
            for slot, v in scores.items():
                if v > overall.get(slot, 0.0):
                    overall[slot] = v

        mine = max((v for s, v in overall.items() if s in scope), default=0.0)
        outside = max(
            ((v, s) for s, v in overall.items() if s not in self.allowed),
            default=(0.0, ""),
        )

        # Дообученный энкодер кладёт всё в известные слоты, и сравнение
        # «чужой против своего» перестаёт срабатывать. Зато у него разошлась
        # шкала — свой слот даёт около 0.9, посторонний вопрос около 0.4, —
        # и работает прямая отсечка по лучшей своей оценке.
        if mine < floor:
            best = (
                outside[1]
                if outside[0] >= mine
                else (max(scope, key=lambda s: overall.get(s, 0.0)) if scope else "")
            )
            return Understanding(
                slots=[best] if best else [],
                keys=[],
                act=act,
                score=mine,
                words=len(words),
                topical=len(topical),
                source="bank",
            )

        if outside[0] >= floor and outside[0] > mine + th.margin:
            # вопрос понятен, но он о сведении, которого у заявителя нет
            return Understanding(
                slots=[outside[1]],
                keys=[],
                act=act,
                score=outside[0],
                words=len(words),
                topical=len(topical),
                source="bank",
            )

            # по одному победителю на сегмент: два вопроса в реплике — два ответа,
            # один вопрос — один ответ, а не всё, что попало в окно похожести
        picked: list[str] = []
        best_score = 0.0
        grey: list[tuple[str, float]] = []
        for scores in per_segment:
            local = sorted(
                ((v, s) for s, v in scores.items() if s in scope), reverse=True
            )
            if not local:
                continue
            v, slot = local[0]
            gap = v - self._runner_up(local)
            best_score = max(best_score, v)
            if v < floor:
                continue
            if gap >= gap_high:
                if slot not in picked:
                    picked.append(slot)
            elif gap >= gap_low:
                grey.append((slot, v))

        if picked:
            rescued = self._thin_rescue(
                text,
                picked,
                best_score
                - (
                    max(
                        (
                            v
                            for scores in per_segment
                            for s, v in scores.items()
                            if s in scope and s not in picked
                        ),
                        default=0.0,
                    )
                ),
            )
            if rescued:
                return self._wrap(
                    text, [rescued], act, best_score, words, topical, "arbiter"
                )
            return self._wrap(text, picked, act, best_score, words, topical, "bank")

            # слой 3: серая зона — арбитр выбирает из пяти кандидатов
        if grey and self.arbiter is not None:
            ranked = sorted(
                ((v, s) for s, v in overall.items() if s in scope), reverse=True
            )[:5]
            pick = self.arbiter.choose(text, [s for _, s in ranked])
            slots = [pick] if pick in scope else []
            return self._wrap(text, slots, act, best_score, words, topical, "arbiter")

        return self._wrap(text, [], act, best_score, words, topical, "bank")

        # ------------------------------------------------------------ помощники

    def _scope(self, text: str) -> tuple[frozenset[str], float]:
        """Где искать и насколько опустить порог.

        Уточнение «а корпус?» темы не задаёт: ищем в пределах семьи слота,
        о котором только что шла речь, и требуем меньше уверенности.
        """
        if not (rules.is_refinement(text) and self.state.last_family):
            return self.allowed, 0.0
        family = self.state.last_family + "."
        near = frozenset(s for s in self.allowed if s.startswith(family))
        if not near:
            return self.allowed, 0.0

            # «А корпус?» после адреса — уточнение того же. «А телефон?» — новый
            # вопрос, и сужать до адресной семьи нельзя. Отличаем по делу: если
            # вне семьи нашлось заметно лучше, значит тема своя.
        lex_scores = self.lexical.score(text, without=self.without)
        inside = max((v for s, v in lex_scores.items() if s in near), default=0.0)
        outside = max(
            (v for s, v in lex_scores.items() if s in self.allowed and s not in near),
            default=0.0,
        )
        if outside > inside + self.thresholds.gap_high:
            return self.allowed, 0.0
        return near, self.thresholds.refine_relief

    def _score(self, text: str, act: Act) -> list[dict[str, float]]:
        """Оценки по каждому вопросу реплики отдельно."""
        said = normalize.numbers(text) if act in (Act.CONFIRM, Act.ASSERT) else set()

        segs = [
            seg
            for seg in normalize.segments(rules.strip_ritual(text))
            if not rules.speech_act(seg)
        ]
        if not segs:
            return []

            # все куски реплики кодируются одним вызовом: прогон энкодера стоит
            # почти столько же на одну строку, сколько на пять
        vecs = (
            self.encoder.encode(segs, kind="query")
            if self.vectors is not None and self.encoder is not None
            else None
        )

        out: list[dict[str, float]] = []
        for i, seg in enumerate(segs):
            lex = self.lexical.score(seg, without=self.without)
            if vecs is not None:
                vec = self.scoped.score(vecs[i], without=self.without)
                # чужие слоты нужны только чтобы отличить «не знаю» от
                # «не расслышала», и только когда свои ничего не дали
                if max(vec.values(), default=0.0) < self.thresholds.floor:
                    vec = self.vectors.score(vecs[i], without=self.without)
                a = self.alpha
                scores = {
                    s: a * max(vec.get(s, 0.0), 0.0) + (1 - a) * lex.get(s, 0.0)
                    for s in vec.keys() | lex.keys()
                }
            else:
                scores = dict(lex)

            form = rules.question_form(seg)
            if form != "any":
                for slot in list(scores):
                    kind = self.kinds.get(slot)
                    mismatch = (form == "open" and kind is SlotKind.YESNO) or (
                        form == "yesno" and kind is SlotKind.DESCRIPTION
                    )
                    if mismatch:
                        scores[slot] -= self.thresholds.form_penalty

            echo = self._echo(seg)
            # Число, совпавшее со значением факта, — сильная улика само по
            # себе: «тридцать семь, верно?» банк не находит ни одним словом,
            # потому что в ответе стоит «37».
            # ...но только если своей темы у реплики нет. «Тридцать семь,
            # верно?» — темы нет, решает число. «На четырнадцатом горит?» —
            # тема есть, решает она, а чужое число станет поправкой.
            topic = normalize.topical_lemmas(seg) - set(normalize.CARDINALS)
            echoed = normalize.numbers(seg) if not topic else set()
            if echoed:
                for slot, fact_numbers in self.value_numbers.items():
                    if echoed & fact_numbers:
                        echo[slot] = max(
                            echo.get(slot, 0.0), self.thresholds.echo_weight
                        )
            for slot, v in echo.items():
                if v > scores.get(slot, 0.0):
                    scores[slot] = v

            if said:
                scores = self._numeric_only(scores, said)

            if scores:
                out.append(scores)
        return out

    def _numeric_only(
        self, scores: dict[str, float], said: set[int]
    ) -> dict[str, float]:
        """Оператор зачитал число — значит речь о факте, у которого есть значение.

        «То есть на четырнадцатом горит?» — про этаж возгорания, и неважно,
        что слова «горит» ближе к «что горит»: подтверждать нечего, если у
        факта нет значения. Дальше сверка покажет, что четырнадцать — это
        этажность, и заявитель поправит. А если названное число у кого-то из
        кандидатов совпало, он и есть ответ.
        """
        numeric = {s: v for s, v in scores.items() if self._has_numbers(s)}
        if not numeric:
            return scores
        for slot in numeric:
            if said & self._numbers_of(slot):
                numeric[slot] += self.thresholds.numeric_match
        return numeric

    def _numbers_of(self, slot: str) -> frozenset[int]:
        return self.value_numbers.get(slot, frozenset())

    def _echo(self, seg: str) -> dict[str, float]:
        """Насколько реплика повторяет значение факта, а не вопрос о нём."""
        query = frozenset(normalize.content_lemmas(seg))
        if not query:
            return {}
        idf = self.lexical.idf
        default = self.lexical.default_idf
        w = lambda ls: sum(idf.get(x, default) for x in ls)  # noqa: E731
        qw = w(query)

        out: dict[str, float] = {}
        for slot, lemmas in self.values.items():
            shared = query & lemmas
            if not shared:
                continue
            total = qw + w(lemmas)
            if total:
                out[slot] = self.thresholds.echo_weight * 2 * w(shared) / total
        return out

    def _thin_rescue(self, text: str, picked: list[str], gap: float) -> str | None:
        """Право арбитра добавить бедный слот, но не спорить с уверенным банком.

        Банк даёт 83% на слотах со 100+ формулировками и 6% на слотах с
        десятком — бедный он просто не видит. Арбитр от населённости не
        зависит, поэтому ему отдаётся выбор среди слотов сценария, а принимается
        ответ, только если он указал на бедный.

        Но когда банк уверен, вмешиваться нельзя: в живом сеансе арбитр трижды
        подряд перебил верный ответ про подъезд, отвечая про этаж, причём с
        уверенностью 0.85–0.88 — то есть порогом это не отсечь, отсекается
        только уверенностью самого банка.
        """
        if not (self.thin_rescue and self.arbiter):
            return None
        if gap >= 2 * self.thresholds.gap_high:
            return None
        thin = {
            s
            for s in self.allowed
            if self.lexical.population.get(s, 0) <= self.thresholds.thin_limit
        }
        if not thin - set(picked):
            return None

            # Выбор даётся среди ВСЕХ слотов сценария, а не только бедных: иначе
            # арбитр лишён возможности согласиться с банком и вынужден спорить.
        candidates = sorted(
            self.allowed, key=lambda s: -self.lexical.population.get(s, 0)
        )[:5]
        for s in list(thin - set(picked))[:2]:
            if s not in candidates:
                candidates = candidates[:4] + [s]
        pick = self.arbiter.choose(text, candidates)
        # перехватываем только если арбитр указал на бедный слот
        return pick if pick in thin and pick not in picked else None

    def _runner_up(self, local: list[tuple[float, str]]) -> float:
        """Оценка лучшего кандидата, ведущего к ДРУГОМУ ответу.

        Слоты одной семьи часто замещают друг друга и приводят к одному
        факту: «адрес» и «улица» у этого заявителя — одна и та же реплика.
        Разрыв между ними ничего не значит, выбирать не из чего.
        """
        first = self._keys_of(local[0][1])
        for v, slot in local[1:]:
            if self._keys_of(slot) != first:
                return v
        return 0.0

    def _keys_of(self, slot: str) -> tuple[str, ...]:
        own = self.scenario.by_slot.get(slot)
        if own is None:
            stand_in = self.scenario.answers_for.get(slot)
            own = self.scenario.by_slot.get(stand_in, ()) if stand_in else ()
        return tuple(own)

    def _has_numbers(self, slot: str) -> bool:
        return slot in self.value_numbers

    def _wrap(
        self,
        text: str,
        slots: list[str],
        act: Act,
        score: float,
        words: list[str],
        topical: set[str],
        source: str,
    ) -> Understanding:
        keys: list[str] = []
        for slot in slots:
            own = self.scenario.by_slot.get(slot)
            if own is None:
                stand_in = self.scenario.answers_for.get(slot)
                own = self.scenario.by_slot.get(stand_in, ()) if stand_in else ()
            for key in own:
                if key not in keys:
                    keys.append(key)

        said = normalize.numbers(text)
        mismatch = False
        if said:
            for key in keys:
                nums = self.scenario.facts[key].numbers
                if nums and not (said & nums):
                    mismatch = True

        return Understanding(
            slots=slots,
            keys=keys,
            act=act,
            score=score,
            words=len(words),
            topical=len(topical),
            value_mismatch=mismatch,
            source=source,
        )

    def save(self) -> None:
        """Кэшировать нечего — совместимость с сессией."""
