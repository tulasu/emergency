# -*- coding: utf-8 -*-
"""Переезд сценариев на канонический формат: одни английские идентификаторы.

  python3 tools/migrate_scenario.py                 # все файлы data/scenarios
  python3 tools/migrate_scenario.py path/to/x.json  # один файл

Старый факт {key, group, value_numbers, ...} становится
{id, slot, numbers, ...}, где id — это слот, а если слот делят несколько
фактов, первый забирает голый слот, остальные получают суффикс #2, #3.
Русский остаётся только в текстах для людей: questions, answers, opening,
значения meta. Файл перезаписывается на месте.
"""

from __future__ import annotations

import json
import sys
from collections import Counter
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from dispatcher.data.ontology import Ontology

ROOT = Path(__file__).resolve().parents[1] / "data" / "scenarios"

META = {
    "ситуация": "situation",
    "адрес_эталон": "address_ref",
    "итоговый_тип": "incident_type",
    "источник": "source",
}


def migrate(path: Path, onto: Ontology) -> str:
    raw = json.loads(path.read_text(encoding="utf-8"))
    sid = raw["id"]

    slots: list[str] = []
    for item in raw["facts"]:
        key = item.get("id", item.get("key"))
        slot = item.get("slot") or onto.resolve(key, item.get("group") or "", sid)
        if slot is None:
            raise SystemExit(f"{sid}:{key}: слот не выводится, допишите slot вручную")
        slots.append(slot)

    counts = Counter(slots)
    seen: Counter[str] = Counter()
    idmap: dict[str, str] = {}
    for item, slot in zip(raw["facts"], slots):
        key = item.get("id", item.get("key"))
        if counts[slot] == 1:
            idmap[key] = slot
        else:
            seen[slot] += 1
            idmap[key] = slot if seen[slot] == 1 else f"{slot}#{seen[slot]}"

    strange = [k for k in raw.get("meta", {}) if k not in META]
    if strange:
        raise SystemExit(f"{sid}: неизвестные ключи meta: {strange}")

    facts = []
    for item, slot in zip(raw["facts"], slots):
        f: dict = {"id": idmap[item.get("id", item.get("key"))], "slot": slot}
        if item.get("numbers", item.get("value_numbers", ())):
            f["numbers"] = list(item.get("numbers", item.get("value_numbers")))
        if item.get("questions"):
            f["questions"] = list(item["questions"])
        f["answers"] = dict(item["answers"])
        if "disclosure" in item:
            f["disclosure"] = item["disclosure"]
        if item.get("requires"):
            f["requires"] = [idmap[r] for r in item["requires"]]
        facts.append(f)

    persona = raw.get("persona", {})
    out = {
        "id": sid,
        "meta": {META[k]: v for k, v in raw.get("meta", {}).items()},
        "persona": {"profile": persona.get("profile", "calm"),
                    "opening": persona.get("opening", "Алло! Помогите!")},
        "critical": [idmap[k] for k in raw.get("critical", ()) if k in idmap],
        "facts": facts,
    }
    path.write_text(json.dumps(out, ensure_ascii=False, indent=1) + "\n",
                     encoding="utf-8")
    return sid


def main(argv: list[str]) -> None:
    onto = Ontology.load()
    paths = [Path(a) for a in argv[1:]] or sorted(ROOT.glob("*.json"))
    for p in paths:
        print(migrate(p, onto))


if __name__ == "__main__":
    main(sys.argv)
