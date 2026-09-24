# -*- coding: utf-8 -*-
"""
Голос маленькой LLM (llama-server, tools/llama_server.sh) при выборе слота.

Тот же контракт, что у LayaArbiter: choose(text, slots) -> slot | None.
Ответ зажат грамматикой до одной цифры — ни генерации, ни парсинга, и
рассуждения выключены: с ними Qwen3 думает секунды (README, «Про арбитра»).
Варианты перемешиваются на каждом вызове: без этого маленькая модель
отвечает «1» независимо от вопроса, и замер выглядит лучше, чем есть.
Нулевой вариант — «ни о чём»: ложный факт хуже честного «не знаю».
"""

from __future__ import annotations

import json
import os
import random
import urllib.request
from dataclasses import dataclass, field

from ..data.ontology import Ontology

LLM_URL = os.environ.get("DISPATCHER_LLM_URL", "http://127.0.0.1:8081")

_SYSTEM = (
    "Ты помогаешь тренажёру службы 112. Оператор задал вопрос заявителю. "
    "Определи, о каком сведении спрашивает оператор. "
    "Ответь одной цифрой — номером варианта. Если ни один не подходит, ответь 0."
)


@dataclass
class LlmArbiter:
    ontology: Ontology
    url: str = LLM_URL
    timeout: float = 2.0
    seed: int = 0
    calls: int = 0
    failures: int = 0
    _rng: random.Random = field(init=False, repr=False)

    def __post_init__(self) -> None:
        self._rng = random.Random(self.seed)

    def _label(self, slot: str) -> str:
        s = self.ontology.slots.get(slot)
        return s.label if s else slot

    def choose(self, text: str, slots: list[str]) -> str | None:
        if not slots:
            return None
        self.calls += 1
        order = slots[:]
        self._rng.shuffle(order)
        options = "\n".join(f"{i}. {self._label(s)}" for i, s in enumerate(order, 1))
        user = (
            f"Вопрос оператора: «{text}»\n\nВарианты:\n0. ни о чём из перечисленного\n"
            f"{options}\n\nНомер варианта:"
        )
        digits = "".join(str(i) for i in range(len(order) + 1))
        body = {
            "messages": [{"role": "system", "content": _SYSTEM},
                         {"role": "user", "content": user}],
            "grammar": f"root ::= [{digits}]",
            "max_tokens": 1,
            "temperature": 0,
            "chat_template_kwargs": {"enable_thinking": False},
        }
        try:
            req = urllib.request.Request(
                self.url + "/v1/chat/completions",
                data=json.dumps(body).encode(),
                headers={"Content-Type": "application/json"},
            )
            with urllib.request.urlopen(req, timeout=self.timeout) as r:
                out = json.loads(r.read())["choices"][0]["message"]["content"].strip()
            n = int(out[:1])
        except Exception:  # noqa: BLE001 — сервер лёг или ответил мусором: голоса нет
            self.failures += 1
            return None
        return order[n - 1] if 1 <= n <= len(order) else None
