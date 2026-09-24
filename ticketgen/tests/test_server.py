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
    def chat_json(self, system, user, **kw):
        if "классификатор" in system or "incident_type" in system.lower() or "ОДИН тип" in system:
            return {"incident_type_code": "101"}
        if "title" in system and "scenario" in system:
            return {"title": "Пожар", "scenario": "Горит склад. Звонит сосед."}
        if "tag_codes" in system:
            return {"tag_codes": []}
        if "service_codes" in system:
            return {"service_codes": ["sluzhba_101"]}
        return {}


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
