# -*- coding: utf-8 -*-
"""
Оценка без разметки: сценарий против банка, который его не видел.

Для каждого сценария берём его собственные формулировки вопросов и
классифицируем их банком, из которого этот сценарий вычеркнут. Метка
известна заранее — это слот факта, которому формулировка принадлежит.

Меряется ровно то, что произойдёт с новой ситуацией пользователя: попадёт
ли непривычно заданный вопрос в нужный слот силами остального корпуса.
Ключ к модели для этого не нужен.

  python3 -m bench.loo            весь корпус
  python3 -m bench.loo -n 20      первые 20 сценариев, быстрая проверка
"""

from __future__ import annotations

import argparse
import statistics
from collections import Counter

from .common import load_bank


def run(limit: int | None = None, verbose: bool = False) -> dict:
    _, loaded, bank = load_bank()

    top1 = top3 = total = 0
    per_scenario: list[tuple[float, str]] = []
    confusion: Counter = Counter()
    misses: list[tuple[str, str, str, str]] = []

    for sid in sorted(loaded.scenarios)[:limit]:
        sc = loaded.scenarios[sid]
        # кандидаты ограничены слотами сценария: заявитель физически не может
        # ответить о том, чего у него нет, — так же работает и рантайм
        allowed = frozenset(sc.by_slot)
        hit = seen = 0
        for fact in sc.facts.values():
            for q in fact.questions:
                ranked = bank.top(q, 3, allowed=allowed, without=sid)
                got = [s for s, _ in ranked]
                seen += 1
                total += 1
                if got and got[0] == fact.slot:
                    hit += 1
                    top1 += 1
                elif got:
                    confusion[(fact.slot, got[0])] += 1
                    if len(misses) < 400:
                        misses.append((sid, q, fact.slot, got[0]))
                if fact.slot in got:
                    top3 += 1
        if seen:
            per_scenario.append((100 * hit / seen, sid))

    per_scenario.sort()
    out = {
        "total": total,
        "top1": round(100 * top1 / total, 1) if total else 0.0,
        "top3": round(100 * top3 / total, 1) if total else 0.0,
        "median": round(statistics.median(p for p, _ in per_scenario), 1),
        "worst": per_scenario[:5],
        "best": per_scenario[-3:],
        "confusion": confusion.most_common(15),
        "misses": misses,
    }
    if verbose:
        for sid, q, want, got in misses[:40]:
            print(f"  {sid:<18} {q[:46]:<46} {want:<18} -> {got}")
    return out


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("-n", type=int, default=None, help="сколько сценариев")
    ap.add_argument("-v", action="store_true", help="показать промахи")
    args = ap.parse_args()

    r = run(args.n, args.v)
    print(f"\nформулировок:        {r['total']}")
    print(f"top-1 слот верный:   {r['top1']}%")
    print(f"верный слот в top-3: {r['top3']}%")
    print(f"медиана по сценарию: {r['median']}%")
    print("\nхудшие сценарии: " + ", ".join(f"{s} {p:.0f}%" for p, s in r["worst"]))
    print("лучшие сценарии: " + ", ".join(f"{s} {p:.0f}%" for p, s in r["best"]))
    print("\nчаще всего путается (эталон -> выдано):")
    for (want, got), n in r["confusion"]:
        print(f"  {n:>4}  {want:<22} -> {got}")


if __name__ == "__main__":
    main()
