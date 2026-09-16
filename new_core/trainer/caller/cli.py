# -*- coding: utf-8 -*-
"""
Запуск тренажёра.

  python3 -m caller.cli chat  scenarios/bilet04_call01.json
  python3 -m caller.cli run   scenarios/bilet04_call01.json -f examples/dialog.txt
  python3 -m caller.cli eval  scenarios/bilet04_call01.json -f examples/blind.txt

Понимание переключается флагом --nlu lexical|llm (по умолчанию lexical).
Для llm нужен ANTHROPIC_API_KEY; без него молча используется lexical.
"""
from __future__ import annotations

import argparse
import sys
from collections import Counter
from pathlib import Path

from .contracts import Style
from .engine import Engine
from .scenario import Scenario


def read_lines(path: str) -> list[str]:
    out = []
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        line = line.strip().lstrip("-•*0123456789. ").strip()
        if line and not line.startswith("#"):
            out.append(line)
    return out


def show(turn) -> None:
    src = turn.understanding.source
    keys = ",".join(turn.decision.reveal) or "—"
    print(f"  О: {turn.utterance}")
    print(f"  З: {turn.reply.text}")
    print(f"     [{turn.decision.style.value} · {turn.decision.mood.name} · "
          f"{keys} · {src}]")


def cmd_chat(eng: Engine) -> None:
    print(f"З: {eng.opening()}\n(пустая строка — завершить)\n")
    while True:
        try:
            line = input("О: ").strip()
        except (EOFError, KeyboardInterrupt):
            break
        if not line:
            break
        t = eng.handle(line)
        print(f"З: {t.reply.text}")
        print(f"   [{t.decision.style.value} · {t.decision.mood.name} · "
              f"{','.join(t.decision.reveal) or '—'}]\n")
    print("\n" + eng.close())


def cmd_run(eng: Engine, phrases: list[str]) -> None:
    print(f"З: {eng.opening()}\n")
    for p in phrases:
        show(eng.handle(p))
    print("\n" + eng.close())


def cmd_eval(eng: Engine, phrases: list[str]) -> None:
    """Покрытие на слепом наборе. Каждая реплика — с чистого состояния."""
    sc = eng.sc
    per_fact: Counter = Counter()
    buckets: Counter = Counter()
    unknown: list[str] = []
    for p in phrases:
        e = Engine(sc, eng.nlu)
        t = e.handle(p)
        st = t.decision.style
        if t.decision.reveal:
            buckets["факт"] += 1
            for k in t.decision.reveal:
                per_fact[k] += 1
        elif st is Style.ACK:
            buckets["речевой акт"] += 1
        elif st is Style.MISHEAR:
            buckets["не расслышала"] += 1
            unknown.append(p)
        else:
            buckets["не знаю"] += 1
            unknown.append(p)

    n = len(phrases)
    print(f"\nСценарий {sc.id} · реплик {n} · понимание {eng.nlu.name}")
    for k in ("факт", "речевой акт", "не знаю", "не расслышала"):
        print(f"  {k:16} {buckets[k]:3}  ({100 * buckets[k] // n:2}%)")
    print("\nпопадания по фактам:")
    for k in sc.facts:
        mark = "" if per_fact[k] else "   ← ни одного"
        print(f"  {per_fact[k]:3}  {k}{mark}")
    covered = [k for k in sc.critical if per_fact[k]]
    print(f"\nобязательные факты покрыты: {len(covered)}/{len(sc.critical)}")
    if unknown:
        print("\nбез ответа по существу (кандидаты в банк вопросов):")
        for p in unknown[:25]:
            print(f"  · {p}")


def main() -> None:
    ap = argparse.ArgumentParser(prog="caller.cli")
    ap.add_argument("command", choices=["chat", "run", "eval"])
    ap.add_argument("scenario")
    ap.add_argument("-f", "--file")
    ap.add_argument("--nlu", default="lexical", choices=["lexical", "llm"])
    ap.add_argument("--profile", choices=["calm", "normal", "hard"])
    ap.add_argument("--seed", type=int, default=7)
    a = ap.parse_args()

    sc = Scenario.load(a.scenario)
    if a.profile:
        from .scenario import Profile
        sc.profile = Profile.preset(a.profile)
    eng = Engine(sc, a.nlu, seed=a.seed)

    if a.command == "chat":
        cmd_chat(eng)
        return
    if not a.file:
        sys.exit("для run и eval нужен -f файл с репликами")
    phrases = read_lines(a.file)
    (cmd_run if a.command == "run" else cmd_eval)(eng, phrases)


if __name__ == "__main__":
    main()
