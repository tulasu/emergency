# -*- coding: utf-8 -*-
"""Dispatcher-pipeline contract: parallel calls, bank swap, drift, lint."""

import json
import threading
import urllib.request
from http.server import ThreadingHTTPServer

import pytest

import dialog.serve as serve
from dialog.bank_source import Bank
from dialog.validator import SnapshotError, validate_snapshot


def make_bank(version="v1"):
    slots = {"t.addr": "адрес", "t.floor": "этаж"}
    questions = {
        "t.addr": ["назовите адрес", "какой адрес происшествия", "где это случилось"],
        "t.floor": ["на каком этаже горит", "какой этаж в огне", "этаж возгорания"],
    }
    return Bank.build(version, slots, questions)


def snap_a():
    return {
        "id": "test_a", "opening": "Алло, горит!", "critical": ["a1"], "mode": "voice",
        "facts": [
            {"key": "a1", "slot": "t.addr",
             "answers": {"plain": "Улица Грина, дом одиннадцать."}},
            {"key": "a2", "slot": "t.floor",
             "answers": {"plain": "На тринадцатом этаже."}},
        ],
    }


def snap_b():
    return {
        "id": "test_b", "opening": "Алло, помощь!", "critical": ["b1"], "mode": "voice",
        "facts": [
            {"key": "b1", "slot": "t.floor",
             "answers": {"plain": "На втором этаже."}},
            {"key": "b2", "slot": "t.addr",
             "answers": {"plain": "Улица Ленина, дом пять."}},
        ],
    }


@pytest.fixture(autouse=True)
def fresh_state():
    serve.HOLDER.swap(make_bank())
    with serve.GUARD:
        serve.SESSIONS.clear()
    yield
    with serve.GUARD:
        serve.SESSIONS.clear()


def test_parallel_calls_no_mixing_and_replay_parity():
    import uuid
    a, b = uuid.uuid4().hex, uuid.uuid4().hex
    serve.open_session(a, snap_a())
    serve.open_session(b, snap_b())
    ra = serve.final(a, "Назовите адрес")["text"]
    rb = serve.final(b, "На каком этаже горит")["text"]
    assert "Грина" in ra, ra
    assert "втором" in rb, rb
    # sessions isolated: b never saw a's question
    assert len(serve.entry(b)["session"].turns) == 1
    # DB-replay == file-replay: fresh session, same utterances, same replies
    c = "0" * 32
    serve.open_session(c, snap_a())
    assert serve.final(c, "Назовите адрес")["text"] == ra
    serve.close(a)
    serve.close(b)
    serve.close(c)


def test_bank_reload_digest_changes_inflight_on_old():
    import uuid
    a = uuid.uuid4().hex
    serve.open_session(a, snap_a())
    old_bank = serve.entry(a)["bank"]
    new_bank = make_bank("v2")
    new_bank.questions["t.addr"].append("точный адрес назовите")
    new_bank = Bank.build("v2", new_bank.slots, new_bank.questions)
    assert new_bank.digest != old_bank.digest
    serve.HOLDER.swap(new_bank)
    assert serve.entry(a)["bank"] is old_bank  # in-flight finishes on old
    b = uuid.uuid4().hex.replace("0", "1")
    serve.open_session(b, snap_a())
    assert serve.entry(b)["bank"] is new_bank
    serve.close(a)
    serve.close(b)


def test_bad_snapshot_rejected():
    import uuid
    drifted = snap_a()
    drifted["facts"][0]["slot"] = "no.such_slot"
    with pytest.raises(SnapshotError):
        validate_snapshot(drifted, make_bank().slot_ids)
    with pytest.raises(SnapshotError):
        serve.open_session(uuid.uuid4().hex, drifted)
    assert serve.SESSIONS == {}  # call never starts


def test_no_session_unknown_uuid():
    with pytest.raises(KeyError):
        serve.final("f" * 32, "алло")


def test_lint_no_unreachable_critical():
    out = serve.lint_scenario(snap_a())
    assert out["id"] == "test_a"
    assert out["unreachable"] == []


def test_serve_http_contract():
    import uuid
    srv = ThreadingHTTPServer(("127.0.0.1", 18199), serve.H)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    try:
        def post(path, obj):
            req = urllib.request.Request(
                "http://127.0.0.1:18199" + path, json.dumps(obj).encode(),
                {"Content-Type": "application/json"})
            try:
                with urllib.request.urlopen(req) as r:
                    return r.status, json.loads(r.read())
            except urllib.error.HTTPError as e:
                return e.code, json.loads(e.read())

        sid = uuid.uuid4().hex
        code, o = post("/sessions/open", {"session_id": sid, "scenario": snap_a()})
        assert code == 200 and o["opening"]
        code, r = post("/sessions/final", {"session_id": sid, "text": "Назовите адрес"})
        assert code == 200 and set(r) == {"text", "audio_id", "style", "mood"}
        assert post("/sessions/final", {"session_id": "e" * 32, "text": "x"})[0] == 404
        bad = snap_a()
        bad["facts"][0]["slot"] = "no.slot"
        assert post("/sessions/open", {"session_id": uuid.uuid4().hex, "scenario": bad})[0] == 400
        assert post("/sessions/check", {})[0] == 404  # removed route stays gone
        assert post("/rtp", {})[0] == 404
        assert post("/sessions/close", {"session_id": sid})[0] == 200
        # reload carries the snapshot: worker swaps without fetch (spec J)
        nb = make_bank("v9")
        code, _ = post("/bank/reload", {"version": "v9", "digest": nb.digest,
                                            "slots": nb.slots, "questions": nb.questions})
        assert code == 202
        for _ in range(100):
            with urllib.request.urlopen("http://127.0.0.1:18199/bank/version") as r:
                ver = json.loads(r.read())
            if ver["digest"] == nb.digest:
                break
            __import__("time").sleep(0.05)
        assert ver["digest"] == nb.digest, ver
        with urllib.request.urlopen("http://127.0.0.1:18199/health") as r:
            assert json.loads(r.read())["ok"] is True
    finally:
        srv.shutdown()


def test_uuid_forms_match_and_digest_gate():
    """AudioSocket hex (no dashes) == traineebox hyphenated UUID (spec A)."""
    import uuid
    call_id = uuid.uuid4()
    serve.open_session(str(call_id), snap_a())  # hyphenated, as traineebox generates
    assert serve.final(call_id.hex, "Назовите адрес")["text"]  # hex, as AudioSocket delivers
    assert serve.norm_sid(call_id.hex) == str(call_id)
    # bank digest mismatch → 400, never a session on a stale bank (spec J)
    import pytest as _pt
    with _pt.raises(Exception):
        serve.open_session(uuid.uuid4().hex, snap_a(),
                           expected_digest="deadbeefdeadbeef")
    serve.close(str(call_id))


def test_audiosocket_routing_and_turns_push():
    """Socket-level: UUID first frame routes, unknown UUID closes,
    close() pushes turns to traineebox (spec R)."""
    import socket
    import struct
    import threading
    import uuid
    from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

    pushed: list[dict] = []

    class TurnsH(BaseHTTPRequestHandler):
        def log_message(self, *a):
            pass

        def do_POST(self):  # noqa: N802
            n = int(self.headers.get("Content-Length", 0))
            pushed.append(json.loads(self.rfile.read(n) or b"{}"))
            raw = b"{}"
            self.send_response(200)
            self.send_header("Content-Length", "1")
            self.end_headers()
            self.wfile.write(b"{}")

    turns_srv = ThreadingHTTPServer(("127.0.0.1", 18298), TurnsH)
    threading.Thread(target=turns_srv.serve_forever, daemon=True).start()
    serve.TRAINEEBOX_URL = "http://127.0.0.1:18298"
    try:
        threading.Thread(target=serve.audiosocket_listener,
                         kwargs={"host": "127.0.0.1", "port": 18299},
                         daemon=True).start()
        __import__("time").sleep(0.2)  # let the listener bind

        sid = uuid.uuid4()
        serve.open_session(str(sid), snap_a())
        serve.final(str(sid), "Назовите адрес")  # one turn to push on close
        c = socket.create_connection(("127.0.0.1", 18299), timeout=5)
        c.sendall(struct.pack(">BH", 0x01, 16) + sid.bytes)  # UUID frame, raw bytes
        c.sendall(struct.pack(">BH", 0x10, 320) + b"\x00" * 320)  # 20ms audio
        c.sendall(struct.pack(">BH", 0x00, 0))  # hangup → close() → turns push
        c.close()
        for _ in range(100):
            if pushed:
                break
            __import__("time").sleep(0.05)
        assert pushed and pushed[0]["turns"], pushed
        assert pushed[0]["turns"][0]["utterance"] == "Назовите адрес"

        # unknown UUID → connection closed, no session created
        c2 = socket.create_connection(("127.0.0.1", 18299), timeout=5)
        c2.sendall(struct.pack(">BH", 0x01, 16) + uuid.uuid4().bytes)
        c2.settimeout(2)
        assert c2.recv(1) == b"", "unknown UUID must close"
        c2.close()
    finally:
        serve.TRAINEEBOX_URL = ""
        turns_srv.shutdown()


def test_bank_digest_parity_fixture():
    """Go/Python digest parity on fixed fixtures (Cyrillic + quotes).

    Mirrors traineebox/internal/dialog digest test byte-for-byte: the same
    slots/questions must hash to the same hex on both sides (spec J).
    """
    from dialog.bank_source import digest_of
    slots = {"addr.street": "Улица, дом", "common.floor": 'Этаж "второй"'}
    questions = {
        "addr.street": ["Назовите адрес", 'Какой "точный" адрес?'],
        "common.floor": ["этаж"],
    }
    assert digest_of(slots, questions) == "5174a6af863e067c"


def test_reload_carries_ontology():
    """Reload swaps bank+ontology together; in-flight keeps old pair (CAP-4)."""
    import uuid
    old_onto = serve.onto()
    try:
        a = uuid.uuid4().hex
        serve.open_session(a, snap_a())
        bank_a, onto_a = serve.entry(a)["bank"], serve.entry(a)["onto"]

        slots2 = {"t.addr": "адрес", "t.floor": "этаж", "t.name": "имя"}
        questions2 = {
            "t.addr": ["назовите адрес", "какой адрес происшествия"],
            "t.floor": ["на каком этаже горит", "какой этаж в огне"],
            "t.name": ["как вас зовут", "назовитесь пожалуйста"],
        }
        onto2 = {"version": "v12", "slots": [
            {"id": "t.addr", "label": "адрес", "kind": "value",
             "aliases": ["адрес"], "fallback": [], "questions": [],
             "urge": "", "disclosure": "volunteered", "since": "0.1"},
            {"id": "t.floor", "label": "этаж", "kind": "value",
             "aliases": ["этаж"], "fallback": [], "questions": [],
             "urge": "", "disclosure": "volunteered", "since": "0.1"},
            {"id": "t.name", "label": "имя", "kind": "value",
             "aliases": ["имя"], "fallback": [], "questions": [],
             "urge": "", "disclosure": "volunteered", "since": "0.1"},
        ], "overrides": {}}
        bank2 = Bank.build("v2", slots2, questions2)
        serve._reload_background("v2", bank2.digest, slots2, questions2,
                                 ontology=onto2)
        assert serve.HOLDER.current.digest == bank2.digest
        assert serve.onto().version == "v12"
        assert set(serve.onto().slots) == {"t.addr", "t.floor", "t.name"}
        assert serve.onto().by_alias["адрес"] == "t.addr"
        # in-flight finishes on the old pair
        assert serve.entry(a)["bank"] is bank_a
        assert serve.entry(a)["onto"] is onto_a
        # new sessions use the new pair; open+lint work with no data files
        b = uuid.uuid4().hex
        serve.open_session(b, snap_a())
        assert serve.entry(b)["bank"].digest == bank2.digest
        assert serve.entry(b)["onto"] is serve.onto()
        assert serve.final(b, "Назовите адрес")["text"]
        out = serve.lint_scenario(snap_a())
        assert out["id"] == "test_a"
        # reload without ontology: bank swaps, ontology kept
        bank3 = Bank.build("v3", slots2, questions2)
        serve._reload_background("v3", bank3.digest, slots2, questions2)
        assert serve.HOLDER.current.digest == bank3.digest
        assert serve.onto().version == "v12"
        # bad kind: old pair kept, failure counted
        f0 = serve.HOLDER.failures
        serve._reload_background("vbad", "", slots2, questions2, ontology={
            "version": "bad", "slots": [{"id": "t.x", "label": "x",
                                         "kind": "nope"}]})
        assert serve.HOLDER.current.digest == bank3.digest
        assert serve.onto().version == "v12"
        assert serve.HOLDER.failures == f0 + 1
        with pytest.raises(ValueError):
            serve.ontology_from_snapshot({"version": "bad", "slots": [
                {"id": "t.x", "label": "x", "kind": "nope"}]})
        serve.close(a)
        serve.close(b)
    finally:
        serve.ONTO = old_onto


def test_empty_boot_contract():
    """Before first reload: fetch raises, open/lint rejected, no crash (matrix).

    NOTE: frozen matrix row 'Lint on empty bank' says 'report'; code rejects
    with SnapshotError (same as open, AD-7 drift). Test pins code behavior;
    matrix needs a one-line human-approved correction.
    """
    import os
    import uuid
    if os.environ.get("TRAINEEBOX_BANK_URL"):
        pytest.skip("TRAINEEBOX_BANK_URL set: boot would fetch, not stay empty")
    with pytest.raises(RuntimeError):
        serve._fetch_bank("boot")
    assert Bank.build("empty", {}, {}).digest == serve.BankHolder().current.digest
    prev = serve.HOLDER.current
    try:
        serve.HOLDER.swap(Bank.build("empty", {}, {}))
        with pytest.raises(SnapshotError):
            serve.open_session(uuid.uuid4().hex, snap_a())
        assert serve.SESSIONS == {}  # call never starts
        with pytest.raises(SnapshotError):
            serve.lint_scenario(snap_a())
    finally:
        serve.HOLDER.swap(prev)


def test_bank_reload_http_swaps_onto():
    """POST /bank/reload carrying ontology swaps serve.onto() (HTTP-level).

    Direct-call reload tests would miss a dropped forwarding arg; this goes
    through the HTTP route like traineebox fan-out does.
    """
    old_bank = serve.HOLDER.current
    old_onto = serve.onto()
    srv = ThreadingHTTPServer(("127.0.0.1", 18301), serve.H)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    try:
        def post(path, obj):
            req = urllib.request.Request(
                "http://127.0.0.1:18301" + path, json.dumps(obj).encode(),
                {"Content-Type": "application/json"})
            with urllib.request.urlopen(req) as r:
                return r.status, json.loads(r.read())

        nb = make_bank("vhttp")
        onto_payload = {"version": "vhttp-o", "slots": [
            {"id": "t.addr", "label": "адрес", "aliases": ["адрес"]},
            {"id": "t.floor", "label": "этаж", "aliases": ["этаж"]},
        ], "overrides": {}}
        code, _ = post("/bank/reload", {"version": "vhttp", "digest": nb.digest,
                                        "slots": nb.slots, "questions": nb.questions,
                                        "ontology": onto_payload})
        assert code == 202
        for _ in range(100):
            with urllib.request.urlopen("http://127.0.0.1:18301/bank/version") as r:
                ver = json.loads(r.read())
            if ver["digest"] == nb.digest:
                break
            __import__("time").sleep(0.05)
        assert ver["digest"] == nb.digest, ver
        assert serve.onto().version == "vhttp-o"
        assert set(serve.onto().slots) == {"t.addr", "t.floor"}
        assert serve.onto().by_alias["адрес"] == "t.addr"
    finally:
        srv.shutdown()
        serve.HOLDER.swap(old_bank)
        serve.ONTO = old_onto


def test_prod_ensemble_uses_session_onto(monkeypatch):
    """DIALOG_PROD=1: dual voters receive the session's onto (not a stale global)."""
    import uuid
    seen: dict = {}
    old_bank = serve.HOLDER.current
    old_onto = serve.onto()
    try:
        slots2 = {"t.addr": "адрес", "t.floor": "этаж"}
        questions2 = {
            "t.addr": ["назовите адрес", "какой адрес происшествия"],
            "t.floor": ["на каком этаже горит", "какой этаж в огне"],
        }
        onto2 = {"version": "vprod", "slots": [
            {"id": "t.addr", "label": "адрес", "aliases": ["адрес"]},
            {"id": "t.floor", "label": "этаж", "aliases": ["этаж"]},
        ], "overrides": {}}
        bank2 = Bank.build("vprod", slots2, questions2)
        serve._reload_background("vprod", bank2.digest, slots2, questions2,
                                 ontology=onto2)
        assert serve.onto().version == "vprod"

        class FakeLaya:
            def __init__(self, *a, **k):
                seen["laya"] = k.get("ontology", a[0] if a else None)

        class FakeLlm:
            def __init__(self, *a, **k):
                seen["llm"] = k.get("ontology", a[0] if a else None)

        class FakeImpro:
            def __init__(self, *a, **k):
                pass

        def fake_ensemble(cascade, voters, rule="majority"):
            seen["voters"] = voters
            return cascade

        monkeypatch.setenv("DIALOG_PROD", "1")
        monkeypatch.setattr("dialog.core.nlu.laya_arbiter.LayaArbiter", FakeLaya)
        monkeypatch.setattr("dialog.core.nlu.llm_arbiter.LlmArbiter", FakeLlm)
        monkeypatch.setattr("dialog.core.dialog.improv.Improviser", FakeImpro)
        monkeypatch.setattr("dialog.core.nlu.ensemble.Ensemble", fake_ensemble)

        sid = uuid.uuid4().hex
        serve.open_session(sid, snap_a())
        e = serve.entry(sid)
        assert e["onto"] is serve.onto()
        assert seen["laya"] is e["onto"], seen
        assert seen["llm"] is e["onto"], seen
        assert len(seen["voters"]) == 2
        serve.close(sid)
    finally:
        serve.HOLDER.swap(old_bank)
        serve.ONTO = old_onto
