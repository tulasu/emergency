# -*- coding: utf-8 -*-
"""
Запуск ядра.

  python3 -m dispatcher.cli chat bilet04_call01
  python3 -m dispatcher.cli chat bilet04_call01 --model rubert-tiny2
  python3 -m dispatcher.cli replay bilet04_call01 -f examples/dialog.txt
  python3 -m dispatcher.cli replay bilet04_call01 -f examples/dialog.txt --stream
  python3 -m dispatcher.cli slots bilet04_call01
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from .data.loader import load_all
from .data.ontology import Ontology
from .nlu.bank import LexicalBank
from .nlu.cascade import Cascade
from .session import Session


def _build(
    sid: str,
    model: str | None,
    device: str | None,
    fresh: bool,
    backend: str = "auto",
    nlu: str = "bank",
    arbiter_kind: str | None = None,
    thin_rescue: bool = False,
) -> tuple[Session, Ontology]:
    onto = Ontology.load()
    loaded = load_all(onto)
    if sid not in loaded.scenarios:
        sys.exit(
            f"нет сценария {sid}; есть {len(loaded.scenarios)} штук, "
            f"например {sorted(loaded.scenarios)[0]}"
        )
    sc = loaded.scenarios[sid]

    if nlu == "laya":
        # понимание целиком на модели: банк формулировок не нужен вовсе.
        # Так работает ситуация, для которой ещё ничего не написано
        from .nlu.laya_nlu import LayaUnderstander

        understander = LayaUnderstander(
            scenario=sc, ontology=onto, device=device or "cpu"
        )
        return Session.open(sc, understander, onto), onto

    bank = LexicalBank.build(loaded.questions, loaded.sources)

    encoder = vectors = None
    if model:
        from .nlu import encoder as enc
        from .nlu.vectors import VectorBank

        encoder = enc.build(
            model,
            backend=backend,
            device=device
            or (
                "cuda"
                if backend == "torch" and enc.available_device() == "cuda"
                else "cpu"
            ),
        )
        vectors = VectorBank.build(loaded.questions, loaded.sources, encoder)

    arbiter = None
    if arbiter_kind == "laya":
        from .nlu.laya_arbiter import LayaArbiter

        print("поднимаю арбитра, это до минуты...", flush=True)
        arbiter = LayaArbiter(
            ontology=onto, device=device or "cpu", min_confidence=0.80
        )
        print("готово", flush=True)

    cascade = Cascade(
        scenario=sc,
        lexical=bank,
        ontology=onto,
        encoder=encoder,
        vectors=vectors,
        arbiter=arbiter,
        thin_rescue=thin_rescue and arbiter is not None,
        without=sc.id if fresh else "",
    )
    return Session.open(sc, cascade, onto), onto


def cmd_chat(args) -> None:
    session, _ = _build(
        args.scenario,
        args.model,
        args.device,
        args.fresh,
        args.backend,
        args.nlu,
        args.arbiter,
        args.thin_rescue,
    )
    print(f"— {session.opening()}\n")
    print("(пустая строка или Ctrl-D — конец)\n")
    while True:
        try:
            line = input("оператор> ").strip()
        except (EOFError, KeyboardInterrupt):
            break
        if not line:
            break
        reply = session.on_final(line)
        u = session.turns[-1].understanding
        print(f"заявитель> {reply.text}")
        if args.verbose:
            print(
                f"           [{u.source} {u.score:.2f} {u.act.value} "
                f"{u.slots} -> {u.keys} {u.latency_ms:.1f} мс]"
            )
    print(f"\nходов: {len(session.turns)}")
    session.close()


def cmd_replay(args) -> None:
    session, _ = _build(
        args.scenario,
        args.model,
        args.device,
        args.fresh,
        args.backend,
        args.nlu,
        args.arbiter,
        args.thin_rescue,
    )
    lines = [
        l.strip() for l in Path(args.file).read_text(encoding="utf-8").splitlines()
    ]
    lines = [l for l in lines if l and not l.startswith("#")]

    print(f"— {session.opening()}\n")
    for line in lines:
        if args.stream:
            for i in range(6, len(line), 5):
                session.on_partial(line[:i])
        reply = session.on_final(line)
        u = session.turns[-1].understanding
        print(f"оператор>  {line}")
        print(f"заявитель> {reply.text}")
        if args.verbose:
            print(
                f"           [{u.source} {u.score:.2f} {u.slots} "
                f"{u.latency_ms:.1f} мс]"
            )
    _summary(session)


def cmd_slots(args) -> None:
    onto = Ontology.load()
    loaded = load_all(onto)
    sc = loaded.scenarios[args.scenario]
    print(f"{sc.id}: {sc.meta.get('situation', '')}\n")
    for slot, keys in sorted(sc.by_slot.items()):
        label = onto.slots[slot].label
        for key in keys:
            fact = sc.facts[key]
            mark = "по запросу" if fact.disclosure.value == "on_request" else ""
            crit = "критичный" if key in sc.critical else ""
            tags = " ".join(t for t in (crit, mark) if t)
            print(f"  {slot:<22} {label:<28} {key:<24} {tags}")
            print(f"  {'':<22} {fact.answers['plain'][:70]}")
    if sc.answers_for:
        print("\n  замещения (слота нет, отвечает другой):")
        for missing, stand_in in sorted(sc.answers_for.items()):
            print(f"    {missing:<22} -> {stand_in}")


def cmd_lint(args) -> None:
    from .data.authoring import lint, render_lint

    onto = Ontology.load()
    loaded = load_all(onto)
    if args.scenario not in loaded.scenarios:
        sys.exit(f"нет сценария {args.scenario}")
    sc = loaded.scenarios[args.scenario]
    bank = LexicalBank.build(loaded.questions, loaded.sources)
    print(render_lint(sc, lint(sc, bank, onto), onto))


def cmd_suggest(args) -> None:
    from .data.authoring import suggest_slot

    onto = Ontology.load()
    loaded = load_all(onto)
    bank = LexicalBank.build(loaded.questions, loaded.sources)
    print(f"«{args.text}»\n")
    for slot, label, score in suggest_slot(args.text, bank, onto):
        print(f"  {score:5.2f}  {slot:<24} {label}")


def _summary(session: Session) -> None:
    answered = sum(1 for t in session.turns if t.decision.reveal)
    unknown = sum(
        1 for t in session.turns if t.understanding.slots and not t.understanding.keys
    )
    lat = sorted(t.understanding.latency_ms for t in session.turns)
    p95 = lat[int(0.95 * (len(lat) - 1))] if lat else 0.0
    got = {k for t in session.turns for k in t.decision.reveal}
    critical = set(session.scenario.critical)
    print(
        f"\nходов {len(session.turns)}, ответов по существу {answered}, "
        f"«не знаю» по делу {unknown}"
    )
    print(f"критичных фактов добыто {len(got & critical)} из {len(critical)}")
    if critical - got:
        print("  не спросили: " + ", ".join(sorted(critical - got)))
    print(f"понимание: медиана {lat[len(lat)//2]:.1f} мс, p95 {p95:.1f} мс")


def main() -> None:
    # общие флаги живут в родителе, иначе argparse требует ставить их
    # перед подкомандой — неочевидно и легко наступить
    common = argparse.ArgumentParser(add_help=False)
    common.add_argument("--model", help="энкодер, например e5-small")
    common.add_argument(
        "--backend", default="auto", help="auto, onnx, onnx-fp32 или torch"
    )
    common.add_argument("--device", help="cpu или cuda")
    common.add_argument(
        "--nlu",
        default="bank",
        choices=["bank", "laya"],
        help="bank — банк формулировок, laya — только модель",
    )
    common.add_argument(
        "--arbiter",
        default=None,
        choices=["laya"],
        help="кому отдавать спорные реплики",
    )
    common.add_argument(
        "--thin-rescue",
        action="store_true",
        help="отдавать арбитру бедные слоты: банк находит их "
        "в 6-13%% случаев, арбитр от населённости не зависит",
    )
    common.add_argument(
        "--fresh",
        action="store_true",
        help="вычеркнуть формулировки самого сценария — "
        "так он выглядит для системы, которая его не видела",
    )
    common.add_argument("-v", "--verbose", action="store_true")

    ap = argparse.ArgumentParser(prog="dispatcher", description=__doc__)
    sub = ap.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("chat", parents=[common], help="диалог руками")
    p.add_argument("scenario")
    p.set_defaults(func=cmd_chat)

    p = sub.add_parser("replay", parents=[common], help="прогон диалога из файла")
    p.add_argument("scenario")
    p.add_argument("-f", "--file", required=True)
    p.add_argument(
        "--stream", action="store_true", help="через частичные гипотезы, как в звонке"
    )
    p.set_defaults(func=cmd_replay)

    p = sub.add_parser("slots", parents=[common], help="что этот заявитель знает")
    p.add_argument("scenario")
    p.set_defaults(func=cmd_slots)

    p = sub.add_parser(
        "lint", parents=[common], help="дотянется ли корпус до фактов сценария"
    )
    p.add_argument("scenario")
    p.set_defaults(func=cmd_lint)

    p = sub.add_parser(
        "suggest", parents=[common], help="в какой слот положить новый факт"
    )
    p.add_argument("text", help="вопрос или ответ нового факта")
    p.set_defaults(func=cmd_suggest)

    args = ap.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
