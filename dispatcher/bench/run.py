# -*- coding: utf-8 -*-
"""
Сквозной прогон слепых наборов через каскад.

Слепые наборы — реплики оператора, которых нет в банке формулировок.
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
  python3 -m bench.run --ensemble majority --voters laya,llm --device cuda
      голосование как в проде (нужен llama-server на 8081)
  python3 -m bench.run --ensemble majority --candidate-scope all
      воспроизведение старого top-5 по всем слотам
  python3 -m bench.run --ensemble majority --outcomes data/build/base.jsonl
      сохранить исходы по репликам для bench.pair
"""

from __future__ import annotations

import argparse
import json
import statistics
import time
from collections import Counter
from pathlib import Path

from dispatcher.data.loader import DATA
from dispatcher.dialog.policy import decide
from dispatcher.dialog.state import CallState
from dispatcher.nlu.cascade import Cascade, Thresholds
from dispatcher.nlu.ensemble import Ensemble
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
    ensemble: str | None = None,
    voters: str = "laya,llm",
    laya_min: float = 0.5,
    rerank_path: str | None = None,
    rerank_min: float = 0.3,
    rerank_margin: float = 0.02,
    candidate_scope: str = "scenario",
    outcomes: str | None = None,
    softmax_path: str | None = None,
    lora_base: str | None = None,
    lora_path: str | None = None,
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

    voter_list = []
    if ensemble:
        for name in (voters or "").split(","):
            name = name.strip()
            if name == "laya":
                from dispatcher.nlu.laya_arbiter import LayaArbiter

                voter_list.append(
                    LayaArbiter(ontology=onto, device=device, min_confidence=laya_min)
                )
            elif name == "llm":
                from dispatcher.nlu.llm_arbiter import LlmArbiter

                voter_list.append(LlmArbiter(onto))
            elif name == "reranker":
                from dispatcher.nlu.reranker import CrossEncoderScorer, RerankerVoter

                voter_list.append(RerankerVoter(
                    CrossEncoderScorer(onto, rerank_path or str(DATA / "build" / "reranker"), device),
                    min_score=rerank_min, min_margin=rerank_margin,
                ))
            elif name == "lora":
                from dispatcher.nlu.lora_arbiter import LoraLlmArbiter

                voter_list.append(LoraLlmArbiter(
                    onto, lora_base or str(DATA / "build" / "qwen3-1.7b"),
                    lora_path or str(DATA / "build" / "lora-arbiter-clean"),
                ))
            elif name:
                raise ValueError(f"нет голосующего {name}")
        print(f"голосование {ensemble} [{voters}], кандидаты {candidate_scope}")

    softmax = None
    if softmax_path:
        if fresh:
            raise ValueError("softmax не поддерживает --fresh: обучен на всём scenario train")
        from dispatcher.nlu.softmax import SoftmaxScorer

        softmax = SoftmaxScorer(softmax_path, device)
        if set(softmax.slots) != set(onto.slots):
            raise ValueError("голова обучена на другой онтологии")
        print(f"softmax 88+1: {softmax_path}; бедные слоты у банка")
    tally: Counter = Counter()
    latencies: list[float] = []
    per_scenario: list[tuple[float, str]] = []
    crit_got = crit_all = 0
    unresolved: list[tuple[str, str]] = []
    outcome_rows: list[dict] = []

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
            softmax=softmax,
        )
        nlu = Ensemble(cascade, voter_list, rule=ensemble,
                       candidate_scope=candidate_scope) if voter_list else cascade
        state = CallState()
        answered = 0
        revealed: set[str] = set()

        for index, line in enumerate(lines):
            t0 = time.perf_counter()
            u = nlu.understand(line)
            d = decide(u, state, sc)
            latencies.append(1000 * (time.perf_counter() - t0))

            outcome = ("ответ" if d.reveal else
                       "речевой акт" if d.style is Style.ACK else
                       "не расслышала" if d.style is Style.MISHEAR else
                       "не знаю" if u.slots else "мимо")
            tally[outcome] += 1
            if d.reveal:
                if u.source == "ensemble":
                    tally["  из них ансамблем"] += 1
                if u.source == "arbiter":
                    tally["  из них арбитром"] += 1
                answered += 1
                revealed.update(d.reveal)
            elif outcome == "мимо" and len(unresolved) < 300:
                unresolved.append((sid, line))
            if outcomes is not None:
                outcome_rows.append({
                    "scenario": sid, "index": index, "text": line,
                    "outcome": outcome, "reveal": d.reveal, "slots": u.slots,
                })

        per_scenario.append((100 * answered / len(lines), sid))
        crit_got += len(revealed & set(sc.critical))
        crit_all += len(sc.critical)

    if outcomes is not None:
        path = Path(outcomes)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text("".join(json.dumps(row, ensure_ascii=False) + "\n"
                                for row in outcome_rows), encoding="utf-8")
        print(f"исходы: {path}")
    # подк ключи «  из них …» — разбор ответов, а не реплики: в итог не входят
    total = sum(v for k, v in tally.items() if not k.startswith("  "))
    per_scenario.sort()
    print(f"\nреплик {total} из {len(per_scenario)} сценариев")
    for kind in (
        "ответ",
        "  из них арбитром",
        "  из них ансамблем",
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
    if voter_list:
        for v in voter_list:
            print(
                f"голос {type(v).__name__}: вызовов {v.calls}, "
                f"отказов {getattr(v, 'refusals', '-')}, сбоев {v.failures}"
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
        "--ensemble",
        default=None,
        choices=["majority", "strict", "cascade+", "llm-lead"],
        help="голосование каскад + перечисленные voter (llm требует llama-server)",
    )
    ap.add_argument("--voters", default="laya,llm", help="laya,llm,reranker,lora через запятую")
    ap.add_argument("--laya-min", type=float, default=0.5)
    ap.add_argument("--rerank-path", default=None, help="каталог обученного реранкера")
    ap.add_argument("--rerank-min", type=float, default=0.3)
    ap.add_argument("--rerank-margin", type=float, default=0.02)
    ap.add_argument("--lora-base", help="локальная Qwen3-1.7B для QLoRA")
    ap.add_argument("--lora-path", help="путь к QLoRA-адаптеру")
    ap.add_argument("--candidate-scope", choices=["scenario", "all"], default="scenario",
                    help="слоты сценария как в dialog; all — исторический топ-5 из 88")
    ap.add_argument("--softmax-path", help="заменить скорер населённых слотов каскада")
    ap.add_argument("--outcomes", help="JSONL исходов по каждой реплике для парного сравнения")
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
        args.ensemble,
        args.voters,
        args.laya_min,
        args.rerank_path,
        args.rerank_min,
        args.rerank_margin,
        args.candidate_scope,
        args.outcomes,
        args.softmax_path,
        args.lora_base,
        args.lora_path,
    )


if __name__ == "__main__":
    main()
