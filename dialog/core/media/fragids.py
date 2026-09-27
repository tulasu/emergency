"""Stable fragment identifiers shared with the audio enumerator."""

from __future__ import annotations


def fskey(key: str) -> str:
    """Encode a fact key injectively as one path-safe UTF-8 component."""
    return "k-" + key.encode("utf-8").hex()
