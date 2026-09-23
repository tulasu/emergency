# -*- coding: utf-8 -*-
"""
Прогон размеченных реплик с живых звонков (data/gold/live_calls.tsv).

В отличие от слепых наборов, здесь есть разметка — какой слот оператор
спрашивал, — и текст такой, каким его отдаёт STT: без пунктуации, числа
словами, с «алло, служба сто двенадцать» в начале. Считается понимание,
а не факт: понятый слот, которого у заявителя нет («не знаю»), — верно.

  python3 -m bench.live                      лексика
  python3 -m bench.live -m e5-small:onnx     со сплавом
  python3 -m bench.live --show               все ошибки построчно
  python3 -m bench.live --arbiter laya [--thin-rescue]   laya в серой зоне
  python3 -m bench.live --nlu laya           понимание целиком на laya
"""

from __future__ import annotations

import argparse
import time
from collections import Counter

from dispatcher.data.loader import DATA
from dispatcher.service import Service
from dispatcher.types import Act

GOLD = DATA / "gold" / "live_calls.tsv"


def rows() -> list[tuple[str, str, list[str]]]:
    out = []
    for line in GOLD.read_text(encoding="utf-8").splitlines():
        if not line.strip() or line.startswith("#"):
            continue
        sid, text, want = line.split("\t")
        out.append((sid, text, want.split("|")))
    return out


def hit(want: list[str], u) -> bool:
    if want == ["ack"]:
        return u.act == Act.SPEECH_ACT or (not u.slots and u.source == "rules")
    got = set(u.slots) | {k.split("#")[0] for k in u.keys}
    for w in want:
        if w.endswith("*"):
            if any(g.startswith(w[:-1]) for g in got):
                return True
        elif w in got:
            return True
    return False


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("-m", "--model", default=None, help="модель:бэкенд, e5-small:onnx")
    ap.add_argument("--show", action="store_true")
    ap.add_argument("--nlu", default="bank", choices=["bank", "laya"])
    ap.add_argument("--arbiter", default=None, choices=["laya"])
    ap.add_argument("--thin-rescue", action="store_true")
    ap.add_argument("--device", default="cpu")
    ap.add_argument("--ensemble", default=None,
                    choices=["majority", "strict", "cascade+", "llm-lead"],
                    help="голосование каскад + laya + LLM (llama-server)")
    ap.add_argument("--voters", default="laya,llm")
    ap.add_argument("--laya-min", type=float, default=0.5)
    args = ap.parse_args()

    name, _, backend = (args.model or "").partition(":")
    backend = backend or "auto"
    if backend.startswith("torch"):
        backend = "torch"
    svc = Service(model=name or None, backend=backend, device=args.device,
                  arbiter_kind=args.arbiter if args.nlu == "bank" else None,
                  thin_rescue=args.thin_rescue)
    router = None
    if args.nlu == "laya":
        from dispatcher.nlu.laya import make_router
        from dispatcher.nlu.laya_nlu import LayaUnderstander
        router = make_router(args.device)
    voters = []
    if args.ensemble:
        from dispatcher.nlu.ensemble import Ensemble
        for v in args.voters.split(","):
            if v == "laya":
                from dispatcher.nlu.laya_arbiter import LayaArbiter
                voters.append(LayaArbiter(ontology=svc.onto, device=args.device,
                                          min_confidence=args.laya_min))
            elif v == "llm":
                from dispatcher.nlu.llm_arbiter import LlmArbiter
                voters.append(LlmArbiter(svc.onto))
    ok, miss = 0, Counter()
    data = rows()
    t0 = time.perf_counter()
    for sid, text, want in data:
        if router is not None:
            u = LayaUnderstander(scenario=svc.loaded.scenarios[sid],
                                 ontology=svc.onto, router=router).understand(text)
        else:
            s_id, _ = svc.open(sid)
            nlu = svc.sessions[s_id].cascade
            if voters:
                nlu = Ensemble(nlu, voters, rule=args.ensemble)
            u = nlu.understand(text)
            svc.close(s_id)
        if hit(want, u):
            ok += 1
            continue
        miss["ack" if want == ["ack"] else want[0].split(".")[0]] += 1
        if args.show:
            got = ",".join(u.slots) or f"[{u.act}]"
            print(f"  MISS {sid:15} {text[:60]:60} | ждали {'|'.join(want)[:30]:30} | {got} ({u.source})")
    ms = (time.perf_counter() - t0) / len(data) * 1000
    label = " + ".join(filter(None, [
        args.model or ("laya-nlu" if args.nlu == "laya" else "лексика"),
        args.arbiter and f"арбитр {args.arbiter}",
        args.thin_rescue and "thin_rescue",
        args.ensemble and f"голосование {args.ensemble} [{args.voters}]"]))
    print(f"{label}: верно {ok}/{len(data)} = {ok / len(data):.1%}, {ms:.0f} мс/реплика")
    print("  промахи по группам:", dict(miss.most_common()))


if __name__ == "__main__":
    main()
