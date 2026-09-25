# -*- coding: utf-8 -*-
"""Pure-function gate: POST /draft with a fake LLM (no network, no DB)."""

from __future__ import annotations

import json
import sys
import threading
import urllib.request
from http.server import ThreadingHTTPServer
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

import server as tgs  # noqa: E402
from catalog import Catalog, IncidentType, Service, Tag, TagGroup  # noqa: E402
from pipeline import Pipeline  # noqa: E402


class FakeLLM:
    def complete(self, system, user, response_model, **kw):
        fields = response_model.model_fields
        if "incident_type_code" in fields:
            return response_model.model_validate({"incident_type_code": "101"})
        if "title" in fields and "scenario" in fields:
            return response_model.model_validate(
                {"title": "Пожар", "scenario": "Горит склад. Звонит сосед."}
            )
        if "tag_codes" in fields:
            schema = response_model.model_json_schema()
            props = schema.get("properties", {}).get("tag_codes", {})
            first = _first_item(props)
            if props.get("minItems", 0) >= 1 and first:
                return response_model.model_validate({"tag_codes": [first]})
            return response_model.model_validate({"tag_codes": []})
        if "service_codes" in fields:
            return response_model.model_validate({"service_codes": ["sluzhba_101"]})
        raise RuntimeError(f"unexpected model {response_model}")


def _first_item(props: dict) -> str | None:
    items = props.get("items", {})
    if "const" in items:
        return items["const"]
    if "enum" in items:
        return items["enum"][0]
    ref = items.get("$ref")
    if ref:
        return ref.rsplit("/", 1)[-1]
    return None


def tiny_catalog() -> Catalog:
    g = TagGroup(code="where", title="Где", selection_mode="single",
                 tags=[Tag(code="where_street", title="Улица")])
    t = IncidentType(code="101", title="Пожар", groups=[g], include_common=[])
    return Catalog(types={"101": t}, services={"sluzhba_101": Service("sluzhba_101", "101")},
                   routing={})


def test_draft_endpoint_pure():
    tgs.PIPE = Pipeline(tiny_catalog(), FakeLLM())
    srv = ThreadingHTTPServer(("127.0.0.1", 18399), tgs.H)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    try:
        def post(obj):
            req = urllib.request.Request("http://127.0.0.1:18399/draft",
                                         json.dumps(obj).encode(),
                                         {"Content-Type": "application/json"})
            try:
                with urllib.request.urlopen(req) as r:
                    return r.status, json.loads(r.read())
            except urllib.error.HTTPError as e:
                return e.code, json.loads(e.read())

        code, out = post({"prompt": "пожар на складе"})
        assert code == 200, out
        assert out["draft_title"] and out["scenario_text"]
        assert out["draft_reference"]["incident_type_code"] == "101"
        assert "dialog_report" in out
        assert post({})[0] == 400  # prompt required
        assert post({"prompt": ""})[0] == 400
    finally:
        srv.shutdown()
