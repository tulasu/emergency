# -*- coding: utf-8 -*-
"""
Онтология слотов: загрузка, проверка целостности, разрешение ключа в слот.

Слот — единица сведений, общая для всех сценариев. Именно в слоты
классифицируется реплика оператора; факт конкретного заявителя находится
уже потом, простым просмотром сценария.
"""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import yaml

from ..types import Disclosure, Slot, SlotKind

DEFAULT_PATH = Path(__file__).resolve().parents[2] / "data" / "ontology" / "slots.yaml"


class OntologyError(Exception):
    """Онтология противоречива. Сборка с такой онтологией запрещена."""


@dataclass(slots=True)
class Ontology:
    version: str
    slots: dict[str, Slot]
    by_alias: dict[str, str]  # «ключ» и «группа/ключ» -> slot id
    overrides: dict[str, str]  # «сценарий:ключ» -> slot id

    # ------------------------------------------------------------ загрузка

    @staticmethod
    def load(path: str | Path = DEFAULT_PATH) -> "Ontology":
        raw = yaml.safe_load(Path(path).read_text(encoding="utf-8"))
        slots: dict[str, Slot] = {}
        by_alias: dict[str, str] = {}
        owner: dict[str, str] = {}

        for item in raw["slots"]:
            sid = item["id"]
            if sid in slots:
                raise OntologyError(f"слот {sid} объявлен дважды")
            if "." not in sid:
                raise OntologyError(f"id слота должен быть вида семья.имя: {sid}")
            aliases = tuple(item.get("aliases", ()))
            for a in aliases:
                if a in by_alias:
                    raise OntologyError(
                        f"алиас «{a}» занят слотом {owner[a]}, повтор в {sid}"
                    )
                by_alias[a] = sid
                owner[a] = sid
            slots[sid] = Slot(
                id=sid,
                label=item["label"],
                kind=SlotKind(item.get("kind", "value")),
                aliases=aliases,
                fallback=tuple(item.get("fallback", ())),
                questions=tuple(item.get("questions", ())),
                urge=item.get("urge", ""),
                default_disclosure=Disclosure(item.get("disclosure", "volunteered")),
                since=str(item.get("since", raw.get("version", "0.1"))),
            )

        for sid, slot in slots.items():
            for other in slot.fallback:
                if other not in slots:
                    raise OntologyError(
                        f"{sid}: fallback ссылается на несуществующий слот {other}"
                    )

        overrides = dict(raw.get("overrides") or {})
        for ref, sid in overrides.items():
            if sid not in slots:
                raise OntologyError(
                    f"override {ref} ссылается на несуществующий слот {sid}"
                )
            if ":" not in ref:
                raise OntologyError(
                    f"override должен быть вида сценарий:ключ, а не {ref}"
                )

        return Ontology(str(raw.get("version", "0.1")), slots, by_alias, overrides)

    # ---------------------------------------------------------- разрешение

    def resolve(
        self, key: str, group: str = "", scenario: str = "", explicit: str | None = None
    ) -> str | None:
        """Слот для факта. Порядок от точного к общему.

        1. явное поле `slot` в сценарии — так пишутся новые ситуации;
        2. точечный override для конкретного сценария;
        3. алиас «группа/ключ» — различает «фио» заявителя и «фио» пострадавшей;
        4. алиас «ключ».
        """
        if explicit:
            if explicit not in self.slots:
                raise OntologyError(
                    f"{scenario}:{key} ссылается на несуществующий слот {explicit}"
                )
            return explicit
        if scenario and (sid := self.overrides.get(f"{scenario}:{key}")):
            return sid
        if group and (sid := self.by_alias.get(f"{group}/{key}")):
            return sid
        return self.by_alias.get(key)

    def family(self, slot_id: str) -> str:
        return slot_id.split(".", 1)[0]

    def in_family(self, family: str) -> list[str]:
        return [s for s in self.slots if s.startswith(family + ".")]
