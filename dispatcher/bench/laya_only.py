# -*- coding: utf-8 -*-
"""
Понимание целиком на laya против банка со сплавом.

Сравнение честное без оговорок: laya корпуса не видела вовсе, поэтому её
результат на формулировках сценариев сопоставим с тем, что банк показывает
с вычеркнутым сценарием.

Проверяется главная ставка: хватает ли одной модели, чтобы новая ситуация
работала вообще без формулировок.

  HF_HOME=$PWD/models python3 -m bench.laya_only --device cuda -n 12
"""

from __future__ import annotations

import argparse
import statistics
import time
from collections import Counter

from dispatcher.data.loader import load_all
from dispatcher.data.ontology import Ontology
from dispatcher.nlu.laya_nlu import LayaUnderstander


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--device", default="cpu")
    ap.add_argument("-n", type=int, default=12, help="сколько сценариев")
    ap.add_argument("--confidence", type=float, default=0.35)
    args = ap.parse_args()

    onto = Ontology.load()
    loaded = load_all(onto)

    from laya import Router

    router = Router(device=args.device, default="multilingual")

    correct = total = 0
    refusals = 0
    lat: list[float] = []
    confusion: Counter = Counter()
    per_scenario: list[tuple[float, str]] = []
    n_options: list[int] = []

    for sid in sorted(loaded.scenarios)[: args.n]:
        sc = loaded.scenarios[sid]
        nlu = LayaUnderstander(
            scenario=sc, ontology=onto, router=router, min_confidence=args.confidence
        )
        n_options.append(len(sc.by_slot))
        hits = seen = 0

        for fact in sc.facts.values():
            for q in fact.questions:
                t = time.perf_counter()
                u = nlu.understand(q)
                lat.append(1000 * (time.perf_counter() - t))
                total += 1
                seen += 1
                if u.slots and u.slots[0] == fact.slot:
                    correct += 1
                    hits += 1
                elif not u.slots:
                    refusals += 1
                    confusion[(fact.slot, "отказ")] += 1
                else:
                    confusion[(fact.slot, u.slots[0])] += 1
        if seen:
            per_scenario.append((100 * hits / seen, sid))

    per_scenario.sort()
    print(f"\nформулировок {total } из {len (per_scenario )} сценариев")
    print(
        f"вариантов на сценарий: от {min (n_options )} до {max (n_options )}, "
        f"в среднем {statistics .mean (n_options ):.0f}"
    )
    print(f"верный слот:    {100 *correct /total :.1f}%")
    print(f"из них отказов: {100 *refusals /total :.1f}%")
    print(
        f"медиана по сценарию: {statistics .median (p for p ,_ in per_scenario ):.1f}%"
    )
    print("худшие: " + ", ".join(f"{s } {p :.0f}%" for p, s in per_scenario[:3]))
    print("лучшие: " + ", ".join(f"{s } {p :.0f}%" for p, s in per_scenario[-3:]))

    lat.sort()
    print(
        f"\nзадержка: медиана {statistics .median (lat ):.1f} мс, "
        f"p95 {lat [int (0.95 *(len (lat )-1 ))]:.1f} мс"
    )
    print("\nчаще всего путается:")
    for (gold, given), n in confusion.most_common(10):
        print(f"  {n :>4}  {gold :<22} -> {given }")


if __name__ == "__main__":
    main()
