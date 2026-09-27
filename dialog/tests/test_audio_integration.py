# -*- coding: utf-8 -*-
import io
import json
import socket
import struct
import threading
import uuid
import wave
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from types import SimpleNamespace

from dialog.core.data.ontology import Ontology
from dialog.core.dialog.render import Renderer
from dialog.core.media import audio_cache
from dialog.core.media.asterisk import Call, MediaConfig
from dialog.core.media.fragids import fskey
from dialog.core.types import Decision, Fact, Profile, Reply, Scenario, Style


def _wav(pcm: bytes) -> bytes:
    raw = io.BytesIO()
    with wave.open(raw, "wb") as out:
        out.setnchannels(1)
        out.setsampwidth(2)
        out.setframerate(8000)
        out.writeframes(pcm)
    return raw.getvalue()


def _clear_memory_cache() -> None:
    with audio_cache._BLOB_LOCK:
        audio_cache._BLOB_CACHE.clear()
        audio_cache._BLOB_CACHE_BYTES = 0


def test_fskey_and_renderer_fact_fragments_are_deterministic():
    key = "caller#2/address"
    scenario = Scenario(
        id="fire", facts={key: Fact(key, "addr", {"plain": "Адрес."})},
        critical=(), profile=Profile(), opening="Алло.", meta={},
    )
    reply = Renderer(scenario, onto=Ontology("0.1", {}, {}, {})).say(Decision(reveal=[key]))
    assert fskey(key) == "k-63616c6c657223322f61646472657373"
    assert reply.audio_id == f"a/fire/{fskey(key)}/plain.wav"


def test_fskey_distinguishes_punctuation_and_utf8_fact_keys():
    keys = ("addr#2", "addr_2", "addr/2", "k-616464722332", "адрес#2")
    assert fskey("addr#2") == "k-616464722332"
    assert len({fskey(key) for key in keys}) == len(keys)
    assert all("/" not in fskey(key) and "#" not in fskey(key) for key in keys)


def test_renderer_uses_plain_fragment_for_missing_or_empty_styles():
    key = "address"
    scenario = Scenario(
        id="fire", facts={key: Fact(key, "addr", {
            "plain": "Это адрес.", "confirm": "", "short": "Кратко.",
        })},
        critical=(), profile=Profile(), opening="Алло.", meta={},
    )
    renderer = Renderer(scenario, onto=Ontology("0.1", {}, {}, {}))
    for style in (Style.CONFIRM, Style.CORRECT):
        reply = renderer.say(Decision(reveal=[key], style=style))
        assert reply.text == "Это адрес."
        assert reply.audio_id == f"a/fire/{fskey(key)}/plain.wav"
    reply = renderer.say(Decision(reveal=[key], style=Style.SHORT))
    assert reply.text == "Кратко."
    assert reply.audio_id == f"a/fire/{fskey(key)}/short.wav"


def test_manifest_and_blob_cache_use_public_manifest_and_tokenized_blob(tmp_path, monkeypatch):
    pcm = b"\x01\x00" * 160
    wav = _wav(pcm)
    blob_hash = "a" * 64
    seen: list[tuple[str, str]] = []

    class AudioH(BaseHTTPRequestHandler):
        def log_message(self, *args):
            pass

        def do_GET(self):  # noqa: N802
            seen.append((self.path, self.headers.get("X-Service-Token", "")))
            if self.path.startswith("/v1/tickets/"):
                ticket = self.path.split("/")[3]
                payload = {
                    "scenario_digest": "digest" if ticket != "stale" else "old-digest",
                    "status": "pending" if ticket == "pending" else "ready",
                    "fragments": {"a/fire/opening.wav": blob_hash, "bad": "not-a-hash"},
                }
                raw = json.dumps(payload).encode()
                self.send_response(200)
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)
                return
            if self.path == f"/v1/blobs/{blob_hash}":
                self.send_response(200)
                self.send_header("Content-Length", str(len(wav)))
                self.end_headers()
                self.wfile.write(wav)
                return
            self.send_error(404)

    server = ThreadingHTTPServer(("127.0.0.1", 0), AudioH)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    monkeypatch.setenv("AUDIO_URL", f"http://127.0.0.1:{server.server_port}")
    monkeypatch.setenv("INTERNAL_SERVICE_TOKEN", "shared-token")
    monkeypatch.setenv("AUDIO_CACHE_DIR", str(tmp_path))
    _clear_memory_cache()
    try:
        fragments = audio_cache.manifest({"ticket_id": "ticket", "digest": "digest"})
        assert fragments == {"a/fire/opening.wav": blob_hash}
        assert audio_cache.manifest({"ticket_id": "pending", "digest": "digest"}) == {}
        assert audio_cache.manifest({"ticket_id": "stale", "digest": "digest"}) == {}
        (tmp_path / f"{blob_hash}.wav").write_bytes(b"corrupt")
        assert audio_cache.resolve(fragments, "a/fire/opening.wav") == pcm
        assert (tmp_path / f"{blob_hash}.wav").read_bytes() == wav
        assert seen == [
            ("/v1/tickets/ticket/manifest", ""),
            ("/v1/tickets/pending/manifest", ""),
            ("/v1/tickets/stale/manifest", ""),
            (f"/v1/blobs/{blob_hash}", "shared-token"),
        ]
    finally:
        server.shutdown()
        _clear_memory_cache()


def test_prefetch_evicts_old_wav_blobs_at_disk_limit(tmp_path, monkeypatch):
    first_hash, second_hash = "f" * 64, "a" * 64
    first_pcm, second_pcm = b"\x01\x00" * 325_000, b"\x02\x00" * 325_000
    wavs = {first_hash: _wav(first_pcm), second_hash: _wav(second_pcm)}
    fetched = []

    def fetch(request, timeout):
        blob_hash = request.full_url.rsplit("/", 1)[-1]
        fetched.append(blob_hash)
        return io.BytesIO(wavs[blob_hash])

    monkeypatch.setenv("AUDIO_URL", "http://audio.invalid")
    monkeypatch.setenv("AUDIO_CACHE_DIR", str(tmp_path))
    monkeypatch.setenv("AUDIO_CACHE_MB", "1")
    monkeypatch.setattr(audio_cache.urllib.request, "urlopen", fetch)
    original_load = audio_cache._load_blob
    results = []
    done = threading.Event()

    def load(blob_hash):
        results.append(original_load(blob_hash))
        if len(results) == 2:
            done.set()

    monkeypatch.setattr(audio_cache, "_load_blob", load)
    _clear_memory_cache()
    try:
        audio_cache.prefetch({
            "a/fire/opening.wav": first_hash,
            "a/fire/other/plain.wav": second_hash,
        })
        assert done.wait(3)
        assert fetched == [first_hash, second_hash]
        assert results == [first_pcm, second_pcm]
        assert not (tmp_path / f"{first_hash}.wav").exists()
        assert (tmp_path / f"{second_hash}.wav").read_bytes() == wavs[second_hash]
        assert sum(path.stat().st_size for path in tmp_path.glob("*.wav")) <= 1024 * 1024
    finally:
        _clear_memory_cache()


def test_zero_disk_limit_removes_existing_blobs_and_skips_new_writes(tmp_path, monkeypatch):
    old_hash, new_hash = "b" * 64, "c" * 64
    (tmp_path / f"{old_hash}.wav").write_bytes(_wav(b"\x01\x00" * 160))
    pcm = b"\x02\x00" * 160
    fetched = []

    def fetch(request, timeout):
        fetched.append(request.full_url)
        return io.BytesIO(_wav(pcm))

    monkeypatch.setenv("AUDIO_URL", "http://audio.invalid")
    monkeypatch.setenv("AUDIO_CACHE_DIR", str(tmp_path))
    monkeypatch.setenv("AUDIO_CACHE_MB", "0")
    monkeypatch.setattr(audio_cache.urllib.request, "urlopen", fetch)
    _clear_memory_cache()
    try:
        assert audio_cache._load_blob(new_hash) == pcm
        assert list(tmp_path.glob("*.wav")) == []
        assert audio_cache._load_blob(new_hash) == pcm
        assert len(fetched) == 2
    finally:
        _clear_memory_cache()


def test_prefetch_prioritizes_opening(monkeypatch):
    opening_hash = "b" * 64
    other_hash = "c" * 64
    loaded: list[str] = []
    done = threading.Event()

    def load(blob_hash):
        loaded.append(blob_hash)
        if len(loaded) == 2:
            done.set()
        return b"pcm"

    monkeypatch.setattr(audio_cache, "_load_blob", load)
    audio_cache.prefetch({"common/plain.wav": other_hash,
                          "a/fire/opening.wav": opening_hash})
    assert done.wait(1)
    assert loaded == [opening_hash, other_hash]


def test_audio_failures_fall_back_to_live_tts_and_mark_turn_generated(tmp_path, monkeypatch):
    monkeypatch.setenv("AUDIO_URL", "http://127.0.0.1:1")
    assert audio_cache.manifest({"ticket_id": "ticket", "digest": "digest"}) == {}

    reply = Reply("Живой ответ.", "a/fire/fact/plain.wav")
    turn = SimpleNamespace(
        understanding=SimpleNamespace(slots={}, keys=[], act="ask", score=1,
                                      source="test", latency_ms=0),
        decision=SimpleNamespace(reveal=[], style="plain"), reply=reply,
    )

    class Session:
        scenario = SimpleNamespace(id="fire")
        turns = [turn]
        cascade = None

        def on_final(self, text):
            return reply

    sent: list[bytes] = []
    call = Call(Session(), SimpleNamespace(), sent.append,
                MediaConfig(log_root=Path(tmp_path)), tts=lambda text: b"\x02\x00",
                audio={})
    call._answer("назовите адрес")
    log = next(Path(tmp_path).glob("*/turns.jsonl"))
    assert json.loads(log.read_text().strip())["generated"] is True
    assert sent == [b"\x02\x00"]


def test_empty_audio_url_keeps_optional_local_audio(tmp_path, monkeypatch):
    monkeypatch.delenv("AUDIO_URL", raising=False)
    pcm = b"\x03\x00" * 160
    audio_root = tmp_path / "legacy"
    path = audio_root / "a" / "fire" / "fact" / "plain.wav"
    path.parent.mkdir(parents=True)
    path.write_bytes(_wav(pcm))
    reply = Reply("Локальный ответ.", "a/fire/fact/plain.wav")
    turn = SimpleNamespace(
        understanding=SimpleNamespace(slots={}, keys=[], act="ask", score=1,
                                      source="test", latency_ms=0),
        decision=SimpleNamespace(reveal=[], style="plain"), reply=reply,
    )

    class Session:
        scenario = SimpleNamespace(id="fire")
        turns = [turn]
        cascade = None

        def on_final(self, text):
            return reply

    sent: list[bytes] = []
    call = Call(Session(), SimpleNamespace(), sent.append,
                MediaConfig(audio_root=audio_root), audio={})
    call._answer("назовите адрес")
    assert sent == [pcm]


def test_configured_opening_is_queued_before_microphone_loop(monkeypatch):
    from dialog import audiosocket

    opening = b"\x04\x00" * 160
    monkeypatch.setenv("AUDIO_URL", "http://audio.invalid")
    monkeypatch.setattr(audiosocket, "resolve_audio", lambda fragments, fragment_id: opening)
    monkeypatch.setattr(audiosocket, "entry", lambda session_id: {
        "session": SimpleNamespace(scenario=SimpleNamespace(id="fire"),
                                   opening=lambda: "Алло."),
        "lock": threading.Lock(), "audio": {},
    })
    monkeypatch.setattr(audiosocket, "close", lambda session_id: None)
    queued: list[bytes] = []

    def writer(call, conn, out, done, stats, lock):
        with lock:
            queued.append(bytes(out))
        done.set()

    monkeypatch.setattr(audiosocket, "_writer_thread", writer)
    client, server = socket.socketpair()
    worker = threading.Thread(target=audiosocket._pump_one_call, args=(server, None))
    worker.start()
    call_id = uuid.uuid4()
    client.sendall(struct.pack(">BH", 0x01, 16) + call_id.bytes + struct.pack(">BH", 0, 0))
    worker.join(timeout=1)
    client.close()
    assert not worker.is_alive()
    assert queued == [opening]
