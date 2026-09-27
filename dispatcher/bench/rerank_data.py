# -*- coding: utf-8 -*-
"""Пары для реранкера: кандидаты top-5 банка без собственного сценария.

Только формулировки сценариев, никаких live/blind. Исключение сценария
предотвращает утечку точного совпадения формулировки в кандидаты. Если
правильного слота в top-5 нет, все пять пар отрицательные: в рантайме
реранкер должен уметь отказаться. Blind остаётся только для оценки.

  python3 -m bench.rerank_data --out data/build/rerank.jsonl
"""

from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path

from dispatcher.data.loader import DATA

from .common import load_bank


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--out", default=str(DATA / "build" / "rerank.jsonl"))
    ap.add_argument("--k", type=int, default=5)
    args = ap.parse_args()

    _, loaded, lex = load_bank()
    tally: Counter = Counter()
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    with out.open("w", encoding="utf-8") as f:
        for sid in sorted(loaded.scenarios):
            for fact in loaded.scenarios[sid].facts.values():
                for text in fact.questions:
                    cands = [slot for slot, _ in lex.top(text, args.k, without=sid)]
                    if not cands:
                        continue
                    label = cands.index(fact.slot) if fact.slot in cands else None
                    kind = "pos" if label is not None else "pos5miss"
                    f.write(json.dumps({"text": text, "cands": cands,
                                        "label": label, "kind": kind},
                                       ensure_ascii=False) + "\n")
                    tally[kind] += 1
    print(f"строк {sum(tally.values())} -> {out}")
    for kind in ("pos", "pos5miss"):
        print(f"  {kind:<9} {tally[kind]:>5}")


if __name__ == "__main__":
    main()
