"""Best-effort client-side cache for prerecorded audio fragments."""

from __future__ import annotations

import io
import json
import os
import re
import tempfile
import threading
import urllib.parse
import urllib.request
import wave
from collections import OrderedDict
from pathlib import Path, PurePosixPath

_BLOB_CACHE: OrderedDict[str, bytes] = OrderedDict()
_BLOB_CACHE_BYTES = 0
_BLOB_LOCK = threading.Lock()
_DISK_LOCK = threading.Lock()

_BLOB_HASH = re.compile(r"[0-9a-f]{64}\Z")


def configured() -> bool:
    return bool(os.environ.get("AUDIO_URL", "").rstrip("/"))


def _url(path: str) -> str:
    return os.environ["AUDIO_URL"].rstrip("/") + path


def _headers() -> dict[str, str]:
    return {"X-Service-Token": os.environ.get("INTERNAL_SERVICE_TOKEN", "")}


def _cache_dir() -> Path:
    return Path(os.environ.get("AUDIO_CACHE_DIR", "/tmp/dialog-audio"))


def _cache_limit() -> int:
    try:
        return max(0, int(os.environ.get("AUDIO_CACHE_MB", "500"))) * 1024 * 1024
    except ValueError:
        return 500 * 1024 * 1024

def _http_timeout() -> float:
    try:
        return min(0.75, max(0.05, float(os.environ.get("AUDIO_HTTP_TIMEOUT", "0.25"))))
    except ValueError:
        return 0.25


def _valid_hash(blob_hash: object) -> bool:
    return isinstance(blob_hash, str) and _BLOB_HASH.fullmatch(blob_hash) is not None


def _pcm_from_wav(raw: bytes) -> bytes:
    with wave.open(io.BytesIO(raw), "rb") as wav:
        if (wav.getcomptype() != "NONE" or wav.getnchannels() != 1
                or wav.getsampwidth() != 2 or wav.getframerate() != 8000):
            raise ValueError("want PCM WAV: 8kHz mono s16")
        frames = wav.readframes(wav.getnframes())
    if not frames:
        raise ValueError("empty WAV")
    return frames


def _remember(blob_hash: str, pcm: bytes) -> bytes:
    global _BLOB_CACHE_BYTES
    limit = _cache_limit()
    with _BLOB_LOCK:
        prior = _BLOB_CACHE.pop(blob_hash, None)
        if prior is not None:
            _BLOB_CACHE_BYTES -= len(prior)
        if len(pcm) <= limit:
            _BLOB_CACHE[blob_hash] = pcm
            _BLOB_CACHE_BYTES += len(pcm)
        while _BLOB_CACHE_BYTES > limit:
            _, old = _BLOB_CACHE.popitem(last=False)
            _BLOB_CACHE_BYTES -= len(old)
    return pcm


def _from_memory(blob_hash: str) -> bytes | None:
    with _BLOB_LOCK:
        pcm = _BLOB_CACHE.pop(blob_hash, None)
        if pcm is not None:
            _BLOB_CACHE[blob_hash] = pcm
        return pcm


def _prune_disk(root: Path, limit: int, newest: str | None = None) -> None:
    """Evict oldest cached WAVs first; callers hold _DISK_LOCK."""
    files = []
    total = 0
    for path in root.glob("*.wav"):
        if not _valid_hash(path.stem):
            continue
        try:
            stat = path.stat()
        except FileNotFoundError:  # another process may be pruning the same cache
            continue
        files.append((stat.st_mtime_ns, path.stem == newest, path.name, stat.st_size, path))
        total += stat.st_size
    for _, _, _, size, path in sorted(files):
        if total <= limit:
            break
        try:
            path.unlink()
            total -= size
        except FileNotFoundError:
            total -= size


def _read_wav(blob_hash: str) -> bytes | None:
    root = _cache_dir()
    path = root / f"{blob_hash}.wav"
    with _DISK_LOCK:
        _prune_disk(root, _cache_limit())
        try:
            pcm = _pcm_from_wav(path.read_bytes())
            path.touch()  # retained entries become most recently used
            return pcm
        except (EOFError, ValueError, wave.Error):
            try:
                path.unlink()
            except FileNotFoundError:
                pass
        except FileNotFoundError:
            pass
    return None


def _write_wav(blob_hash: str, raw: bytes) -> None:
    root = _cache_dir()
    with _DISK_LOCK:
        root.mkdir(parents=True, exist_ok=True)
        limit = _cache_limit()
        if len(raw) > limit:
            _prune_disk(root, limit)
            return
        fd, name = tempfile.mkstemp(dir=root, prefix=f".{blob_hash}.", suffix=".tmp")
        try:
            with os.fdopen(fd, "wb") as f:
                f.write(raw)
            os.replace(name, root / f"{blob_hash}.wav")
            _prune_disk(root, limit, newest=blob_hash)
        finally:
            try:
                os.unlink(name)
            except FileNotFoundError:
                pass

def _load_blob(blob_hash: str) -> bytes | None:
    if not _valid_hash(blob_hash):
        return None
    hit = _from_memory(blob_hash)
    if hit is not None:
        return hit
    try:
        pcm = _read_wav(blob_hash)
        if pcm is not None:
            return _remember(blob_hash, pcm)
    except OSError:
        pass
    if not configured():
        return None
    try:
        request = urllib.request.Request(
            _url("/v1/blobs/" + urllib.parse.quote(blob_hash, safe="")),
            headers=_headers(),
        )
        with urllib.request.urlopen(request, timeout=_http_timeout()) as response:
            raw = response.read()
        pcm = _pcm_from_wav(raw)
        _write_wav(blob_hash, raw)
        return _remember(blob_hash, pcm)
    except Exception as exc:  # audio is only an acceleration of live TTS
        print(f"[audio] blob {blob_hash} unavailable: {exc}", flush=True)
        return None


def local(fragment_id: str | None, root: Path | None) -> bytes | None:
    """Load a legacy local fragment only when the audio service is disabled."""
    if root is None or not fragment_id:
        return None
    out = bytearray()
    try:
        for fragment in fragment_id.split("+"):
            parts = PurePosixPath(fragment).parts
            if (not parts or PurePosixPath(fragment).is_absolute()
                    or any(part in ("", ".", "..") for part in parts)):
                return None
            out.extend(_pcm_from_wav(root.joinpath(*parts).read_bytes()))
    except (OSError, EOFError, ValueError, wave.Error):
        return None
    return bytes(out) or None


def resolve(fragments: dict[str, str], fragment_id: str | None) -> bytes | None:
    """Resolve a composite fragment ID to PCM, or let the caller use live TTS."""
    if not fragment_id:
        return None
    out = bytearray()
    for part in fragment_id.split("+"):
        blob_hash = fragments.get(part)
        if not blob_hash:
            return None
        pcm = _load_blob(blob_hash)
        if pcm is None:
            return None
        out.extend(pcm)
    return bytes(out)


def manifest(audio: object) -> dict[str, str]:
    """Fetch a ready, matching manifest. Every error leaves the call on live TTS."""
    if not configured() or not isinstance(audio, dict):
        return {}
    ticket_id = audio.get("ticket_id")
    digest = audio.get("digest")
    if not ticket_id or not digest:
        return {}
    try:
        request = urllib.request.Request(
            _url("/v1/tickets/" + urllib.parse.quote(str(ticket_id), safe="") + "/manifest"),
        )
        with urllib.request.urlopen(request, timeout=_http_timeout()) as response:
            payload = json.loads(response.read())
        if payload.get("scenario_digest") != digest:
            print(f"[audio] manifest digest mismatch for {ticket_id}", flush=True)
            return {}
        if payload.get("status") != "ready":
            return {}
        fragments = payload.get("fragments")
        if not isinstance(fragments, dict):
            return {}
        return {
            fragment_id: blob_hash
            for fragment_id, blob_hash in fragments.items()
            if isinstance(fragment_id, str) and _valid_hash(blob_hash)
        }
    except Exception as exc:  # a failed audio service must not reject /sessions/open
        print(f"[audio] manifest unavailable for {ticket_id}: {exc}", flush=True)
        return {}


def prefetch(fragments: dict[str, str]) -> None:
    """Warm the manifest in the background, with the opening fragment first."""
    ordered = sorted(fragments, key=lambda key: (not key.endswith("/opening.wav"), key))

    def load() -> None:
        for key in ordered:
            _load_blob(fragments[key])

    threading.Thread(target=load, name="dialog-audio-prefetch", daemon=True).start()
