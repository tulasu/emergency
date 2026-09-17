# -*- coding: utf-8 -*-
"""
Сравнение backends на gold JSONL: lexical / semantic / local_llm (GGUF).

  python -m bench --all
  python -m bench --all --backends lexical,semantic,local_llm
"""
from __future__ import annotations

import argparse
import json
import statistics
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path

ROOT = Path(__file__).resolve().parent
TRAINER = ROOT.parent / "trainer"
if str(TRAINER) not in sys.path:
    sys.path.insert(0, str(TRAINER))
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from caller.scenario import Scenario  # noqa: E402
from caller.understand import Lexical  # noqa: E402

from embedder import DEFAULT_MODEL, Embedder  # noqa: E402
from understand import SemanticUnderstander  # noqa: E402

DEFAULT_SETS = [
    ("gold/bilet04_call01.jsonl", "../trainer/scenarios/bilet04_call01.json"),
    ("gold/bilet01_call01.jsonl", "../trainer/scenarios/bilet01_call01.json"),
    ("gold/bilet02_call01.jsonl", "../trainer/scenarios/bilet02_call01.json"),
]
ALL_BACKENDS = ("lexical", "semantic", "local_llm")


@dataclass
class Row:
    text: str
    expect: list[str]
    note: str = ""


@dataclass
class BackendStats:
    name: str
    exact: int = 0
    top1: int = 0
    false_accept: int = 0
    reject_ok: int = 0
    n: int = 0
    latencies: list[float] = field(default_factory=list)
    disagreements: list[str] = field(default_factory=list)
    skipped: bool = False
    skip_reason: str = ""

    def add(self, expect: list[str], got: list[str], ms: float, text: str,
            note: str) -> None:
        self.n += 1
        self.latencies.append(ms)
        exp = list(expect)
        if got == exp:
            self.exact += 1
        if (not exp and not got) or (exp and got and got[0] == exp[0]):
            self.top1 += 1
        if not exp and got:
            self.false_accept += 1
        if not exp and not got:
            self.reject_ok += 1
        if got != exp:
            self.disagreements.append(
                f"  [{note}] {text!r}\n    expect={exp} got={got}"
            )

    def pct(self, x: int) -> str:
        return f"{100 * x / self.n:.0f}%" if self.n else "—"

    def lat_summary(self) -> str:
        if not self.latencies:
            return "—"
        xs = sorted(self.latencies)
        p50 = statistics.median(xs)
        p95 = xs[min(len(xs) - 1, int(0.95 * len(xs)))]
        return (f"p50={p50:.1f}ms p95={p95:.1f}ms "
                f"avg={statistics.mean(xs):.1f}ms max={max(xs):.1f}ms")


def load_gold(path: Path) -> list[Row]:
    rows = []
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        o = json.loads(line)
        rows.append(Row(o["text"], list(o.get("expect", [])), o.get("note", "")))
    return rows


def resolve(p: str) -> Path:
    path = Path(p)
    if not path.is_absolute():
        path = (ROOT / path).resolve()
    return path


def parse_backends(raw: str) -> list[str]:
    parts = [x.strip() for x in raw.split(",") if x.strip()]
    bad = [p for p in parts if p not in ALL_BACKENDS]
    if bad:
        sys.exit(f"неизвестные backends: {bad}; допустимо: {','.join(ALL_BACKENDS)}")
    return parts or list(ALL_BACKENDS)


def print_stats(st: BackendStats) -> None:
    if st.skipped:
        print(f"\n[{st.name}] SKIPPED: {st.skip_reason}")
        return
    print(f"\n[{st.name}] n={st.n}")
    print(f"  exact key-set   {st.exact:3}  ({st.pct(st.exact)})")
    print(f"  top-1           {st.top1:3}  ({st.pct(st.top1)})")
    print(f"  false accept    {st.false_accept:3}  ({st.pct(st.false_accept)})")
    print(f"  reject ok       {st.reject_ok:3}  ({st.pct(st.reject_ok)})")
    print(f"  latency         {st.lat_summary()}")


def run_one(
    gold_path: Path,
    scenario_path: Path,
    backends: list[str],
    embedder: Embedder | None,
    llm_factory,
    threshold: float,
    margin: float,
) -> dict:
    sc = Scenario.load(scenario_path)
    gold = load_gold(gold_path)
    stats: dict[str, BackendStats] = {b: BackendStats(b) for b in backends}

    lex = Lexical(sc) if "lexical" in backends else None
    sem = None
    build_ms = 0.0
    if "semantic" in backends:
        assert embedder is not None
        sem = SemanticUnderstander(
            sc, embedder=embedder, threshold=threshold, margin=margin,
        )
        build_ms = sem.build()
        _ = sem.match("warmup")

    llm = None
    if "local_llm" in backends:
        if llm_factory is None:
            stats["local_llm"].skipped = True
            stats["local_llm"].skip_reason = "not loaded"
        else:
            llm = llm_factory(sc)
            try:
                llm.load()
            except Exception as e:
                stats["local_llm"].skipped = True
                stats["local_llm"].skip_reason = str(e)
                llm = None

    negation_rows = []
    keysets: dict[str, list[str]] = {}

    for row in gold:
        got: dict[str, list[str]] = {}

        if lex is not None:
            t0 = time.perf_counter()
            u = lex.understand(row.text)
            ms = (time.perf_counter() - t0) * 1000
            got["lexical"] = list(u.keys)
            stats["lexical"].add(row.expect, got["lexical"], ms, row.text, row.note)

        if sem is not None:
            m = sem.match(row.text)
            got["semantic"] = list(m.keys)
            stats["semantic"].add(
                row.expect, got["semantic"], m.total_ms, row.text, row.note,
            )
            if "negation" in row.note:
                negation_rows.append(
                    f"  {row.text!r}\n"
                    f"    expect={row.expect} "
                    + " ".join(f"{k}={got.get(k)}" for k in backends if k in got)
                    + f" sem_score={m.score:.3f}"
                )

        if llm is not None and not stats["local_llm"].skipped:
            keys, ms, status = llm.classify(row.text)
            got["local_llm"] = list(keys)
            stats["local_llm"].add(
                row.expect, got["local_llm"], ms, row.text, row.note,
            )
            if status == "error" and llm.last_error:
                # keep going; one error shouldn't kill the run
                pass

        for name, keys in got.items():
            keysets.setdefault(name, [])
        # agreement lexical <-> semantic if both present
        if "lexical" in got and "semantic" in got:
            keysets.setdefault("_agree_ls", [])
            if got["lexical"] == got["semantic"]:
                keysets["_agree_ls"].append(1)
            else:
                keysets["_agree_ls"].append(0)

    n = len(gold)
    agree_ls = keysets.get("_agree_ls", [])
    return {
        "id": sc.id,
        "n": n,
        "stats": stats,
        "agree_ls": sum(agree_ls),
        "agree_ls_pct": 100 * sum(agree_ls) / len(agree_ls) if agree_ls else 0,
        "negation_rows": negation_rows,
        "build_ms": build_ms,
    }


def main() -> None:
    ap = argparse.ArgumentParser(
        description="lexical / semantic / local_llm NLU bench",
    )
    ap.add_argument("--gold", help="путь к gold JSONL")
    ap.add_argument("--scenario", help="путь к scenario JSON")
    ap.add_argument("--all", action="store_true", help="прогнать DEFAULT_SETS")
    ap.add_argument(
        "--backends",
        default="lexical,semantic,local_llm",
        help="comma-list: lexical,semantic,local_llm",
    )
    ap.add_argument("--model", default=DEFAULT_MODEL)
    ap.add_argument("--threshold", type=float, default=0.45)
    ap.add_argument("--margin", type=float, default=0.03)
    ap.add_argument("--llm-gguf", default=None,
                    help="локальный путь к .gguf (иначе HF download)")
    ap.add_argument("--show-miss", action="store_true", default=True)
    a = ap.parse_args()
    backends = parse_backends(a.backends)

    pairs: list[tuple[Path, Path]] = []
    if a.all or (not a.gold and not a.scenario):
        for g, s in DEFAULT_SETS:
            pairs.append((resolve(g), resolve(s)))
    else:
        if not a.gold or not a.scenario:
            sys.exit("нужны --gold и --scenario, либо --all")
        pairs.append((resolve(a.gold), resolve(a.scenario)))

    print(f"backends={','.join(backends)}")
    embedder = None
    if "semantic" in backends:
        print(f"sts_model={a.model} device=cpu "
              f"threshold={a.threshold} margin={a.margin}")
        embedder = Embedder(a.model, device="cpu")
        t0 = time.perf_counter()
        load_ms = embedder.load()
        sample = ["На каком этаже пожар?"] * 8
        t1 = time.perf_counter()
        embedder.encode(sample)
        encode_batch_ms = (time.perf_counter() - t1) * 1000
        print(f"sts cold load: {load_ms:.0f}ms  "
              f"(wall {(time.perf_counter() - t0) * 1000:.0f}ms)")
        print(f"encode batchx{len(sample)}: {encode_batch_ms:.1f}ms "
              f"({encode_batch_ms / len(sample):.1f}ms/utt)")

    llm_factory = None
    if "local_llm" in backends:
        from local_llm import GgufClassifier, ensure_model, llama_available

        if not llama_available():
            print("local_llm: SKIPPED "
                  "(pip install -r requirements-llm.txt)")
            # keep in backends but every run will skip via factory returning broken
            def llm_factory(sc):  # noqa: F811
                c = GgufClassifier(sc)
                return c
        else:
            try:
                gguf = ensure_model(local_path=a.llm_gguf)
                print(f"gguf={gguf}")
                # probe load once, reuse path
                probe = GgufClassifier(
                    Scenario.load(pairs[0][1]), gguf_path=gguf,
                )
                ms = probe.load()
                print(f"llm cold load: {ms:.0f}ms")

                def llm_factory(sc, _path=gguf):  # noqa: F811
                    return GgufClassifier(sc, gguf_path=_path)
            except Exception as e:
                print(f"local_llm: SKIPPED ({e})")

                def llm_factory(sc):  # noqa: F811
                    c = GgufClassifier(sc)
                    # force fail on load
                    return c

    totals: dict[str, list[int]] = {b: [0, 0] for b in backends}  # exact, n
    totals_agree = 0
    totals_n_agree = 0

    for gold_path, sc_path in pairs:
        if not gold_path.exists():
            print(f"\n!! нет gold: {gold_path}")
            continue
        if not sc_path.exists():
            print(f"\n!! нет scenario: {sc_path}")
            continue
        r = run_one(
            gold_path, sc_path, backends, embedder, llm_factory,
            a.threshold, a.margin,
        )
        print(f"\n{'=' * 60}")
        print(f"scenario={r['id']}  gold={gold_path.name}  n={r['n']}")
        if "semantic" in r["stats"] and not r["stats"]["semantic"].skipped:
            print(f"prototype build: {r.get('build_ms', 0):.0f}ms")
        for b in backends:
            print_stats(r["stats"][b])
            st = r["stats"][b]
            if not st.skipped:
                totals[b][0] += st.exact
                totals[b][1] += st.n
                if a.show_miss and st.disagreements:
                    print(f"\n{b} misses:")
                    print("\n".join(st.disagreements[:10]))
        if "lexical" in backends and "semantic" in backends:
            print(f"\nagreement lexical<->semantic: "
                  f"{r['agree_ls']}/{r['n']} ({r['agree_ls_pct']:.0f}%)")
            totals_agree += r["agree_ls"]
            totals_n_agree += r["n"]
        if r["negation_rows"]:
            print("\nnegation cases:")
            print("\n".join(r["negation_rows"]))

    if len(pairs) > 1:
        print(f"\n{'=' * 60}")
        print("TOTAL")
        for b in backends:
            ex, n = totals[b]
            if n:
                print(f"  {b:12} exact {ex}/{n} ({100 * ex / n:.0f}%)")
            else:
                print(f"  {b:12} skipped / n=0")
        if totals_n_agree:
            print(f"  agreement ls {totals_agree}/{totals_n_agree} "
                  f"({100 * totals_agree / totals_n_agree:.0f}%)")


if __name__ == "__main__":
    main()
