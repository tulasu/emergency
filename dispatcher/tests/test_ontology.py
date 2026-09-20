# -*- coding: utf-8 -*-
"""Онтология: целостность и разрешение ключа в слот."""

import pytest
import yaml

from dispatcher.data.ontology import Ontology, OntologyError
from dispatcher.types import Disclosure, SlotKind

ONTO = Ontology.load()


def test_ontology_loads():
    assert ONTO.slots and ONTO.by_alias


def test_every_slot_has_family_in_id():
    for sid in ONTO.slots:
        assert "." in sid, sid


def test_alias_owned_by_single_slot(tmp_path):
    bad = tmp_path / "bad.yaml"
    bad.write_text(
        yaml.safe_dump(
            {
                "version": "0.1",
                "slots": [
                    {"id": "a.one", "label": "раз", "aliases": ["общий"]},
                    {"id": "a.two", "label": "два", "aliases": ["общий"]},
                ],
            },
            allow_unicode=True,
        ),
        encoding="utf-8",
    )
    with pytest.raises(OntologyError, match="занят слотом"):
        Ontology.load(bad)


def test_group_disambiguates_same_keys():
    # «фио» в одном сценарии — заявитель, в другом — пострадавшая
    assert ONTO.resolve("фио", "заявитель") == "caller.name"
    assert ONTO.resolve("фио", "пациент") == "victim.name"


def test_unknown_key_unresolved():
    assert ONTO.resolve("такого_ключа_нет") is None


def test_dangling_slot_link_forbidden():
    with pytest.raises(OntologyError, match="несуществующий слот"):
        ONTO.resolve("что угодно", explicit="нет.такого")


def test_address_refinement_on_request_by_default():
    assert ONTO.slots["addr.refined"].default_disclosure is Disclosure.ON_REQUEST
    assert ONTO.slots["addr.full"].default_disclosure is Disclosure.VOLUNTEERED


def test_slot_family_for_ellipsis():
    assert ONTO.family("addr.building") == "addr"
    assert "addr.landmark" in ONTO.in_family("addr")
    assert all(s.startswith("addr.") for s in ONTO.in_family("addr"))


def test_slot_kind_parsed():
    assert ONTO.slots["building.gas"].kind is SlotKind.YESNO
