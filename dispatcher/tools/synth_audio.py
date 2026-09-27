# -*- coding: utf-8 -*-
"""Предрендер ответов заявителя в wav 8 кГц mono — фразы уже заготовлены.

Обходит все факты всех сценариев + общие реплики (GENERIC/SLOW_DOWN/URGE),
синтезирует Silero v4_ru (по умолчанию) или espeak-ng -> 8000 Гц, пишет
data/audio/<id>.wav + data/audio/index.json (audio_id -> путь).
Дедуп по тексту: повторы («Да.») ссылаются на один файл.

  python3 tools/synth_audio.py --limit 20     # проверка пайплайна
  python3 tools/synth_audio.py                # полный прогон (~3.5k файлов)
  python3 tools/synth_audio.py --check        # все id резолвятся в файлы
  python3 tools/synth_audio.py --force        # перезаписать уже готовые

Silero нужен torch+torchaudio (CPU хватает, RTF ~0.05): удобно гонять
из ../test-tts/.venv. Фразы, на которых Silero падает (латиница, цифры),
уходят в espeak-ng.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
import wave
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


_ONES = ("ноль один два три четыре пять шесть семь восемь девять десять "
         "одиннадцать двенадцать тринадцать четырнадцать пятнадцать "
         "шестнадцать семнадцать восемнадцать девятнадцать").split()
_TENS = "_ _ двадцать тридцать сорок пятьдесят шестьдесят семьдесят восемьдесят девяносто".split()
_HUNDREDS = ("_ сто двести триста четыреста пятьсот шестьсот семьсот "
             "восемьсот девятьсот").split()


def _say999(n: int) -> str:
    """0..999 словами (м. р., им. п.)."""
    if n < 20:
        return _ONES[n]
    words = []
    if n >= 100:
        words.append(_HUNDREDS[n // 100])
        n %= 100
    if n >= 20:
        words.append(_TENS[n // 10])
        n %= 10
    if n:
        words.append(_ONES[n])
    return " ".join(words)


def _say_digits(m: re.Match) -> str:
    d = m.group(0)
    if len(d) >= 5:  # телефон/код: как диктуют — 3-3-2-2, ведущие нули цифрами
        groups, rest = [], d
        while len(rest) > 4:
            groups.append(rest[:3])
            rest = rest[3:]
        groups += [rest[:2], rest[2:]] if len(rest) == 4 else [rest]
    elif len(d) == 4 and d[0] == "0":  # код «0453» — парами
        groups = [d[:2], d[2:]]
    else:
        groups = [d]
    out = []
    for g in groups:
        lead = len(g) - len(g.lstrip("0"))
        out += ["ноль"] * min(lead, len(g) - 1)
        n = int(g)
        out.append(_say999(n) if n < 1000 else _say_thousands(n))
    return ", ".join(out)


def _say_thousands(n: int) -> str:
    k, r = divmod(n, 1000)
    head = {1: "одна", 2: "две"}.get(k % 10) if k % 100 not in (11, 12) else None
    kw = _say999(k)
    if k == 1:
        return "тысяча" + (f" {_say999(r)}" if r else "")
    if head:
        kw = kw.rsplit(" ", 1)[0] + " " + head if " " in kw else head
    if k % 10 == 1 and k % 100 != 11:
        form = "тысяча"
    elif k % 10 in (2, 3, 4) and k % 100 not in (12, 13, 14):
        form = "тысячи"
    else:
        form = "тысяч"
    return f"{kw} {form}" + (f" {_say999(r)}" if r else "")


def speakable(text: str) -> str:
    """Silero не читает цифры — переводим в слова."""
    # «916 896 3254», «903-226-13-83» — один номер, склеиваем перед группировкой
    text = re.sub(r"\d[\d \-]{5,}\d",
                  lambda m: re.sub(r"[ \-]", "", m.group(0))
                  if sum(c.isdigit() for c in m.group(0)) >= 7 else m.group(0),
                  text)
    return re.sub(r"\d+", _say_digits, text)


_SILERO = None
_SILERO_RATE = 48000


def synth_silero(text: str, out: Path, speaker: str) -> None:
    global _SILERO
    import numpy as np
    import torch
    import torchaudio.functional as AF

    if _SILERO is None:
        _SILERO, _ = torch.hub.load(
            repo_or_dir="snakers4/silero-models", model="silero_tts",
            language="ru", speaker="v4_ru",
        )
        _SILERO.to(torch.device("cpu"))
    audio = _SILERO.apply_tts(text=speakable(text), speaker=speaker,
                              sample_rate=_SILERO_RATE,
                              put_accent=True, put_yo=True)
    # 48k -> 8k: синтез на 8k звучит глуше, децимация чище
    narrow = AF.resample(audio, _SILERO_RATE, RATE).numpy()
    pcm = np.clip(narrow * 32767, -32768, 32767).astype("<i2")
    out.parent.mkdir(parents=True, exist_ok=True)
    with wave.open(str(out), "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(RATE)
        w.writeframes(pcm.tobytes())


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
    ap.add_argument("--engine", default="silero", choices=["silero", "espeak"])
    ap.add_argument("--speaker", default="kseniya",
                    help="silero v4_ru: aidar, baya, kseniya, xenia, eugene")
    ap.add_argument("--force", action="store_true",
                    help="пересинтезировать и уже существующие файлы")
    ap.add_argument("--grep", default="",
                    help="только реплики, чей текст матчит regex (с --force)")
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
        if args.grep and not re.search(args.grep, text):
            continue
        norm = hashlib.md5(" ".join(text.lower().split()).encode()).hexdigest()
        if norm in by_text:
            files[aid] = by_text[norm]
            continue
        if not args.force and aid in files and (AUDIO / files[aid]).exists():
            by_text[norm] = files[aid]
            continue
        if args.engine == "silero":
            try:
                synth_silero(text, AUDIO / aid, args.speaker)
            except Exception as e:  # noqa: BLE001 — silero рвётся на экзотике
                print(f"  espeak-фолбэк {aid}: {e}", flush=True)
                synth(text, AUDIO / aid, args.voice)
        else:
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
