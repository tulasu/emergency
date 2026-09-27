# -*- coding: utf-8 -*-
"""
Подбор порога уверенности.

Эталон известен без разметки: слот формулировки — это слот факта, которому
она принадлежит. Сценарий вычёркивается из банка, поэтому цифры те же, что
получит новая ситуация пользователя.

Для тренажёра цена ошибок несимметрична. Ответить не тем фактом — значит
записать обучающемуся в карточку ложные данные, и он будет уверен, что всё
верно. Промолчать «не знаю» — потеря хода, не более. Поэтому порог ищется
по доле ложных ответов, а не по общей точности.

  python3 -m bench.thresholds
  python3 -m bench.thresholds -m e5-small:onnx --alpha 0.85
"""

from __future__ import annotations

import argparse

from .common import load_bank, make_vectors

GRID = [0.20, 0.25, 0.30, 0.35, 0.40, 0.45, 0.50, 0.55, 0.60, 0.70]


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--target", type=float, default=2.0, help="допустимая доля ложных ответов, %%"
    )
    ap.add_argument("-m", "--model", default=None, help="модель:бэкенд")
    ap.add_argument("--alpha", type=float, default=0.85, help="вес векторов")
    args = ap.parse_args()

    _, loaded, bank = load_bank()

    encoder = vectors = None
    if args.model:
        encoder, vectors = make_vectors(args.model, loaded.questions, loaded.sources)
        print(f"энкодер {encoder.name}, вес векторов {args.alpha}")

        # один проход по корпусу, решения на всех порогах сразу
    rows: list[tuple[float, bool]] = []  # (оценка лидера, верен ли он)
    for sid in sorted(loaded.scenarios):
        sc = loaded.scenarios[sid]
        scope = sc.slots
        pairs = [(fact, q) for fact in sc.facts.values() for q in fact.questions]
        if not pairs:
            continue

        vec_scores = None
        if vectors is not None:
            qv = encoder.encode([q for _, q in pairs], kind="query")
            sims = qv @ vectors.matrix.T
            src = vectors._source_id(sid)
            if src >= 0:
                sims[:, vectors.sources == src] = -1.0
            import numpy as np

            vec_scores = np.maximum.reduceat(sims, vectors.starts[:-1], axis=1)
            pos = {sl: i for i, sl in enumerate(vectors.slots)}

        for row, (fact, q) in enumerate(pairs):
            own = {fact.slot} | {m for m, v in sc.answers_for.items() if v == fact.slot}
            lex = bank.score(q, allowed=scope, without=sid)
            if vec_scores is None:
                merged = lex
            else:
                a = args.alpha
                merged = {
                    sl: a * max(vec_scores[row, pos[sl]], 0.0)
                    + (1 - a) * lex.get(sl, 0.0)
                    for sl in scope
                    if sl in pos
                }
            if merged:
                slot = max(merged, key=merged.get)
                rows.append((merged[slot], slot in own))
            else:
                rows.append((0.0, False))

    total = len(rows)
    print(f"формулировок {total }\n")
    print(
        f"{'порог':>6} {'отвечено':>9} {'верно':>8} {'ложных':>8} "
        f"{'молчим':>8}   доля ложных от всех"
    )
    best = None
    for t in GRID:
        answered = [ok for score, ok in rows if score >= t]
        right = sum(answered)
        wrong = len(answered) - right
        silent = total - len(answered)
        wrong_pct = 100 * wrong / total
        print(
            f"{t :>6.2f} {len (answered ):>9} {right :>8} {wrong :>8} "
            f"{silent :>8}   {wrong_pct :>5.1f}%"
        )
        if wrong_pct <= args.target and (best is None or right > best[1]):
            best = (t, right)

    if best:
        t, right = best
        print(
            f"\nпри допуске {args .target }% ложных лучший порог {t :.2f}: "
            f"верных ответов {right } ({100 *right /total :.1f}% корпуса)"
        )
    else:
        print(
            f"\nни один порог не укладывается в {args .target }% ложных — "
            f"нужен арбитр либо векторный слой"
        )


if __name__ == "__main__":
    main()
