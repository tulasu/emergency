#!/usr/bin/env python3
"""
Прослушивание Silero TTS для тренажёра 112.

Что делает:
  1. синтезирует одну и ту же реплику в нескольких вариантах
     (нейтрально / SSML с темпом и высотой / текст с речевыми сбоями);
  2. каждый вариант сохраняет дважды — как есть и после телефонного
     тракта 8 кГц + G.711 A-law;
  3. печатает RTF, чтобы вы сразу видели запас по задержке.

Второй пункт — главный. В наушниках на 48 кГц синтез звучит одним
образом, а обучаемый услышит его после G.711, и часть претензий
к интонации там растворяется сама.

Установка:
    python3 -m venv .venv && source .venv/bin/activate
    pip install torch torchaudio numpy

Запуск:
    python silero_audition.py
    python silero_audition.py --speaker kseniya --text "своя фраза"
"""
import argparse
import os
import time
import wave

import numpy as np
import torch

# ---------- G.711 A-law, векторно ----------

SEG_END = np.array([0x1F, 0x3F, 0x7F, 0xFF, 0x1FF, 0x3FF, 0x7FF, 0xFFF], dtype=np.int32)


def alaw_encode(pcm: np.ndarray) -> np.ndarray:
    """int16 -> uint8 A-law (ITU-T G.711, референсный алгоритм)."""
    x = pcm.astype(np.int32) >> 3          # A-law работает с 13-битной величиной
    neg = x < 0
    mag = np.where(neg, -x - 1, x)
    mask = np.where(neg, 0x55, 0xD5).astype(np.int32)

    seg = np.searchsorted(SEG_END, mag)    # первый сегмент, куда влезает величина
    over = seg >= 8
    seg = np.minimum(seg, 7)

    shift = np.where(seg < 2, 1, seg)
    aval = (seg << 4) | ((mag >> shift) & 0x0F)
    aval = np.where(over, 0x7F, aval)
    return ((aval ^ mask) & 0xFF).astype(np.uint8)


def alaw_decode(a: np.ndarray) -> np.ndarray:
    """uint8 A-law -> int16."""
    v = (a.astype(np.int32) ^ 0x55)
    t = (v & 0x0F) << 4
    seg = (v & 0x70) >> 4
    t = np.where(seg == 0, t + 8, t + 0x108)
    t = np.where(seg >= 2, t << np.maximum(seg - 1, 0), t)
    # бит 0x80 установлен -> ПОЛОЖИТЕЛЬНОЕ значение
    out = np.where((v & 0x80) != 0, t, -t)
    return np.clip(out, -32768, 32767).astype(np.int16)


def write_wav(path: str, pcm: np.ndarray, rate: int):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with wave.open(path, "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(rate)
        w.writeframes(pcm.astype("<i2").tobytes())


# ---------- варианты ----------

PHRASE = "Алло! Тут машина в машину влетела, человек лежит, помогите!"

DISFLUENT = (
    "Алло! Алло, вы слышите? Тут это... машина, машина в машину влетела! "
    "Человек лежит, он не... он не шевелится, вы можете быстрее?"
)


def variants(text: str):
    """(имя, kwargs для apply_tts)"""
    return [
        ("01_neutral", dict(text=text)),
        (
            "02_ssml_fast_high",
            dict(ssml_text=f'<speak><prosody rate="fast" pitch="high">{text}</prosody></speak>'),
        ),
        (
            "03_ssml_broken",
            dict(
                ssml_text=(
                    '<speak><prosody rate="x-fast" pitch="high">Алло!</prosody>'
                    '<break time="300ms"/>'
                    '<prosody rate="fast" pitch="high">Тут машина в машину влетела!</prosody>'
                    '<break time="500ms"/>'
                    '<prosody rate="fast">Человек лежит, помогите!</prosody></speak>'
                )
            ),
        ),
        ("04_disfluent_text", dict(text=DISFLUENT)),
    ]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="v4_ru",
                    help="v4_ru (стабильно, умеет sample_rate=8000) или id модели v5 — "
                         "точный идентификатор смотрите в models.yml репозитория silero-models")
    ap.add_argument("--speaker", default="kseniya",
                    help="для v4_ru: aidar, baya, kseniya, xenia, eugene")
    ap.add_argument("--rate", type=int, default=48000, help="частота синтеза")
    ap.add_argument("--text", default=PHRASE)
    ap.add_argument("--out", default="out")
    args = ap.parse_args()

    torch.set_num_threads(max(1, os.cpu_count() // 2))
    print(f"загружаю {args.model} ...")
    model, _ = torch.hub.load(
        repo_or_dir="snakers4/silero-models",
        model="silero_tts",
        language="ru",
        speaker=args.model,
    )
    model.to(torch.device("cpu"))

    import torchaudio.functional as AF

    for name, kw in variants(args.text):
        t0 = time.perf_counter()
        try:
            audio = model.apply_tts(
                speaker=args.speaker,
                sample_rate=args.rate,
                put_accent=True,
                put_yo=True,
                **kw,
            )
        except Exception as e:
            print(f"  {name}: пропуск — {e}")
            continue
        dt = time.perf_counter() - t0

        wide = audio.numpy()
        dur = len(wide) / args.rate
        print(f"  {name}: {dur:.2f} c, синтез {dt:.2f} c, RTF {dt / dur:.3f}")

        write_wav(f"{args.out}/{name}_{args.rate // 1000}k.wav",
                  np.clip(wide * 32767, -32768, 32767).astype(np.int16), args.rate)

        # телефонный тракт: децимация до 8 кГц + G.711 туда-обратно
        narrow = AF.resample(audio, args.rate, 8000).numpy()
        pcm8 = np.clip(narrow * 32767, -32768, 32767).astype(np.int16)
        through = alaw_decode(alaw_encode(pcm8))
        write_wav(f"{args.out}/{name}_phone8k.wav", through, 8000)

    print(f"\nготово: {args.out}/")
    print("слушайте парами: *_48k.wav и *_phone8k.wav — судить надо по второму")


if __name__ == "__main__":
    main()
