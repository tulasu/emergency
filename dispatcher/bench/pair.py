# -*- coding: utf-8 -*-
"""Сравнить два blind-прогона по одним и тем же репликам.

  python3 -m bench.pair data/build/baseline-scenario.jsonl data/build/reranker-scenario.jsonl
"""

from __future__ import annotations

import argparse
import json
from collections import Counter, defaultdict
from itertools import zip_longest
from pathlib import Path

from dispatcher.data.loader import load_all


def compare(left: Path, right: Path) -> None:
    changed: Counter = Counter()
    totals = [Counter(), Counter()]
    facts = [defaultdict(set), defaultdict(set)]
    count = 0
    with left.open(encoding="utf-8") as a, right.open(encoding="utf-8") as b:
        for count, (first, second) in enumerate(zip_longest(a, b), 1):
            if first is None or second is None:
                raise ValueError(f"разное число реплик: строка {count}")
            x, y = json.loads(first), json.loads(second)
            key = lambda row: (row["scenario"], row["index"], row["text"])
            if key(x) != key(y):
                raise ValueError(f"непарные входные реплики на строке {count}: {key(x)} != {key(y)}")
            for n, row in enumerate((x, y)):
                totals[n][row["outcome"]] += 1
                facts[n][row["scenario"]].update(row["reveal"])
            if x["outcome"] != y["outcome"]:
                changed[x["outcome"], y["outcome"]] += 1
    loaded = load_all()
    critical = [
        {(sid, fact) for sid, keys in observed.items()
         for fact in keys if fact in loaded.scenarios[sid].critical}
        for observed in facts
    ]
    print(f"совпали {count} входных реплик")
    for n, path in enumerate((left, right)):
        print(f"{path}: {dict(totals[n])}; критичных {len(critical[n])}")
    print("смена исхода:", dict(changed))
    print("критичные приобретены:", sorted(critical[1] - critical[0]))
    print("критичные потеряны:", sorted(critical[0] - critical[1]))


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("baseline", type=Path)
    ap.add_argument("experiment", type=Path)
    args = ap.parse_args()
    compare(args.baseline, args.experiment)


if __name__ == "__main__":
    main()
