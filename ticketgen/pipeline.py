"""Atomic generation pipeline: scenario → type → type tags → common → services."""

from __future__ import annotations

import json
import logging
import re
from dataclasses import dataclass, field
from typing import Any, Callable, Protocol

from catalog import Catalog, TagGroup
from llm import LLMClient

log = logging.getLogger("ticketgen.pipeline")

try:
    from faker import Faker
except ImportError:  # pragma: no cover
    Faker = None  # type: ignore[misc, assignment]


class LLM(Protocol):
    def chat_json(
        self,
        system: str,
        user: str,
        *,
        temperature: float = 0.4,
        max_tokens: int = 1024,
    ) -> dict: ...


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


_TYPE_ALIASES = {
    "traffic_accidents": "traffic_accident",
    "dtp": "traffic_accident",
    "accident": "traffic_accident",
    "fire": "101",
    "пожар": "101",
    "gas": "104",
    "газ": "104",
}

SCENARIO_SYSTEM = (
    "Ты помощник тренажёра службы 112. По короткому описанию происшествия "
    "придумай реалистичный сценарий звонка заявителя. Ответь строго JSON: "
    '{"title":"...","scenario":"..."}. '
    "title — короткий заголовок билета. scenario — 2–5 предложений: что произошло, "
    "где (с адресом), кто звонит. Не выбирай тип происшествия."
)

TYPE_SYSTEM = (
    "Ты классификатор происшествий 112. По сценарию выбери ОДИН тип. "
    "Ответь строго JSON: {\"incident_type_code\":\"...\"}. "
    "code — ТОЧНО из списка (не придумывай свои коды)."
)

GROUP_TAG_SYSTEM = (
    "Ты заполняешь карточку происшествия 112. "
    "Выбери теги ТОЛЬКО из одной группы. Ответь строго JSON: "
    '{"tag_codes":["code"]}. '
    "Если selection_mode=single — ровно один тег (обязательно, если группа без parent "
    "или родитель уже выбран). Если multi — 0 или больше релевантных. "
    "Используй только коды из списка."
)

COMMON_TAGS_SYSTEM = (
    "Ты заполняешь общие признаки происшествия 112. "
    "Ответь строго JSON: {\"tag_codes\":[\"code\"]}. "
    "Для каждой single-группы выбери ровно один тег. "
    "Для multi — только если явно следует из сценария."
)

SERVICES_SYSTEM = (
    "Ты маршрутизируешь вызов 112. "
    "Ответь строго JSON: {\"service_codes\":[\"code\"]}. "
    "Выбирай только из предложенного списка. Можно добавить релевантные к recommended."
)


class Pipeline:
    """Pure function: prompt -> draft (no DB, no queue, no dialog link).

    Traineebox owns the job rows (building_dialog/checking_dialog) and calls
    run() per stage; dialog snapshots are authored by teachers via
    PUT /tickets/{id}/scenario — ticketgen output feeds briefing/reference
    only, never dialog facts directly.
    """

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

        def stage(name: str) -> None:
            if on_stage:
                on_stage(name, state)

        state.title, state.scenario = self.generate_scenario(prompt)
        stage("enriching")

        state.pii = self.fill_pii()
        if state.pii.get("address") and state.pii["address"] not in state.scenario:
            state.scenario = f"{state.scenario} Адрес: {state.pii['address']}."
        stage("filling_pii")

        state.type_code = self.pick_type(state.scenario, prompt)
        stage("picking_type")

        state.type_tags = self.fill_type_tags(state.scenario, state.type_code)
        stage("tagging_type")

        state.common_tags = self.fill_common_tags(state.scenario, state.type_code)
        stage("tagging_common")

        state.service_codes = self.pick_services(
            state.scenario, state.type_code, state.all_tags()
        )
        stage("building_services")

        ref = state.draft_reference()
        self.catalog.validate_reference(ref)
        return PipelineResult(
            draft_title=state.title[:256],
            scenario_text=state.scenario,
            draft_reference=ref,
        )

    def draft_scenario(self, prompt: str) -> PipelineResult:
        """Pure draft: prompt -> title/scenario/reference (no side effects)."""
        return self.run(prompt)

    def dialog_report(self, result: PipelineResult) -> dict[str, Any]:
        """Report traineebox stores on the job before approve.

        building_dialog: briefing payload; checking_dialog: lint/PII gate input.
        Slot-fact authoring stays teacher-side (PUT /scenario).
        """
        ref = dict(result.draft_reference)
        return {
            "draft_title": result.draft_title,
            "scenario_text": result.scenario_text,
            "draft_reference": ref,
            "stages": ["building_dialog", "checking_dialog"],
            "pii_keys": [k for k in ("applicant_last_name", "applicant_first_name",
                                        "caller_number", "dictated_number") if ref.get(k)],
        }

    def generate_scenario(self, prompt: str) -> tuple[str, str]:
        user = f"Описание преподавателя: {prompt}"
        data = self.llm.chat_json(SCENARIO_SYSTEM, user, temperature=0.5, max_tokens=512)
        scenario = str(data.get("scenario") or "").strip()
        if not scenario:
            raise ValueError("enrich returned empty scenario")
        title = str(data.get("title") or prompt).strip() or prompt
        return title, scenario

    def fill_pii(self) -> dict[str, str]:
        if self.faker is None:
            return {
                "last_name": "Иванов",
                "first_name": "Иван",
                "caller_number": "79001234567",
                "dictated_number": "79001234567",
                "address": "г. Москва, ул. Тверская, д. 1",
            }
        return {
            "last_name": self.faker.last_name(),
            "first_name": self.faker.first_name(),
            "caller_number": "79" + self.faker.numerify("#########"),
            "dictated_number": "79" + self.faker.numerify("#########"),
            "address": re.sub(r"\s+", " ", self.faker.address().replace("\n", ", ")),
        }

    def pick_type(self, scenario: str, prompt: str = "") -> str:
        hinted = self._keyword_type(prompt)
        types_json = json.dumps(self.catalog.type_summaries(), ensure_ascii=False)
        user = f"Сценарий:\n{scenario}\n\nДоступные типы:\n{types_json}"
        data = self.llm.chat_json(TYPE_SYSTEM, user, temperature=0.1, max_tokens=64)
        raw = str(data.get("incident_type_code") or "").strip()
        try:
            resolved = self.resolve_type_code(raw, prompt or scenario)
        except ValueError:
            if hinted:
                return hinted
            raise
        if hinted and hinted != resolved and self._strong_keyword(prompt):
            log.info("type override %s -> %s (prompt hint)", resolved, hinted)
            return hinted
        return resolved

    def _strong_keyword(self, text: str) -> bool:
        p = (text or "").lower()
        return any(w in p for w in ("пожар", "горит", "дым", "дтп", "газ", "gas", "fire"))

    def _keyword_type(self, text: str) -> str | None:
        p = (text or "").lower()
        if any(w in p for w in ("пожар", "горит", "дым", "fire")) and "101" in self.catalog.types:
            return "101"
        if any(w in p for w in ("дтп", "столкнов")) and "traffic_accident" in self.catalog.types:
            return "traffic_accident"
        if any(w in p for w in ("газ", "gas")) and "104" in self.catalog.types:
            return "104"
        return None

    def resolve_type_code(self, raw: str, prompt: str = "") -> str:
        code = (raw or "").strip()
        if code in self.catalog.types:
            return code
        lowered = code.lower().replace(" ", "_").replace("-", "_")
        if lowered in _TYPE_ALIASES and _TYPE_ALIASES[lowered] in self.catalog.types:
            return _TYPE_ALIASES[lowered]
        if lowered in self.catalog.types:
            return lowered
        needle = (code or prompt).lower()
        for t in self.catalog.types.values():
            if needle and (needle in t.title.lower() or t.title.lower() in needle):
                return t.code
        hinted = self._keyword_type(prompt + " " + code)
        if hinted:
            return hinted
        raise ValueError(f"enrich returned unknown type {raw!r}")

    def fill_type_tags(self, scenario: str, type_code: str) -> list[str]:
        groups = self.catalog.type_local_groups(type_code)
        selected: list[str] = []
        processed: set[str] = set()
        for _ in range(24):
            visible = [
                g for g in self.catalog.visible_groups(groups, selected) if g.code not in processed
            ]
            if not visible:
                break
            g = visible[0]
            codes = self._pick_group_tags(scenario, type_code, g, require_single=True)
            selected.extend(codes)
            selected = self.repair_tags(type_code, selected, groups_scope=groups)
            processed.add(g.code)
        return selected

    def fill_common_tags(self, scenario: str, type_code: str) -> list[str]:
        groups = self.catalog.common_groups_for_type(type_code)
        if not groups:
            return []
        prompt_groups = [self.catalog.group_prompt(g) for g in groups]
        user = (
            f"Сценарий:\n{scenario}\n\n"
            f"Тип: {type_code}\n"
            f"Общие группы тегов:\n{json.dumps(prompt_groups, ensure_ascii=False)}\n"
        )
        data = self.llm.chat_json(COMMON_TAGS_SYSTEM, user, temperature=0.1, max_tokens=256)
        raw = _as_code_list(data.get("tag_codes"))
        selected = self.repair_tags(type_code, raw, groups_scope=groups)
        allowed = {tag.code for g in groups for tag in g.tags}
        selected = [c for c in selected if c in allowed]
        for g in groups:
            if g.selection_mode != "single":
                continue
            if any(c in {t.code for t in g.tags} for c in selected):
                continue
            forced = self._pick_group_tags(scenario, type_code, g, require_single=True)
            selected.extend(forced)
            selected = self.repair_tags(type_code, selected, groups_scope=groups)
        return [c for c in selected if c in allowed]

    def pick_services(self, scenario: str, type_code: str, tag_codes: list[str]) -> list[str]:
        recommended = self.catalog.recommend_services(type_code, tag_codes)
        service_hint = [
            {"code": c, "title": self.catalog.services[c].title}
            for c in recommended
            if c in self.catalog.services
        ][:16]
        if not service_hint and recommended:
            service_hint = [{"code": c, "title": c} for c in recommended[:16]]
        if not service_hint:
            return []
        user = (
            f"Сценарий:\n{scenario}\n"
            f"Тип: {type_code}\n"
            f"Теги: {tag_codes}\n"
            f"Рекомендованные службы:\n{json.dumps(service_hint, ensure_ascii=False)}\n"
        )
        try:
            data = self.llm.chat_json(SERVICES_SYSTEM, user, temperature=0.1, max_tokens=128)
            llm_services = _as_code_list(data.get("service_codes"))
        except Exception as e:  # noqa: BLE001
            log.warning("services llm failed: %s", e)
            llm_services = []
        merged: list[str] = []
        seen: set[str] = set()
        for c in recommended + llm_services:
            if c in self.catalog.services and c not in seen:
                merged.append(c)
                seen.add(c)
        return merged

    def _pick_group_tags(
        self,
        scenario: str,
        type_code: str,
        g: TagGroup,
        *,
        require_single: bool,
    ) -> list[str]:
        gp = self.catalog.group_prompt(g)
        user = (
            f"Сценарий:\n{scenario}\n"
            f"Тип: {type_code}\n"
            f"Группа:\n{json.dumps(gp, ensure_ascii=False)}\n"
        )
        if require_single and g.selection_mode == "single":
            user += "Обязательно выбери ровно один tag_code.\n"
        data = self.llm.chat_json(GROUP_TAG_SYSTEM, user, temperature=0.1, max_tokens=128)
        codes = _as_code_list(data.get("tag_codes"))
        allowed = {t.code for t in g.tags}
        codes = [c for c in codes if c in allowed]
        if g.selection_mode == "single":
            if codes:
                return [codes[0]]
            if require_single and g.tags:
                log.warning("group %s empty pick; fallback %s", g.code, g.tags[0].code)
                return [g.tags[0].code]
            return []
        return codes

    def repair_tags(
        self,
        type_code: str,
        tag_codes: list[str],
        groups_scope: list[TagGroup] | None = None,
    ) -> list[str]:
        t = self.catalog.types.get(type_code)
        if not t:
            return []
        groups = groups_scope if groups_scope is not None else t.groups
        allowed = {tag.code for g in groups for tag in g.tags}
        selected = [c for c in tag_codes if c in allowed]
        for _ in range(4):
            keep: list[str] = []
            selected_set = set(selected)
            for g in groups:
                visible = not g.parent_tag or g.parent_tag in selected_set
                if not visible:
                    continue
                group_tags = [c for c in selected if c in {x.code for x in g.tags}]
                if not group_tags:
                    continue
                if g.selection_mode == "single":
                    keep.append(group_tags[0])
                else:
                    keep.extend(group_tags)
            if keep == selected:
                break
            selected = keep
        out: list[str] = []
        seen: set[str] = set()
        for c in selected:
            if c not in seen:
                out.append(c)
                seen.add(c)
        return out


def _as_code_list(raw) -> list[str]:
    if not raw:
        return []
    out: list[str] = []
    for item in raw:
        if isinstance(item, str):
            code = item.strip()
        elif isinstance(item, dict):
            code = str(item.get("code") or item.get("service_code") or item.get("tag_code") or "").strip()
        else:
            code = str(item).strip()
        if code:
            out.append(code)
    return out
