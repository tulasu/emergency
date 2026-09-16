# -*- coding: utf-8 -*-
"""
Запуск любого сценария.

  python3 run_scenario.py built.json                      # список сценариев
  python3 run_scenario.py built.json -s bilet02           # интерактивный диалог
  python3 run_scenario.py built.json -s bilet02 -f q.txt  # прогон фраз из файла
  python3 run_scenario.py built.json -s bilet02 -f q.txt --blind   # отчёт покрытия

Формат файла фраз: одна реплика оператора на строку, пустые и # игнорируются.
Именно такой файл выдаёт модель по промпту из PROMPT_blind_operator.md.
"""
from __future__ import annotations

import argparse
import json
import sys
from collections import Counter

from dialog_engine import Dialog, Matcher, Scenario, normalize, segment


def load(path: str) -> dict[str, Scenario]:
    raw = json.load(open(path, encoding="utf-8"))
    items = raw if isinstance(raw, list) else [raw]
    return {r["scenario_id"]: Scenario.from_dict(r) for r in items}


def pick(scenarios: dict[str, Scenario], key: str) -> Scenario:
    hits = [v for k, v in scenarios.items() if key in k]
    if len(hits) != 1:
        sys.exit(f"по «{key}» найдено {len(hits)} сценариев: "
                 f"{list(scenarios)}")
    return hits[0]


def read_phrases(path: str) -> list[str]:
    out = []
    for line in open(path, encoding="utf-8"):
        line = line.strip().lstrip("-•*0123456789. ").strip()
        if line and not line.startswith("#"):
            out.append(line)
    return out


def interactive(sc: Scenario):
    d = Dialog(sc, Matcher(sc))
    print(f"Сценарий: {sc.scenario_id}. Пустая строка — выход.\n")
    while True:
        try:
            utt = input("Оператор : ").strip()
        except (EOFError, KeyboardInterrupt):
            break
        if not utt:
            break
        t = d.handle(utt)
        acts = ", ".join(f"{k}:{v.value}" for k, v in t.acts.items()) or "—"
        print(f"Заявитель: {t.reply}\n           [{acts}]"
              f"{'  ← ПОПРАВИЛ' if t.corrected else ''}\n")
    print("ИТОГ:", d.scorecard())


def scripted(sc: Scenario, phrases: list[str]):
    d = Dialog(sc, Matcher(sc))
    for utt in phrases:
        t = d.handle(utt)
        acts = ", ".join(f"{k}:{v.value}" for k, v in t.acts.items()) or "—"
        print(f"  О: {utt}\n  З: {t.reply}\n     [{acts}]"
              f"{'  ← ПОПРАВИЛ' if t.corrected else ''}")
    print("\nИТОГ:", d.scorecard())


def blind(sc: Scenario, phrases: list[str]):
    """Оценка покрытия на фразах, сгенерированных вслепую."""
    m = Matcher(sc)
    per_fact: Counter = Counter()
    uncovered, ambiguous = [], []
    for p in phrases:
        hits = [m.match(s) for s in segment(normalize(p))]
        answered = [h for h in hits if h.status == "answered"]
        if answered:
            for h in answered:
                per_fact[h.fact_key] += 1
        elif any(h.status == "ambiguous" for h in hits):
            ambiguous.append(p)
        else:
            uncovered.append(p)

    n = len(phrases)
    covered = n - len(uncovered) - len(ambiguous)
    print(f"\nСЦЕНАРИЙ {sc.scenario_id}")
    print(f"фраз: {n} | распознано: {covered} ({100*covered/n:.0f}%) | "
          f"неоднозначно: {len(ambiguous)} | не покрыто: {len(uncovered)}")

    print("\nпопадания по фактам:")
    for key in sc.facts:
        mark = "" if per_fact[key] else "   ← НИ ОДНОГО ПОПАДАНИЯ"
        print(f"  {per_fact[key]:3}  {key}{mark}")

    if ambiguous:
        print("\nнеоднозначные (нужны различающие слова):")
        for p in ambiguous:
            print(f"  · {p}")
    if uncovered:
        print("\nНЕ ПОКРЫТО — кандидаты в банк перефразировок:")
        for p in uncovered:
            print(f"  · {p}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("scenarios")
    ap.add_argument("-s", "--scenario")
    ap.add_argument("-f", "--file")
    ap.add_argument("--blind", action="store_true")
    a = ap.parse_args()

    scs = load(a.scenarios)
    if not a.scenario:
        for k, v in scs.items():
            print(f"  {k}  ({len(v.facts)} фактов)")
        return
    sc = pick(scs, a.scenario)

    if not a.file:
        interactive(sc)
    elif a.blind:
        blind(sc, read_phrases(a.file))
    else:
        scripted(sc, read_phrases(a.file))


if __name__ == "__main__":
    main()
