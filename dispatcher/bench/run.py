# -*- coding: utf-8 -*-
"""
Сквозной прогон слепых наборов через каскад.

Слепые наборы — 6011 реплик оператора, которых нет в банке формулировок.
Считается не «доля реплик, на которые дан хоть какой-то ответ» (так мерило
старое ядро, и эта цифра не отличала верный факт от неверного), а разбор по
исходам:

  ответ         выдан факт из сценария
  не знаю       вопрос понят, но такого сведения у заявителя нет
  не расслышала реплика без смысловых слов
  мимо          ничего не нашлось, хотя тема в реплике есть

Последняя строка — та, которую надо уменьшать: это вопросы, на которые
заявитель мог бы ответить, но система не поняла.

  python3 -m bench.run                       лексика, весь корпус
  python3 -m bench.run -m e5-small:onnx      со сплавом
  python3 -m bench.run -n 15 --show          посмотреть, что не разобралось
"""

from __future__ import annotations

import argparse
import statistics
import time
from collections import Counter

from dispatcher.data.loader import DATA
from dispatcher.dialog.policy import decide
from dispatcher.dialog.state import CallState
from dispatcher.nlu.cascade import Cascade, Thresholds
from dispatcher.types import Style
from .common import load_bank

BLIND = DATA / "blind"


def phrases(sid: str) -> list[str]:
    out: list[str] = []
    for path in sorted(BLIND.glob(f"blind_{sid}*.txt")):
        for line in path.read_text(encoding="utf-8").splitlines():
            line = line.strip().lstrip("-•*0123456789. ").strip()
            if line and not line.startswith("#"):
                out.append(line)
    return out


def run(
    models: str | None,
    limit: int | None,
    show: bool,
    fresh: bool,
    gap: float | None = None,
    arbiter_kind: str | None = None,
    device: str = "cpu",
    floor: float | None = None,
    gap_low: float | None = None,
) -> None:
    onto, loaded, lex = load_bank()

    encoder = vectors = None
    if models:
        from .common import make_vectors

        encoder, vectors = make_vectors(models, loaded.questions, loaded.sources)
        print(f"энкодер: {encoder.name}")

    arbiter = None
    if arbiter_kind == "laya":
        from dispatcher.nlu.laya_arbiter import LayaArbiter

        arbiter = LayaArbiter(ontology=onto, device=device)
        print(f"арбитр: laya на {device}")

    tally: Counter = Counter()
    latencies: list[float] = []
    per_scenario: list[tuple[float, str]] = []
    crit_got = crit_all = 0
    unresolved: list[tuple[str, str]] = []

    for sid in sorted(loaded.scenarios)[:limit]:
        sc = loaded.scenarios[sid]
        lines = phrases(sid)
        if not lines:
            continue
        th = Thresholds.preset(
            Cascade(
                scenario=sc,
                lexical=lex,
                ontology=onto,
                encoder=encoder,
                vectors=vectors,
            ).kind()
        )
        if gap is not None:
            th.gap_high = gap
            th.gap_low = gap / 3
        if floor is not None:
            th.floor = floor
        if gap_low is not None:
            th.gap_low = gap_low
        cascade = Cascade(
            scenario=sc,
            lexical=lex,
            ontology=onto,
            encoder=encoder,
            vectors=vectors,
            thresholds=th,
            arbiter=arbiter,
            without=sid if fresh else "",
        )
        state = CallState()
        answered = 0
        revealed: set[str] = set()

        for line in lines:
            t0 = time.perf_counter()
            u = cascade.understand(line)
            d = decide(u, state, sc)
            latencies.append(1000 * (time.perf_counter() - t0))

            if d.reveal:
                tally["ответ"] += 1
                if u.source == "arbiter":
                    tally["  из них арбитром"] += 1
                answered += 1
                revealed.update(d.reveal)
            elif d.style is Style.ACK:
                tally["речевой акт"] += 1
            elif d.style is Style.MISHEAR:
                tally["не расслышала"] += 1
            elif u.slots:
                tally["не знаю"] += 1
            else:
                tally["мимо"] += 1
                if len(unresolved) < 300:
                    unresolved.append((sid, line))

        per_scenario.append((100 * answered / len(lines), sid))
        crit_got += len(revealed & set(sc.critical))
        crit_all += len(sc.critical)

    total = sum(tally.values())
    per_scenario.sort()
    print(f"\nреплик {total} из {len(per_scenario)} сценариев")
    for kind in (
        "ответ",
        "  из них арбитром",
        "не знаю",
        "речевой акт",
        "не расслышала",
        "мимо",
    ):
        if kind not in tally and kind.startswith(" "):
            continue
        n = tally[kind]
        print(f"  {kind:<16} {n:>5}  {100 * n / total:5.1f}%")
    print(
        f"\nкритичных фактов добыто {crit_got} из {crit_all} "
        f"({100 * crit_got / crit_all:.1f}%)"
    )
    print(
        f"медиана ответов по сценарию {statistics.median(p for p, _ in per_scenario):.1f}%"
    )
    print("худшие: " + ", ".join(f"{s} {p:.0f}%" for p, s in per_scenario[:4]))
    print("лучшие: " + ", ".join(f"{s} {p:.0f}%" for p, s in per_scenario[-3:]))

    latencies.sort()
    p50 = latencies[len(latencies) // 2]
    p95 = latencies[int(0.95 * (len(latencies) - 1))]
    print(f"\nпонимание: медиана {p50:.2f} мс, p95 {p95:.2f} мс")
    if arbiter is not None:
        print(
            f"арбитр: вызовов {arbiter.calls}, отказов {arbiter.refusals}, "
            f"сбоев {arbiter.failures}"
        )

    if show and unresolved:
        print(f"\nне разобрано ({len(unresolved)} показано):")
        for sid, line in unresolved[:40]:
            print(f"  {sid:<18} {line}")


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("-m", "--model", default=None, help="модель:бэкенд")
    ap.add_argument("-n", type=int, default=None, help="сколько сценариев")
    ap.add_argument("--show", action="store_true", help="показать неразобранное")
    ap.add_argument(
        "--fresh", action="store_true", help="вычеркнуть формулировки самого сценария"
    )
    ap.add_argument(
        "--gap",
        type=float,
        default=None,
        help="переопределить разрыв, при котором отвечаем",
    )
    ap.add_argument(
        "--arbiter",
        default=None,
        choices=["laya"],
        help="подключить арбитра серой зоны",
    )
    ap.add_argument("--device", default="cpu", help="cpu или cuda для арбитра")
    ap.add_argument(
        "--floor", type=float, default=None, help="переопределить нижнюю отсечку оценки"
    )
    ap.add_argument(
        "--gap-low",
        type=float,
        default=None,
        help="нижняя граница серой зоны: ниже неё сразу «не знаю»",
    )
    args = ap.parse_args()
    run(
        args.model,
        args.n,
        args.show,
        args.fresh,
        args.gap,
        args.arbiter,
        args.device,
        args.floor,
        args.gap_low,
    )


if __name__ == "__main__":
    main()
