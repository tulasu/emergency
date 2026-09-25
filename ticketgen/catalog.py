"""Load and validate TraineeBox YAML catalog for ticket generation."""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml


@dataclass
class Tag:
    code: str
    title: str


@dataclass
class TagGroup:
    code: str
    title: str
    selection_mode: str
    parent_tag: str = ""
    tags: list[Tag] = field(default_factory=list)
    common: bool = False


@dataclass
class IncidentType:
    code: str
    title: str
    groups: list[TagGroup] = field(default_factory=list)
    include_common: list[str] = field(default_factory=list)


@dataclass
class Service:
    code: str
    title: str


@dataclass
class WhenRule:
    tags_any: list[str]
    add: list[str]


@dataclass
class RoutingRule:
    type_code: str
    always: list[str] = field(default_factory=list)
    when: list[WhenRule] = field(default_factory=list)


@dataclass
class Catalog:
    types: dict[str, IncidentType]
    services: dict[str, Service]
    routing: dict[str, RoutingRule]

    def type_summaries(self) -> list[dict[str, str]]:
        return [{"code": t.code, "title": t.title} for t in self.types.values()]

    def type_local_groups(self, type_code: str) -> list[TagGroup]:
        t = self.types.get(type_code)
        if not t:
            return []
        return [g for g in t.groups if not g.common]

    def common_groups_for_type(self, type_code: str) -> list[TagGroup]:
        t = self.types.get(type_code)
        if not t:
            return []
        return [g for g in t.groups if g.common]

    def visible_groups(self, groups: list[TagGroup], selected: list[str]) -> list[TagGroup]:
        selected_set = set(selected)
        return [g for g in groups if not g.parent_tag or g.parent_tag in selected_set]

    def group_prompt(self, g: TagGroup) -> dict[str, Any]:
        return {
            "group": g.code,
            "title": g.title,
            "mode": g.selection_mode,
            "parent_tag": g.parent_tag or None,
            "tags": [{"code": x.code, "title": x.title} for x in g.tags],
        }

    def tags_for_prompt(self, type_code: str, *, common: bool | None = None) -> list[dict[str, Any]]:
        t = self.types.get(type_code)
        if not t:
            return []
        out: list[dict[str, Any]] = []
        for g in t.groups:
            if common is True and not g.common:
                continue
            if common is False and g.common:
                continue
            out.append(self.group_prompt(g))
        return out

    def recommend_services(self, type_code: str, tag_codes: list[str]) -> list[str]:
        rule = self.routing.get(type_code)
        if not rule:
            return []
        selected = set(tag_codes)
        codes: list[str] = list(rule.always)
        seen = set(codes)
        for w in rule.when:
            if any(t in selected for t in w.tags_any):
                for c in w.add:
                    if c not in seen:
                        codes.append(c)
                        seen.add(c)
        return codes

    def validate_reference(self, ref: dict[str, Any]) -> None:
        type_code = (ref.get("incident_type_code") or "").strip()
        if not type_code or type_code not in self.types:
            raise ValueError(f"unknown incident_type_code: {type_code!r}")
        tags = list(ref.get("tag_codes") or [])
        services = list(ref.get("service_codes") or [])
        self._validate_tags(type_code, tags)
        for s in services:
            if s not in self.services:
                raise ValueError(f"unknown service_code: {s!r}")

    def repair_tags(
        self,
        type_code: str,
        tag_codes: list[str],
        groups_scope: list[TagGroup] | None = None,
    ) -> list[str]:
        t = self.types.get(type_code)
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

    def _validate_tags(self, type_code: str, tag_codes: list[str]) -> None:
        if not tag_codes:
            return
        groups = self.types[type_code].groups
        allowed = {tag.code for g in groups for tag in g.tags}
        selected = set(tag_codes)
        for code in selected:
            if code not in allowed:
                raise ValueError(f"tag {code!r} not allowed for type {type_code}")
        for g in groups:
            visible = not g.parent_tag or g.parent_tag in selected
            count = sum(1 for tag in g.tags if tag.code in selected)
            if not visible:
                if count > 0:
                    raise ValueError(f"tags from hidden group {g.code}")
                continue
            if g.selection_mode == "single" and count > 1:
                raise ValueError(f"too many tags in single group {g.code}")


def _parse_group(raw: dict[str, Any], *, common: bool = False) -> TagGroup:
    return TagGroup(
        code=raw["code"],
        title=raw.get("title") or raw["code"],
        selection_mode=raw.get("selection_mode") or "multi",
        parent_tag=raw.get("parent_tag") or "",
        tags=[Tag(code=t["code"], title=t.get("title") or t["code"]) for t in raw.get("tags") or []],
        common=common,
    )


def load_catalog(dir_path: str | Path) -> Catalog:
    root = Path(dir_path)
    with (root / "common_tags.yaml").open(encoding="utf-8") as f:
        common_raw = yaml.safe_load(f) or {}
    common_groups = {
        k: _parse_group(v, common=True) for k, v in (common_raw.get("groups") or {}).items()
    }

    with (root / "services.yaml").open(encoding="utf-8") as f:
        services_raw = yaml.safe_load(f) or {}
    services = {
        s["code"]: Service(code=s["code"], title=s.get("title") or s["code"])
        for s in services_raw.get("services") or []
    }

    types: dict[str, IncidentType] = {}
    types_dir = root / "types"
    for path in sorted(types_dir.glob("*.yaml")):
        with path.open(encoding="utf-8") as f:
            raw = yaml.safe_load(f) or {}
        code = raw["code"]
        include_common = list(raw.get("include_common") or [])
        groups: list[TagGroup] = []
        for key in include_common:
            if key in common_groups:
                groups.append(common_groups[key])
        for g in raw.get("groups") or []:
            parsed = _parse_group(g, common=False)
            if parsed.code in common_groups:
                parsed.common = True
            groups.append(parsed)
        types[code] = IncidentType(
            code=code,
            title=raw.get("title") or code,
            groups=groups,
            include_common=include_common,
        )

    routing: dict[str, RoutingRule] = {}
    routing_path = root / "routing.yaml"
    if routing_path.exists():
        with routing_path.open(encoding="utf-8") as f:
            routing_raw = yaml.safe_load(f) or {}
        for rule in routing_raw.get("rules") or []:
            type_code = rule.get("match", {}).get("type")
            if not type_code:
                continue
            when = [
                WhenRule(tags_any=w.get("tags_any") or [], add=w.get("add") or [])
                for w in rule.get("when") or []
            ]
            routing[type_code] = RoutingRule(
                type_code=type_code,
                always=list(rule.get("always") or []),
                when=when,
            )

    return Catalog(types=types, services=services, routing=routing)
