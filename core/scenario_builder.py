# -*- coding: utf-8 -*-
"""
Сборка сценария из черновика, который выдала LLM.

Делает три вещи:
  1) проверяет схему и достраивает недостающие реплики;
  2) САМ выводит правила дизамбигуации из банка перефразировок
     (руками их писать нельзя — там были 4 бага из 9);
  3) гоняет самопроверку: каждая перефразировка обязана попадать
     в свой же факт, иначе границы слотов проведены неверно.

Использование:
    python3 scenario_builder.py draft.json -o scenario.json
"""
from __future__ import annotations

import argparse
import json
import sys
from collections import defaultdict

from dialog_engine import Matcher, Scenario, normalize

# только вопросительный каркас и предлоги: местоимения оставляем,
# они часто и есть различающий признак (ваш этаж против этажности дома)
STOP = {
    "какой", "какая", "какое", "какие", "какого", "какую", "каком", "сколько",
    "где", "когда", "кто", "что", "чем", "как", "почему", "есть", "ли",
    "назовите", "скажите", "уточните", "именно", "или", "не", "на", "в", "с",
    "у", "и", "а", "по", "до", "от", "за", "это", "пожалуйста",
    # Полярность и модальность: не различают факты внутри группы.
    "нет", "без", "отсутствуют", "отсутствует",
    "возможно", "кажется", "наверное", "видимо", "вероятно",
    "вообще", "никак", "вроде",
}
STEM = 5
REQUIRED_REPLIES = ("ask", "repeat", "confirm", "assert")


def stem(token: str) -> str:
    return token[:STEM] if len(token) > STEM else token


def content_tokens(phrases: list[str]) -> set[str]:
    out = set()
    for p in phrases:
        for tok in normalize(p).replace(",", " ").split():
            tok = tok.rstrip("?,")  # «?» цепляется к последнему слову при normalize
            if tok in STOP or len(tok) < 2 or tok.isdigit():
                continue
            out.add(tok)
    return out


def auto_disambig(facts: list[dict]) -> dict:
    """Различающие основы = те, что есть у одного факта группы и нет у соседей."""
    by_group = defaultdict(list)
    for f in facts:
        by_group[f["group"]].append(f)

    rules = {}
    for group, members in by_group.items():
        if len(members) < 2:
            continue
        toks = {f["key"]: content_tokens(f["paraphrases"]) for f in members}
        group_rules = []
        for f in members:
            others = set().union(*(t for k, t in toks.items() if k != f["key"]))
            # основа различает, только если ни одно слово соседей с неё не начинается
            short, long_ = set(), set()
            for t in toks[f["key"]]:
                st = stem(t)
                if len(st) <= 3:
                    # короткое слово: сравниваем точно, иначе "вы" убьёт "высотность"
                    if t not in others:
                        short.add(t)
                elif not any(o.startswith(st) or st.startswith(o) for o in others):
                    long_.add(st)
            distinctive = sorted(long_) + [f"{w}\\b" for w in sorted(short)]
            if not distinctive:
                print(f"  ! {f['key']}: нет ни одного различающего слова внутри "
                      f"группы «{group}» — перефразировки надо развести", file=sys.stderr)
                continue
            group_rules.append(["\\b(" + "|".join(distinctive) + ")", f["key"]])
        rules[group] = group_rules
    return rules


def fill_replies(fact: dict) -> dict:
    r = dict(fact.get("replies", {}))
    if "ask" not in r:
        raise ValueError(f"{fact['key']}: обязательна реплика ask")
    r.setdefault("repeat", r["ask"])
    r.setdefault("confirm", "Да, всё верно")
    r.setdefault("assert", "Да, верно")
    if fact.get("value_numbers") and "correct" not in r:
        r["correct"] = "Нет, вы неправильно записали. " + r["ask"]
    fact["replies"] = r
    return fact


def build(draft: dict) -> dict:
    facts = [fill_replies(dict(f)) for f in draft["facts"]]
    for f in facts:
        f.setdefault("value_numbers", [])
        f.setdefault("requires", [])
        if len(f["paraphrases"]) < 5:
            print(f"  ! {f['key']}: всего {len(f['paraphrases'])} перефразировок, "
                  f"нужно минимум 5", file=sys.stderr)
    out = {
        "scenario_id": draft["scenario_id"],
        "meta": draft.get("meta", {}),
        "thresholds": draft.get("thresholds",
                                {"tau_low": 0.30, "tau_high": 0.42, "delta": 0.04}),
        "facts": facts,
        "disambig": draft.get("disambig") or auto_disambig(facts),
    }
    return out


def selftest(scenario_dict: dict) -> tuple[int, int, list[str]]:
    """Каждая перефразировка обязана попасть в свой факт."""
    sc = Scenario.from_dict(scenario_dict)
    m = Matcher(sc)
    ok = bad = 0
    problems = []
    for fact in sc.facts.values():
        for p in fact.paraphrases:
            h = m.match(normalize(p))
            if h.fact_key == fact.key:
                ok += 1
            else:
                bad += 1
                problems.append(f"    «{p}» -> {h.fact_key or h.status} "
                                f"(ожидался {fact.key}, s={h.score:.2f})")
    return ok, bad, problems


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("draft")
    ap.add_argument("-o", "--out", default=None)
    args = ap.parse_args()

    draft = json.load(open(args.draft, encoding="utf-8"))
    drafts = draft if isinstance(draft, list) else [draft]

    built = []
    for d in drafts:
        print(f"\n=== {d['scenario_id']}")
        sc = build(d)
        ok, bad, problems = selftest(sc)
        print(f"  фактов: {len(sc['facts'])}, перефразировок: {ok + bad}")
        print(f"  самопроверка: {ok} ок, {bad} промахов")
        for p in problems:
            print(p)
        groups = {f["group"] for f in sc["facts"]}
        auto = [g for g in sc["disambig"]]
        print(f"  групп: {len(groups)}, правил выведено для: {auto or '—'}")
        built.append(sc)

    if args.out:
        payload = built if isinstance(draft, list) else built[0]
        json.dump(payload, open(args.out, "w", encoding="utf-8"),
                  ensure_ascii=False, indent=1)
        print(f"\nзаписано: {args.out}")


if __name__ == "__main__":
    main()
