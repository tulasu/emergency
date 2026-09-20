# -*- coding: utf-8 -*-
"""
Сравнение энкодеров и подбор веса сплава.

Метод тот же, что в bench/loo.py: формулировки сценария классифицируются
банком, из которого этот сценарий вычеркнут. Разметка не нужна, меряется
ровно то, что случится с новой ситуацией пользователя.

  python3 -m bench.encoders                          всё, что заработает
  python3 -m bench.encoders -m e5-small:onnx         одна связка
  python3 -m bench.encoders -m rubert-tiny2:torch-cuda

Связка пишется как модель:бэкенд. Бэкенды: onnx (int8), onnx-fp32,
torch-cpu, torch-cuda.
"""

from __future__ import annotations

import argparse
import statistics
import time
from collections import Counter

import numpy as np
from typing import TYPE_CHECKING

from dispatcher.nlu import encoder as enc

from .common import load_bank, make_vectors

if TYPE_CHECKING:
    from dispatcher.nlu.bank import LexicalBank
    from dispatcher.nlu.vectors import VectorBank

ALPHAS = (0.0, 0.3, 0.45, 0.6, 0.75, 0.85, 1.0)


def _queries(loaded):
    """Все формулировки корпуса с эталонным слотом и сценарием-владельцем."""
    rows = []
    for sid in sorted(loaded.scenarios):
        sc = loaded.scenarios[sid]
        for fact in sc.facts.values():
            for q in fact.questions:
                rows.append((sid, q, fact.slot))
    return rows


def evaluate(
    loaded, lex: LexicalBank, vb: VectorBank | None, qvecs: np.ndarray | None, rows
) -> dict[float, dict]:
    """Точность при каждом весе сплава за один проход по корпусу."""
    slot_pos = {s: i for i, s in enumerate(vb.slots)} if vb else {}
    results = {
        a: {"hit": 0, "top3": 0, "per": Counter(), "seen": Counter(), "conf": Counter()}
        for a in ALPHAS
    }
    total = 0

    by_scenario: dict[str, list[int]] = {}
    for i, (sid, _, _) in enumerate(rows):
        by_scenario.setdefault(sid, []).append(i)

    for sid, idx in by_scenario.items():
        sc = loaded.scenarios[sid]
        allowed = frozenset(sc.by_slot)

        vec_scores = None
        if vb is not None:
            sims = qvecs[idx] @ vb.matrix.T  # (k, n)
            src = vb._source_id(sid)
            if src >= 0:
                sims[:, vb.sources == src] = -1.0
            vec_scores = np.maximum.reduceat(sims, vb.starts[:-1], axis=1)

        for row, i in enumerate(idx):
            _, q, gold = rows[i]
            total += 1
            lex_scores = lex.score(q, allowed=allowed, without=sid)

            for a in ALPHAS:
                if a == 0.0 or vb is None:
                    merged = lex_scores
                else:
                    merged = {}
                    for slot in allowed:
                        v = vec_scores[row, slot_pos[slot]] if slot in slot_pos else 0.0
                        merged[slot] = a * max(v, 0.0) + (1 - a) * lex_scores.get(
                            slot, 0.0
                        )
                ranked = sorted(merged.items(), key=lambda kv: -kv[1])[:3]
                got = [s for s, _ in ranked]
                r = results[a]
                r["seen"][sid] += 1
                if got and got[0] == gold:
                    r["hit"] += 1
                    r["per"][sid] += 1
                elif got:
                    r["conf"][(gold, got[0])] += 1
                if gold in got:
                    r["top3"] += 1

    for a, r in results.items():
        r["top1_pct"] = round(100 * r["hit"] / total, 1)
        r["top3_pct"] = round(100 * r["top3"] / total, 1)
        r["median"] = round(
            statistics.median(100 * r["per"][s] / r["seen"][s] for s in r["seen"]), 1
        )
    return results


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    default = [f"{m}:onnx" for m, sp in enc.REGISTRY.items() if sp.onnx_dir]
    default += [f"{m}:torch-cpu" for m in enc.REGISTRY]
    if enc.available_device() == "cuda":
        default += [f"{m}:torch-cuda" for m in enc.REGISTRY]

    ap.add_argument("-m", "--models", nargs="*", default=default)
    args = ap.parse_args()

    _, loaded, lex = load_bank()
    rows = _queries(loaded)
    print(
        f"формулировок {len(rows)}, видеокарта "
        f"{'есть' if enc.available_device() == 'cuda' else 'нет'}\n"
    )

    base = evaluate(loaded, lex, None, None, rows)[0.0]
    print(
        f"{'лексика без модели':<34} top-1 {base['top1_pct']:5.1f}%  "
        f"top-3 {base['top3_pct']:5.1f}%  медиана {base['median']:5.1f}%"
    )

    for spec in args.models:
        try:
            t0 = time.perf_counter()
            e, vb = make_vectors(spec, loaded.questions, loaded.sources)
            load_s = time.perf_counter() - t0

            t0 = time.perf_counter()
            qvecs = e.encode([q for _, q, _ in rows], kind="query")
            query_s = time.perf_counter() - t0

            # задержка одной реплики, как в реальном звонке
            warm = ["на каком этаже горит"]
            for _ in range(3):
                e.encode(warm)
            t0 = time.perf_counter()
            for _ in range(30):
                e.encode(warm)
            per_ms = 1000 * (time.perf_counter() - t0) / 30

            res = evaluate(loaded, lex, vb, qvecs, rows)
            print(
                f"\n{e.name}  ({e.dim} измерений, сборка {load_s:.1f} с, "
                f"векторы вопросов {query_s:.1f} с, реплика {per_ms:.1f} мс)"
            )
            for a in ALPHAS:
                r = res[a]
                tag = (
                    "только лексика"
                    if a == 0
                    else "только векторы" if a == 1 else f"сплав α={a}"
                )
                print(
                    f"   {tag:<20} top-1 {r['top1_pct']:5.1f}%  "
                    f"top-3 {r['top3_pct']:5.1f}%  медиана {r['median']:5.1f}%"
                )
            best = max(ALPHAS, key=lambda a: res[a]["top1_pct"])
            print(
                f"   лучший α={best}, чаще всего путается: "
                + ", ".join(
                    f"{g}->{p}:{n}" for (g, p), n in res[best]["conf"].most_common(5)
                )
            )
        except Exception as exc:  # noqa: BLE001
            print(f"\n{spec}: не вышло — {type(exc).__name__}: {exc}")


if __name__ == "__main__":
    main()
