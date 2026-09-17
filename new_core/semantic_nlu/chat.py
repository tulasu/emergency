# -*- coding: utf-8 -*-
"""
Интерактивный чат: semantic NLU + диалоговый Engine из trainer.

  cd new_core/semantic_nlu
  python -m chat ../trainer/scenarios/bilet04_call01.json

Пустая строка / Ctrl+C — выход. Неси дичь — заявитель ответит по сценарию.
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
TRAINER = ROOT.parent / "trainer"
if str(TRAINER) not in sys.path:
    sys.path.insert(0, str(TRAINER))
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from caller.engine import Engine  # noqa: E402
from caller.scenario import Scenario  # noqa: E402

from embedder import DEFAULT_MODEL, Embedder  # noqa: E402
from understand import SemanticUnderstander  # noqa: E402


def resolve_scenario(path: str) -> Path:
    p = Path(path)
    if not p.is_absolute():
        cand = (Path.cwd() / p).resolve()
        if cand.exists():
            return cand
        cand = (ROOT / p).resolve()
        if cand.exists():
            return cand
        # удобный шорткат: bilet04_call01 → scenarios/...
        short = TRAINER / "scenarios" / f"{path}.json"
        if short.exists():
            return short
        short = TRAINER / "scenarios" / path
        if short.exists():
            return short
    return p.resolve()


def main() -> None:
    ap = argparse.ArgumentParser(description="chat with semantic NLU + trainer Engine")
    ap.add_argument(
        "scenario",
        nargs="?",
        default="bilet04_call01",
        help="путь к JSON или id вроде bilet04_call01",
    )
    ap.add_argument("--model", default=DEFAULT_MODEL)
    ap.add_argument("--threshold", type=float, default=0.45)
    ap.add_argument("--margin", type=float, default=0.03)
    ap.add_argument("--strong", type=float, default=0.85,
                    help="accept top-1 without margin if score >= strong")
    ap.add_argument("--profile", choices=["calm", "normal", "hard"])
    ap.add_argument("--timing", action=argparse.BooleanOptionalAction, default=True)
    ap.add_argument("--debug", action="store_true",
                    help="печать score/second/status матча")
    a = ap.parse_args()

    sc_path = resolve_scenario(a.scenario)
    if not sc_path.exists():
        sys.exit(f"нет сценария: {sc_path}")

    print(f"loading scenario={sc_path.name} model={a.model} …")
    sc = Scenario.load(sc_path)
    if a.profile:
        from caller.scenario import Profile
        sc.profile = Profile.preset(a.profile)

    embedder = Embedder(a.model, device="cpu")
    load_ms = embedder.load()
    nlu = SemanticUnderstander(
        sc, embedder=embedder, threshold=a.threshold, margin=a.margin,
        strong=a.strong,
    )
    build_ms = nlu.build()
    print(f"ready (cold load {load_ms:.0f}ms, prototypes {build_ms:.0f}ms)")
    print(f"facts: {', '.join(sc.facts)}")
    print("пустая строка — выход\n")

    eng = Engine(sc, nlu)
    print(f"З: {eng.opening()}\n")

    while True:
        try:
            line = input("О: ").strip()
        except (EOFError, KeyboardInterrupt):
            print()
            break
        if not line:
            break

        if a.debug:
            m = nlu.match(line)
            print(f"   [match status={m.status} score={m.score:.3f} "
                  f"second={m.second:.3f} keys={m.keys} "
                  f"neg={m.negated} {m.total_ms:.1f}ms]")

        t = eng.handle(line)
        print(f"З: {t.reply.text}")
        print(f"   [{t.decision.style.value} · {t.decision.mood.name} · "
              f"{','.join(t.decision.reveal) or '—'} · {t.understanding.source}]")
        if a.timing:
            print(f"   [{t.timing.format()}]")
        print()

    print(eng.close())


if __name__ == "__main__":
    main()
