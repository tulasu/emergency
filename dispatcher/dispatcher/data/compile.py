# -*- coding: utf-8 -*-
"""
Отчёт о корпусе: сценарии + онтология.

  python3 -m dispatcher.data.compile

Банк формулировок никуда не пишется — он строится в памяти из сценариев.
Векторы банка появляются на следующем этапе.
"""

from __future__ import annotations

from collections import Counter

from .loader import Loaded, load_all
from .ontology import Ontology


def report(onto: Ontology, loaded: Loaded) -> int:
    facts = sum(len(s.facts) for s in loaded.scenarios.values())
    used = {sl for s in loaded.scenarios.values() for sl in s.by_slot}
    unmapped = [i for i in loaded.issues if i.kind == "unmapped"]
    crowded = [i for i in loaded.issues if i.kind == "crowded"]

    print(f"сценариев      {len(loaded.scenarios)}")
    print(f"фактов         {facts}")
    print(f"слотов         {len(used)} задействовано из {len(onto.slots)}")
    print(
        f"формулировок   {sum(len(v) for v in loaded.questions.values())} "
        f"после дедупликации"
    )

    sizes = Counter(
        len(v) for s in loaded.scenarios.values() for v in s.by_slot.values()
    )
    print("фактов на слот " + ", ".join(f"{k}:{v}" for k, v in sorted(sizes.items())))

    idle = sorted(set(onto.slots) - used)
    if idle:
        print(f"\nслоты без единого факта ({len(idle)}): {', '.join(idle)}")

    thin = sorted((len(v), k) for k, v in loaded.questions.items() if len(v) < 10)
    if thin:
        print("\nслоты с бедным банком (<10 формулировок), риск промаха:")
        for n, k in thin:
            print(f"  {n:>3}  {k}")

    if crowded:
        print(
            f"\nслоты с тремя и более фактами ({len(crowded)}) — "
            f"ответ будет из трёх кусков:"
        )
        for i in crowded:
            print(f"  {i.scenario:<20} {i.detail}: {i.key}")

    if unmapped:
        print(f"\nНЕ РАЗЛОЖЕНО ПО СЛОТАМ ({len(unmapped)}):")
        for i in unmapped:
            print(f"  {i.scenario:<20} {i.key:<28} {i.detail}")
        return 1
    print("\nвсе факты разложены по слотам")
    return 0


def main() -> int:
    onto = Ontology.load()
    loaded = load_all(onto)
    return report(onto, loaded)


if __name__ == "__main__":
    raise SystemExit(main())
