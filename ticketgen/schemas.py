"""Pydantic response models built from the YAML catalog."""

from __future__ import annotations

from collections.abc import Sequence
from typing import Literal

from pydantic import BaseModel, Field, create_model

from catalog import Catalog, TagGroup


class ScenarioDraft(BaseModel):
    title: str = Field(min_length=1)
    scenario: str = Field(min_length=1)


def _literal(codes: Sequence[str]):
    uniq = tuple(dict.fromkeys(codes))
    if not uniq:
        raise ValueError("cannot build enum from empty codes")
    return Literal[*uniq]


def type_model(catalog: Catalog) -> type[BaseModel]:
    return create_model(
        "TypePick",
        incident_type_code=(_literal(catalog.types), ...),
    )


def group_tags_model(group: TagGroup) -> type[BaseModel]:
    codes = tuple(t.code for t in group.tags)
    name = f"GroupTags_{group.code}"
    if not codes:
        return create_model(name, tag_codes=(list[str], Field(default_factory=list, max_length=0)))
    item = _literal(codes)
    if group.selection_mode == "single":
        return create_model(
            name,
            tag_codes=(list[item], Field(min_length=1, max_length=1)),
        )
    return create_model(
        name,
        tag_codes=(list[item], Field(default_factory=list)),
    )


def tags_model(groups: Sequence[TagGroup], name: str) -> type[BaseModel]:
    codes = tuple(dict.fromkeys(t.code for g in groups for t in g.tags))
    if not codes:
        return create_model(name, tag_codes=(list[str], Field(default_factory=list, max_length=0)))
    return create_model(
        name,
        tag_codes=(list[_literal(codes)], Field(default_factory=list)),
    )


def services_model(catalog: Catalog) -> type[BaseModel]:
    return create_model(
        "ServicePick",
        service_codes=(list[_literal(catalog.services)], Field(default_factory=list)),
    )
