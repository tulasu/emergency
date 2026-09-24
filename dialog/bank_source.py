# -*- coding: utf-8 -*-
"""Immutable bank with atomic swap (AD-8). In-flight calls finish on old.

The bank is owned by traineebox/Postgres; dialog holds an immutable
snapshot. Reload builds the new snapshot in background and swaps one
reference — sessions opened before the swap keep the old object.
"""

from __future__ import annotations

import hashlib
import json
import os
import sys
import threading
from dataclasses import dataclass, field

# Import-time path setup, once: lexical() must not touch sys.path per call.
_DISPATCHER_PATH = os.environ.get(
    "DISPATCHER_PATH", "/home/poezd/work/emergency/dispatcher")
if _DISPATCHER_PATH not in sys.path:
    sys.path.insert(0, _DISPATCHER_PATH)


def digest_of(slots: dict[str, str], questions: dict[str, list[str]]) -> str:
    # Canon: sorted keys AND sorted phrasings — matches traineebox BankDigest
    # so DB-recompute and worker-recompute agree byte-for-byte (spec J).
    canon = json.dumps({"slots": slots,
                        "questions": {k: sorted(v) for k, v in questions.items()}},
                       ensure_ascii=False, sort_keys=True)
    return hashlib.sha256(canon.encode()).hexdigest()[:16]


@dataclass(frozen=True, slots=True)
class Bank:
    version: str
    slots: dict[str, str]  # slot id -> label
    questions: dict[str, list[str]]  # slot -> phrasings
    digest: str = ""

    @staticmethod
    def build(version: str, slots: dict[str, str],
              questions: dict[str, list[str]]) -> "Bank":
        return Bank(version, dict(slots),
                    {k: list(v) for k, v in questions.items()},
                    digest_of(slots, questions))

    @property
    def slot_ids(self) -> set[str]:
        return set(self.slots)

    def lexical(self):
        """LexicalBank over this snapshot (frozen core, no vectors)."""
        from dispatcher.nlu.bank import LexicalBank

        sources = {s: ["bank:" + self.version] * len(q)
                   for s, q in self.questions.items()}
        return LexicalBank.build(self.questions, sources)


@dataclass(slots=True)
class BankHolder:
    """Single mutable cell holding the current immutable Bank."""
    _current: Bank = field(default_factory=lambda: Bank.build("empty", {}, {}))
    failures: int = 0
    _lock: threading.Lock = field(default_factory=threading.Lock,
                                   repr=False, compare=False)

    @property
    def current(self) -> Bank:
        """Locked read: swap never races readers."""
        with self._lock:
            return self._current

    @current.setter
    def current(self, bank: Bank) -> None:
        with self._lock:
            self._current = bank

    def get(self) -> Bank:
        """Locked read for the hot call path."""
        return self.current

    def swap(self, bank: Bank) -> Bank:
        with self._lock:
            old = self._current
            self._current = bank
            self.failures = 0
            return old

    def mark_failure(self) -> bool:
        """3 retries then bank_stale in health (spec I/O matrix)."""
        with self._lock:
            self.failures += 1
            return self.failures >= 3

    @property
    def stale(self) -> bool:
        with self._lock:
            return self.failures >= 3
