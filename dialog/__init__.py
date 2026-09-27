# -*- coding: utf-8 -*-
"""Stateless snapshot executor (dialog), owned by traineebox."""
from __future__ import annotations

__all__ = ["Bank", "digest_of", "main", "validate_snapshot"]


def __getattr__(name: str):
    if name in {"Bank", "digest_of"}:
        from .bank_source import Bank, digest_of
        return {"Bank": Bank, "digest_of": digest_of}[name]
    if name == "main":
        from .serve import main
        return main
    if name == "validate_snapshot":
        from .validator import validate_snapshot
        return validate_snapshot
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")
