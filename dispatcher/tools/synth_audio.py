# -*- coding: utf-8 -*-
"""Предрендер ответов заявителя в wav 8 кГц mono — фразы уже заготовлены.

Обходит все факты всех сценариев + общие реплики (GENERIC/SLOW_DOWN/URGE),
синтезирует espeak-ng (ru) -> ffmpeg ресемпл 8000 Гц, пишет
data/audio/<id>.wav + data/audio/index.json (audio_id -> путь).
Дедуп по тексту: повторы («Да.») ссылаются на один файл.

  python3 tools/synth_audio.py --limit 20     # проверка пайплайна
  python3 tools/synth_audio.py                # полный прогон (~3.5k файлов)
  python3 tools/synth_audio.py --check        # все id резолвятся в файлы
"""

from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from dispatcher.data.loader import load_all  # noqa: E402
from dispatcher.data.ontology import Ontology  # noqa: E402
from dispatcher.dialog.render import GENERIC, SLOW_DOWN, URGE  # noqa: E402

AUDIO = ROOT / "data" / "audio"
RATE = 8000


def fskey(key: str) -> str:
    return key.replace("#", "_").replace("/", "_")


def collect() -> list[tuple[str, str]]:
    """(audio_id, текст). Порядок стабильный — индекс детерминирован."""
    onto = Ontology.load()
    loaded = load_all(onto)
    items: list[tuple[str, str]] = []
    for sid in sorted(loaded.scenarios):
        sc = loaded.scenarios[sid]
        items.append((f"a/{sid}/opening.wav", sc.opening))
        for key in sorted(sc.facts):
            fact = sc.facts[key]
            for style in sorted(fact.answers):
                items.append((f"a/{sid}/{fskey(key)}/{style}.wav", fact.answers[style]))
    for style in sorted(GENERIC, key=str):
        for mood in sorted(GENERIC[style]):
            for i, text in enumerate(GENERIC[style][mood]):
                items.append((f"common/{style.value}/{mood.name.lower()}_{i}.wav", text))
    for mood in sorted(SLOW_DOWN):
        items.append((f"common/slow_down/{mood.name.lower()}.wav", SLOW_DOWN[mood]))
    urge_slots: set[str] = set()
    for sc in loaded.scenarios.values():
        for k in sc.critical:
            if (f := sc.facts.get(k)) is not None:
                urge_slots.add(f.slot)
    for mood in sorted(URGE):
        for slot in sorted(urge_slots):
            s = onto.slots.get(slot)
            what = (s.urge or s.label) if s else slot
            items.append((f"common/urge/{mood.name.lower()}/{slot}.wav",
                          URGE[mood].format(what=what)))
    return items


def synth(text: str, out: Path, voice: str = "ru") -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    p1 = subprocess.Popen(
        ["espeak-ng", "-v", voice, "-s", "175", "--stdout", text],
        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
    )
    p2 = subprocess.Popen(
        ["ffmpeg", "-y", "-v", "error", "-i", "pipe:0",
         "-ar", str(RATE), "-ac", "1", "-c:a", "pcm_s16le", str(out)],
        stdin=p1.stdout, stdout=subprocess.DEVNULL,
    )
    p1.stdout.close()
    rc = p2.wait()
    p1.wait()
    if rc != 0:
        raise RuntimeError(f"ffmpeg не смог: {text[:40]!r}")


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--limit", type=int, default=0)
    ap.add_argument("--voice", default="ru")
    ap.add_argument("--check", action="store_true")
    args = ap.parse_args()

    items = collect()
    print(f"всего реплик: {len(items)}", flush=True)

    if args.check:
        idx = json.loads((AUDIO / "index.json").read_text(encoding="utf-8"))
        missing = [i for i, _ in items
                   if i not in idx["files"] or not (AUDIO / idx["files"][i]).exists()]
        print(f"нет файлов: {len(missing)}", flush=True)
        for m in missing[:20]:
            print(f"  MISSING {m}")
        sys.exit(1 if missing else 0)

    by_text: dict[str, str] = {}  # norm -> audio_id первого вхождения
    files: dict[str, str] = {}
    if (AUDIO / "index.json").exists():
        files = json.loads((AUDIO / "index.json").read_text(encoding="utf-8"))["files"]
    done = 0
    for aid, text in items:
        if args.limit and done >= args.limit:
            break
        norm = hashlib.md5(" ".join(text.lower().split()).encode()).hexdigest()
        if norm in by_text:
            files[aid] = by_text[norm]
            continue
        if aid in files and (AUDIO / files[aid]).exists():
            by_text[norm] = files[aid]
            continue
        synth(text, AUDIO / aid, args.voice)
        files[aid] = aid
        by_text[norm] = aid
        done += 1
        if done % 100 == 0:
            print(f"  {done}...", flush=True)
    AUDIO.mkdir(parents=True, exist_ok=True)
    (AUDIO / "index.json").write_text(
        json.dumps({"version": 1, "files": files}, ensure_ascii=False, indent=1),
        encoding="utf-8",
    )
    print(f"готово: синтезировано {done}, всего в индексе {len(files)}", flush=True)


if __name__ == "__main__":
    main()
