"""Unit tests for catalog helpers and atomic pipeline stages."""

from __future__ import annotations

import sys
from pathlib import Path

import pytest
from pydantic import ValidationError

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from catalog import load_catalog  # noqa: E402
from pipeline import Pipeline  # noqa: E402
from schemas import type_model  # noqa: E402

CATALOG_DIR = Path(__file__).resolve().parents[2] / "artifacts" / "etc" / "traineebox" / "catalog"


class MockLLM:
    def __init__(self, responses: list[dict]):
        self.responses = list(responses)
        self.calls = 0
        self.prompts: list[str] = []

    def complete(self, system: str, user: str, response_model, *, temperature: float = 0.4, max_tokens: int = 1024):
        self.prompts.append(user)
        if self.calls >= len(self.responses):
            raise RuntimeError(f"no more mock responses (call {self.calls})")
        out = self.responses[self.calls]
        self.calls += 1
        return response_model.model_validate(out)


@pytest.fixture(scope="module")
def catalog():
    if not CATALOG_DIR.exists():
        pytest.skip(f"catalog missing: {CATALOG_DIR}")
    return load_catalog(CATALOG_DIR)


def test_load_catalog_has_types(catalog):
    assert "101" in catalog.types
    assert "sluzhba_101" in catalog.services
    assert catalog.recommend_services("101", []) == ["sluzhba_101"]


def test_local_vs_common_groups(catalog):
    local = catalog.type_local_groups("101")
    common = catalog.common_groups_for_type("101")
    assert local
    assert all(not g.common for g in local)
    assert all(g.common for g in common)
    assert {g.code for g in common} >= {"threat_people", "medical_help", "evacuation"}
    where = next(g for g in local if g.code == "where")
    assert where.parent_tag == ""
    fire_sign = next(g for g in local if g.code == "fire_sign_street")
    assert fire_sign.parent_tag == "where_street"
    visible0 = catalog.visible_groups(local, [])
    assert any(g.code == "where" for g in visible0)
    assert not any(g.code == "fire_sign_street" for g in visible0)
    visible1 = catalog.visible_groups(local, ["where_street"])
    assert any(g.code == "fire_sign_street" for g in visible1)


def test_validate_reference_ok(catalog):
    catalog.validate_reference(
        {
            "incident_type_code": "101",
            "tag_codes": ["where_street", "street_flame_smoke", "burn_trash"],
            "service_codes": ["sluzhba_101"],
        }
    )


def test_validate_reference_bad_tag(catalog):
    with pytest.raises(ValueError):
        catalog.validate_reference(
            {
                "incident_type_code": "101",
                "tag_codes": ["not_a_real_tag"],
                "service_codes": [],
            }
        )


def test_type_schema_from_yaml(catalog):
    model = type_model(catalog)
    assert model.model_validate({"incident_type_code": "101"}).incident_type_code == "101"
    with pytest.raises(ValidationError):
        model.model_validate({"incident_type_code": "пожар"})
    with pytest.raises(ValidationError):
        model.model_validate({"incident_type_code": "fire"})


def test_atomic_pipeline_walk_101(catalog):
    """scenario → type → per-group tags → common → services."""
    llm = MockLLM(
        [
            {"title": "Горит мусор", "scenario": "На улице горит бак около дома."},
            {"incident_type_code": "101"},
            {"tag_codes": ["where_street"]},
            {"tag_codes": ["street_flame_smoke"]},
            {"tag_codes": ["burn_trash"]},
            {"tag_codes": ["place_pedestrian"]},
            {"tag_codes": ["gas_unknown"]},
            {"tag_codes": [
                "threat_people_yes",
                "medical_help_no",
                "evacuation_no",
                "no_access",
            ]},
            {"service_codes": ["sluzhba_101"]},
        ]
    )
    stages: list[str] = []
    result = Pipeline(catalog, llm=llm).run(
        "пожар",
        on_stage=lambda name, _state: stages.append(name),
    )
    assert stages == [
        "enriching",
        "filling_pii",
        "picking_type",
        "tagging_type",
        "tagging_common",
        "building_services",
    ]
    assert result.draft_reference["incident_type_code"] == "101"
    tags = result.draft_reference["tag_codes"]
    assert "where_street" in tags
    assert "street_flame_smoke" in tags
    assert "burn_trash" in tags
    assert "threat_people_yes" in tags
    assert "sluzhba_101" in result.draft_reference["service_codes"]
    assert result.draft_reference["applicant_last_name"]
    assert len(tags) >= 4


def test_pipeline_fails_on_bad_type(catalog):
    llm = MockLLM(
        [
            {"title": "x", "scenario": "непонятная ситуация без деталей"},
            {"incident_type_code": "no_such_type_xyz"},
        ]
    )
    with pytest.raises(ValidationError):
        Pipeline(catalog, llm=llm).run("абвгд неизвестное")
