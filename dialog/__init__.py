# -*- coding: utf-8 -*-
"""Stateless snapshot executor (dialog), owned by traineebox.

Owner→executor arrows only: traineebox→dialog (open/lint/reload),
dialog→traineebox (turns/events on close). Wav stays on dialog volume.

Frozen ML core is vendored 1:1 in dialog.core (e5-small-tuned, torch/cuda,
GigaAM, majority-ensemble, IMPROV=1 grounded, dual voters laya 0.5 + LLM,
Qwen3-4B 3s, k=5, tuned cascade thresholds). This package adds only the
stateless shell: snapshot validation (400 on drift), per-session locks +
UUID routing, immutable bank with atomic swap, service-token guard.
"""

from .bank_source import Bank
from .serve import main
from .validator import validate_snapshot

__all__ = ["Bank", "main", "validate_snapshot"]
