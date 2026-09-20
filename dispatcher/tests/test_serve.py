# -*- coding: utf-8 -*-
"""Шаг 6: HTTP-контракт сервера — текст полностью, /rtp молчит без STT."""

import base64
import json
import threading
import urllib.request
from http.server import ThreadingHTTPServer

import dispatcher.serve as serve
from dispatcher.service import Service

SID = "bilet04_call01"
URL = "http://127.0.0.1:18099"


def post(path: str, obj: dict):
    req = urllib.request.Request(
        URL + path, json.dumps(obj).encode(), {"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req) as r:
            return r.status, json.loads(r.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read())


def test_serve_text_roundtrip():
    serve.SVC = Service()
    srv = ThreadingHTTPServer(("127.0.0.1", 18099), serve.H)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    try:
        code, opened = post("/sessions/check", {"scenario_id": SID})
        assert code == 200
        assert post("/sessions/check", {"scenario_id": "нет"})[0] == 404
        code, o = post("/sessions/open", {"scenario_id": SID, "session_id": "t1"})
        assert code == 200 and o["opening"]
        code, r = post("/sessions/final",
                       {"session_id": "t1", "text": "На каком этаже горит?"})
        assert code == 200 and "тринадцат" in r["text"].lower()
        assert set(r) == {"text", "audio_id", "style", "mood"}
        # /rtp с тишиной: endpoint не срабатывает, в канал тишина
        silence = base64.b64encode(b"\xd5" * 320).decode()  # alaw тишина
        code, out = post("/rtp", {"session_id": "t1", "audio_b64": silence})
        assert code == 200 and out == {"audio_b64": ""}
        assert post("/sessions/close", {"session_id": "t1"})[0] == 200
        assert post("/sessions/final",
                    {"session_id": "t1", "text": "алло"})[0] == 404
    finally:
        srv.shutdown()
