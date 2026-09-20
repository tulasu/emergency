# -*- coding: utf-8 -*-
"""
Эталонная разметка слепых наборов сильной моделью.

Всё, что меряется сейчас, опирается на формулировки корпуса: эталон известен,
потому что формулировка принадлежит факту. У слепых наборов такого эталона
нет, поэтому `bench/run.py` умеет считать исходы, но не может отличить верный
факт от неверного.

Здесь эти 6105 реплик размечаются один раз облачной моделью. Дальше все
эксперименты меряются по `data/gold/` бесплатно.

Модель работает классификатором: на вход каталог слотов сценария, на выход
список слотов. Пустой список значит «заявитель этого знать не может» — это
такой же правильный ответ, как любой другой, и размечать его надо наравне.

  ANTHROPIC_API_KEY=... python3 -m bench.label            всё
  ANTHROPIC_API_KEY=... python3 -m bench.label -n 5       пять сценариев
  python3 -m bench.label --dry                            смотр промпта без вызовов
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

from dispatcher.data.loader import DATA, load_all
from dispatcher.data.ontology import Ontology

GOLD = DATA / "gold"
URL = "https://api.anthropic.com/v1/messages"
MODEL = "claude-sonnet-5"
BATCH = 20

SYSTEM = """Ты размечаешь реплики оператора службы 112 для учебного тренажёра.

Заявитель в этом вызове знает ровно перечисленные сведения и ничего сверх:

{каталог}

Для каждой реплики оператора определи, о каких из этих сведений он спрашивает.

Верни ТОЛЬКО JSON вида {{"1": ["ключ", ...], "2": [], ...}}, где ключ строки —
номер реплики.

Правила:
- пустой список, если оператор спрашивает о том, чего в списке нет: заявитель
  имеет право не знать, и это правильный ответ, а не ошибка разметки;
- пустой список, если реплика вообще не вопрос (успокаивает, сообщает о
  выезде, инструктирует);
- несколько ключей, если в одной реплике несколько вопросов;
- ключей, которых нет в списке, не выдумывай."""


def catalog(sc, onto: Ontology) -> str:
    rows = []
    for slot, keys in sorted(sc.by_slot.items()):
        label = onto.slots[slot].label if slot in onto.slots else slot
        answer = sc.facts[keys[0]].answers["plain"]
        rows.append(f"- {slot } ({label }): {answer }")
    return "\n".join(rows)


def ask_user(
    system: str, utterances: list[str], key: str, attempts: int = 4
) -> dict[str, list[str]] | None:
    numbered = "\n".join(f"{i +1 }. {t }" for i, t in enumerate(utterances))
    body = json.dumps(
        {
            "model": MODEL,
            "max_tokens": 2000,
            "temperature": 0,
            "system": system,
            "messages": [{"role": "user", "content": numbered}],
        }
    ).encode()

    for attempt in range(attempts):
        req = urllib.request.Request(
            URL,
            data=body,
            headers={
                "content-type": "application/json",
                "anthropic-version": "2023-06-01",
                "x-api-key": key,
            },
        )
        try:
            with urllib.request.urlopen(req, timeout=120) as r:
                data = json.loads(r.read())
            raw = "".join(c.get("text", "") for c in data.get("content", []))
            return json.loads(re.sub(r"```json|```", "", raw).strip())
        except (
            urllib.error.URLError,
            TimeoutError,
            OSError,
            json.JSONDecodeError,
            KeyError,
        ):
            if attempt == attempts - 1:
                return None
            time.sleep(2**attempt)
    return None


def utterances(sid: str) -> list[str]:
    from bench.run import phrases

    return phrases(sid)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("-n", type=int, default=None, help="сколько сценариев")
    ap.add_argument("--dry", action="store_true", help="показать промпт, не вызывать")
    args = ap.parse_args()

    onto = Ontology.load()
    loaded = load_all(onto)
    key = os.environ.get("ANTHROPIC_API_KEY")

    if args.dry:
        sc = loaded.scenarios["bilet04_call01"]
        print(SYSTEM.format(catalog=catalog(sc, onto)))
        print("\n--- пример реплик ---")
        for i, t in enumerate(utterances(sc.id)[:5]):
            print(f"{i +1 }. {t }")
        return 0

    if not key:
        print("нужен ANTHROPIC_API_KEY", file=sys.stderr)
        return 1

    GOLD.mkdir(parents=True, exist_ok=True)
    n_calls = labeled = skipped = 0

    for sid in sorted(loaded.scenarios)[: args.n]:
        path = GOLD / f"{sid }.json"
        if path.exists():
            print(f"{sid }: уже размечен")
            continue
        sc = loaded.scenarios[sid]
        rows = utterances(sid)
        if not rows:
            continue

        system = SYSTEM.format(catalog=catalog(sc, onto))
        own = set(sc.by_slot) | set(sc.answers_for)
        labels: dict[str, list[str]] = {}

        for start in range(0, len(rows), BATCH):
            batch = rows[start : start + BATCH]
            answer = ask_user(system, batch, key)
            n_calls += 1
            if answer is None:
                skipped += len(batch)
                continue
            for i, text in enumerate(batch):
                slots = answer.get(str(i + 1), [])
                labels[text] = [s for s in slots if s in own]

        path.write_text(
            json.dumps(labels, ensure_ascii=False, indent=1), encoding="utf-8"
        )
        labeled += len(labels)
        print(f"{sid }: {len (labels )} реплик")

    print(
        f"\nразмечено {labeled } реплик за {n_calls } вызовов, " f"не вышло {skipped }"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
