"""Pipeline steps. Order is declared in pipeline.STEPS."""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from typing import Any

from catalog import Catalog, TagGroup
from llm import LLM
from schemas import ScenarioDraft, group_tags_model, services_model, tags_model, type_model

try:
    from faker import Faker
except ImportError:  # pragma: no cover
    Faker = None  # type: ignore[misc, assignment]


SCENARIO_SYSTEM = (
    "Ты помощник тренажёра службы 112. По описанию преподавателя "
    "разверни сценарий звонка заявителя. Ответь строго JSON: "
    '{"title":"...","scenario":"..."}. '
    "title — короткая перефразировка описания, те же объекты и событие "
    "(если горит бак с мусором — так и пиши, не подменяй на огнетушитель, взрыв и т.п.). "
    "scenario — 2–5 предложений: что произошло, где (с адресом), кто звонит. "
    "Не добавляй новых угроз, предметов и исходов, которых нет в описании. "
    "Не выбирай тип происшествия."
)

TYPE_SYSTEM = (
    "Ты классификатор происшествий 112. По сценарию выбери ОДИН тип. "
    "Ответь строго JSON: {\"incident_type_code\":\"...\"}. "
    "code — ТОЧНО из списка (не придумывай свои коды). "
    "Классифицируй то, что уже произошло, не домыслы («может взорваться»)."
)

GROUP_TAG_SYSTEM = (
    "Ты заполняешь карточку происшествия 112. "
    "Выбери теги ТОЛЬКО из одной группы. Ответь строго JSON: "
    '{"tag_codes":["code"]}. '
    "Если selection_mode=single — ровно один тег. Если multi — 0 или больше релевантных. "
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


@dataclass
class Ctx:
    catalog: Catalog
    llm: LLM
    faker: Any
    state: Any


class DraftScenario:
    name = "enriching"

    def run(self, ctx: Ctx) -> None:
        user = f"Описание преподавателя: {ctx.state.prompt}"
        draft = ctx.llm.complete(
            SCENARIO_SYSTEM, user, ScenarioDraft, temperature=0.2, max_tokens=192
        )
        ctx.state.title = draft.title.strip()
        ctx.state.scenario = draft.scenario.strip()


class FillPII:
    name = "filling_pii"

    def run(self, ctx: Ctx) -> None:
        if ctx.faker is None:
            pii = {
                "last_name": "Иванов",
                "first_name": "Иван",
                "caller_number": "79001234567",
                "dictated_number": "79001234567",
                "address": "г. Москва, ул. Тверская, д. 1",
            }
        else:
            pii = {
                "last_name": ctx.faker.last_name(),
                "first_name": ctx.faker.first_name(),
                "caller_number": "79" + ctx.faker.numerify("#########"),
                "dictated_number": "79" + ctx.faker.numerify("#########"),
                "address": re.sub(r"\s+", " ", ctx.faker.address().replace("\n", ", ")),
            }
        ctx.state.pii = pii
        if pii.get("address") and pii["address"] not in ctx.state.scenario:
            ctx.state.scenario = f"{ctx.state.scenario} Адрес: {pii['address']}."


class PickType:
    name = "picking_type"

    def run(self, ctx: Ctx) -> None:
        types_json = json.dumps(ctx.catalog.type_summaries(), ensure_ascii=False)
        user = f"Сценарий:\n{ctx.state.scenario}\n\nДоступные типы:\n{types_json}"
        pick = ctx.llm.complete(TYPE_SYSTEM, user, type_model(ctx.catalog), temperature=0.1, max_tokens=32)
        ctx.state.type_code = pick.incident_type_code


class FillTypeTags:
    name = "tagging_type"

    def run(self, ctx: Ctx) -> None:
        groups = ctx.catalog.type_local_groups(ctx.state.type_code)
        selected: list[str] = []
        processed: set[str] = set()
        for _ in range(24):
            visible = [
                g
                for g in ctx.catalog.visible_groups(groups, selected)
                if g.code not in processed
            ]
            if not visible:
                break
            g = visible[0]
            selected.extend(_pick_group_tags(ctx, g))
            selected = ctx.catalog.repair_tags(
                ctx.state.type_code, selected, groups_scope=groups
            )
            processed.add(g.code)
        ctx.state.type_tags = selected


class FillCommonTags:
    name = "tagging_common"

    def run(self, ctx: Ctx) -> None:
        groups = ctx.catalog.common_groups_for_type(ctx.state.type_code)
        if not groups:
            ctx.state.common_tags = []
            return
        prompt_groups = [ctx.catalog.group_prompt(g) for g in groups]
        user = (
            f"Сценарий:\n{ctx.state.scenario}\n\n"
            f"Тип: {ctx.state.type_code}\n"
            f"Общие группы тегов:\n{json.dumps(prompt_groups, ensure_ascii=False)}\n"
        )
        picked = ctx.llm.complete(
            COMMON_TAGS_SYSTEM,
            user,
            tags_model(groups, "CommonTags"),
            temperature=0.1,
            max_tokens=96,
        )
        selected = ctx.catalog.repair_tags(
            ctx.state.type_code, list(picked.tag_codes), groups_scope=groups
        )
        allowed = {tag.code for g in groups for tag in g.tags}
        selected = [c for c in selected if c in allowed]
        for g in groups:
            if g.selection_mode != "single":
                continue
            if any(c in {t.code for t in g.tags} for c in selected):
                continue
            selected.extend(_pick_group_tags(ctx, g))
            selected = ctx.catalog.repair_tags(
                ctx.state.type_code, selected, groups_scope=groups
            )
        ctx.state.common_tags = [c for c in selected if c in allowed]


class PickServices:
    name = "building_services"

    def run(self, ctx: Ctx) -> None:
        if not ctx.catalog.services:
            ctx.state.service_codes = []
            return
        recommended = ctx.catalog.recommend_services(ctx.state.type_code, ctx.state.all_tags())
        service_hint = [
            {"code": c, "title": ctx.catalog.services[c].title}
            for c in recommended
            if c in ctx.catalog.services
        ][:16]
        if not service_hint:
            service_hint = [
                {"code": c, "title": s.title} for c, s in list(ctx.catalog.services.items())[:16]
            ]
        user = (
            f"Сценарий:\n{ctx.state.scenario}\n"
            f"Тип: {ctx.state.type_code}\n"
            f"Теги: {ctx.state.all_tags()}\n"
            f"Рекомендованные службы:\n{json.dumps(service_hint, ensure_ascii=False)}\n"
        )
        picked = ctx.llm.complete(
            SERVICES_SYSTEM,
            user,
            services_model(ctx.catalog),
            temperature=0.1,
            max_tokens=48,
        )
        merged: list[str] = []
        seen: set[str] = set()
        for c in recommended + list(picked.service_codes):
            if c in ctx.catalog.services and c not in seen:
                merged.append(c)
                seen.add(c)
        ctx.state.service_codes = merged


def _pick_group_tags(ctx: Ctx, g: TagGroup) -> list[str]:
    if not g.tags:
        return []
    gp = ctx.catalog.group_prompt(g)
    user = (
        f"Сценарий:\n{ctx.state.scenario}\n"
        f"Тип: {ctx.state.type_code}\n"
        f"Группа:\n{json.dumps(gp, ensure_ascii=False)}\n"
    )
    if g.selection_mode == "single":
        user += "Обязательно выбери ровно один tag_code.\n"
    picked = ctx.llm.complete(
        GROUP_TAG_SYSTEM,
        user,
        group_tags_model(g),
        temperature=0.1,
        max_tokens=48,
    )
    return list(picked.tag_codes)
