# -*- coding: utf-8 -*-
"""Исключать точные нормализованные blind/live-совпадения из train.

Тексты отложенных наборов используются только как стоп-лист, не как пары.
"""

from __future__ import annotations

import re

from dispatcher.data.loader import DATA


_PUNCT = re.compile(r"[^\w\s]", re.UNICODE)


def normalized(text: str) -> str:
    return " ".join(_PUNCT.sub(" ", text.lower().replace("ё", "е")).split())


def heldout_texts() -> set[str]:
    excluded: set[str] = set()
    for path in sorted((DATA / "blind").glob("blind_*.txt")):
        for line in path.read_text(encoding="utf-8").splitlines():
            line = line.strip().lstrip("-•*0123456789. ").strip()
            if line and not line.startswith("#"):
                excluded.add(normalized(line))
    for line in (DATA / "gold" / "live_calls.tsv").read_text(encoding="utf-8").splitlines():
        if line.strip() and not line.startswith("#"):
            excluded.add(normalized(line.split("\t")[1]))
    return excluded
