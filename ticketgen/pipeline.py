"""Generation pipeline: STEPS defines the call order."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Callable

from catalog import Catalog
from llm import LLM, LLMClient
from steps import (
    Ctx,
    DraftScenario,
    FillCommonTags,
    FillPII,
    FillTypeTags,
    PickServices,
    PickType,
)

try:
    from faker import Faker
except ImportError:  # pragma: no cover
    Faker = None  # type: ignore[misc, assignment]


STEPS = [
    DraftScenario(),
    FillPII(),
    PickType(),
    FillTypeTags(),
    FillCommonTags(),
    PickServices(),
]


@dataclass
class PipelineResult:
    draft_title: str
    scenario_text: str
    draft_reference: dict[str, Any]


@dataclass
class PipelineState:
    prompt: str = ""
    title: str = ""
    scenario: str = ""
    type_code: str = ""
    type_tags: list[str] = field(default_factory=list)
    common_tags: list[str] = field(default_factory=list)
    service_codes: list[str] = field(default_factory=list)
    pii: dict[str, str] = field(default_factory=dict)

    def all_tags(self) -> list[str]:
        out: list[str] = []
        seen: set[str] = set()
        for c in self.type_tags + self.common_tags:
            if c not in seen:
                out.append(c)
                seen.add(c)
        return out

    def draft_reference(self) -> dict[str, Any]:
        return {
            "incident_type_code": self.type_code,
            "tag_codes": self.all_tags(),
            "service_codes": list(self.service_codes),
            "applicant_last_name": self.pii.get("last_name", ""),
            "applicant_first_name": self.pii.get("first_name", ""),
            "caller_number": self.pii.get("caller_number", ""),
            "dictated_number": self.pii.get("dictated_number", ""),
        }


class Pipeline:
    def __init__(self, catalog: Catalog, llm: LLM | None = None, locale: str = "ru_RU"):
        self.catalog = catalog
        self.llm = llm or LLMClient()
        self.faker = Faker(locale) if Faker else None

    def run(
        self,
        prompt: str,
        on_stage: Callable[[str, PipelineState], None] | None = None,
    ) -> PipelineResult:
        state = PipelineState(prompt=prompt)
        ctx = Ctx(catalog=self.catalog, llm=self.llm, faker=self.faker, state=state)
        for step in STEPS:
            step.run(ctx)
            if on_stage:
                on_stage(step.name, state)
        ref = state.draft_reference()
        self.catalog.validate_reference(ref)
        return PipelineResult(
            draft_title=state.title[:256],
            scenario_text=state.scenario,
            draft_reference=ref,
        )

    def draft_scenario(self, prompt: str) -> PipelineResult:
        return self.run(prompt)

    def dialog_report(self, result: PipelineResult) -> dict[str, Any]:
        ref = dict(result.draft_reference)
        return {
            "draft_title": result.draft_title,
            "scenario_text": result.scenario_text,
            "draft_reference": ref,
            "stages": [step.name for step in STEPS],
            "pii_keys": [
                k
                for k in (
                    "applicant_last_name",
                    "applicant_first_name",
                    "caller_number",
                    "dictated_number",
                )
                if ref.get(k)
            ],
        }
