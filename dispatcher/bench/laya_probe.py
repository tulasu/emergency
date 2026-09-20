# -*- coding: utf-8 -*-
"""
Проба laya как арбитра серой зоны.

Laya решает типизированный вопрос за один проход: ни генерации, ни парсинга,
ни срывов формата — то, на чём развалился Qwen. Здесь проверяется, годится ли
она по существу: выбрать нужный слот из пяти и честно сказать «ни о чём»,
когда вопрос не про них.

Позиции вариантов перемешиваются. Без этого замер врёт: модель, отвечающая
всегда первым вариантом, покажет мнимые 60% точности.

  HF_HOME=$PWD/models python3 -m bench.laya_probe
  HF_HOME=$PWD/models python3 -m bench.laya_probe --device cuda
"""

from __future__ import annotations

import argparse
import random
import statistics
import time

from dispatcher.data.loader import load_all
from dispatcher.data.ontology import Ontology

NOTHING = "ни о чём из перечисленного"

CASES = [
    (
        "на каком этаже полыхает",
        ["fire.floor", "fire.what", "building.floors", "addr.street", "caller.name"],
        "fire.floor",
    ),
    (
        "а дом-то весь какой высоты",
        ["fire.floor", "building.floors", "addr.building", "fire.what", "caller.name"],
        "building.floors",
    ),
    (
        "как к вам обращаться",
        ["caller.name", "caller.phone", "victim.present", "addr.street", "fire.what"],
        "caller.name",
    ),
    (
        "какая марка машины",
        ["caller.name", "fire.what", "addr.street", "victim.present", "fire.floor"],
        None,
    ),
    (
        "группа крови какая",
        ["victim.present", "victim.name", "caller.name", "fire.what", "addr.street"],
        None,
    ),
    (
        "телефон для связи продиктуйте",
        ["caller.phone", "caller.name", "addr.street", "fire.what", "victim.present"],
        "caller.phone",
    ),
    (
        "а что в доме поблизости расположено",
        ["addr.landmark", "addr.street", "addr.building", "fire.what", "caller.name"],
        "addr.landmark",
    ),
    (
        "сатурацию мерили",
        [
            "victim.present",
            "med.consciousness",
            "caller.name",
            "fire.what",
            "addr.street",
        ],
        None,
    ),
    (
        "у вас есть страховка",
        ["caller.name", "caller.phone", "addr.street", "fire.what", "victim.present"],
        None,
    ),
    (
        "пламя из окон бьёт",
        ["fire.flame", "fire.what", "fire.floor", "victim.present", "addr.street"],
        "fire.flame",
    ),
    (
        "людей в квартире не осталось",
        [
            "victim.present",
            "fire.what",
            "caller.location",
            "addr.street",
            "caller.name",
        ],
        "victim.present",
    ),
    (
        "вы сами откуда смотрите",
        [
            "caller.location",
            "addr.street",
            "caller.name",
            "fire.what",
            "victim.present",
        ],
        "caller.location",
    ),
]


def describe(onto, slot: str) -> str:
    """Чем слот отличается от соседей — это и есть критерий выбора."""
    sl = onto.slots[slot]
    examples = ", ".join(sl.questions[:2]) if sl.questions else ""
    return f"{sl .label }{'; '+examples if examples else ''}"


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--device", default="cpu")
    args = ap.parse_args()

    from laya import Router

    onto = Ontology.load()
    load_all(onto)

    t0 = time.perf_counter()
    # нужен только мультиязычный чекпоинт: вопросы оператора кириллицей,
    # английский на них по их же замерам даёт ноль при высокой уверенности
    router = Router(device=args.device, default="multilingual")
    print(f"загрузка {time .perf_counter ()-t0 :.1f} с")

    correct = 0
    refusals_ok = refusals_total = 0
    lat: list[float] = []

    for i, (text, slots, want) in enumerate(CASES):
        r = random.Random(i)
        s = list(slots)
        r.shuffle(s)

        criteria = {onto.slots[x].label: describe(onto, x) for x in s}
        criteria[NOTHING] = "вопрос не про перечисленное"
        question = {
            "слот": {
                "type": "choice",
                "instructions": "О чём спрашивает оператор службы 112?",
                "criteria": criteria,
            }
        }

        if i == 0:  # первый проход строит модель
            router.predict({"вопрос оператора": text}, question)
        t = time.perf_counter()
        res = router.predict({"вопрос оператора": text}, question)
        lat.append(1000 * (time.perf_counter() - t))

        answer = res["answers"]["слот"]
        pick = answer.get("choice")
        conf = answer.get("confidence", 0.0)
        expected = onto.slots[want].label if want else NOTHING
        ok = pick == expected
        correct += ok
        if want is None:
            refusals_total += 1
            refusals_ok += ok
        print(
            f"  {text :<36} -> {str (pick )[:26 ]:<26} {conf :.2f}  "
            f"{'ok'if ok else 'ждали '+expected }"
        )

    print(
        f"\nверно {correct }/{len (CASES )}, из них отказов {refusals_ok }/{refusals_total }"
    )
    print(
        f"задержка: медиана {statistics .median (lat ):.0f} мс, "
        f"p95 {sorted (lat )[int (0.95 *(len (lat )-1 ))]:.0f} мс"
    )
    print(f"маршрут: {res ['routing']['model']} — {res ['routing']['reason']}")


if __name__ == "__main__":
    main()
