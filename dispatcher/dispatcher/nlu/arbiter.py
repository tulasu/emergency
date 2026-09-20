# -*- coding: utf-8 -*-
"""
Арбитр серой зоны: заглушка для тестов.

Боевой арбитр — `laya_arbiter.LayaArbiter`. Генеративный вариант на Qwen
проверен и отвергнут (2–3 из 10 — уровень случайного тыка, единственный
соображающий режим стоит секунды, и модель ни разу не выбрала «ни о чём»).
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(slots=True)
class NullArbiter:
    """Заглушка: серая зона всегда становится «не знаю»."""

    def choose(self, text: str, slots: list[str]) -> str | None:
        return None
