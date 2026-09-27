# -*- coding: utf-8 -*-
"""Общее для laya-слоёв: веса, роутер, отказ от ответа."""

from __future__ import annotations

import os
from pathlib import Path
from typing import Any

# Веса laya лежат рядом с остальными моделями. Без этого её загрузчик пойдёт
# в сеть за тем, что уже скачано.
MODELS_DIR = Path(__file__).resolve().parents[2] / "models"
os.environ.setdefault("HF_HOME", str(MODELS_DIR))

NOTHING = "ни о чём из перечисленного"
FIELD = "вопрос оператора"
QUESTION = "О чём спрашивает оператор службы 112?"


def make_router(device: str = "cpu") -> Any:
    """Только мультиязычный чекпоинт: реплики оператора по-русски."""
    from laya import Router

    return Router(device=device, default="multilingual")


def refused(pick: str | None, confidence: float, minimum: float) -> bool:
    """Модель отказалась («ни о чём») или не уверена — отвечать нельзя."""
    return pick == NOTHING or confidence < minimum
