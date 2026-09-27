# -*- coding: utf-8 -*-
"""Экспериментальная 1:1 замена LlmArbiter: Qwen3-1.7B QLoRA.

Выбирает одну цифру по logits следующего токена; ноль — честный отказ.
Веса адаптера создаёт tools.train_lora_arbiter; продовый voter не меняется.
"""

from __future__ import annotations

import random
from dataclasses import dataclass, field
from pathlib import Path

from ..data.ontology import Ontology
from .llm_arbiter import _SYSTEM, format_user


@dataclass
class LoraLlmArbiter:
    ontology: Ontology
    base: str | Path
    adapter: str | Path
    seed: int = 0
    calls: int = 0
    failures: int = 0
    refusals: int = 0
    last: str = ""
    model: object = field(init=False, repr=False)
    tokenizer: object = field(init=False, repr=False)
    _rng: random.Random = field(init=False, repr=False)
    _digits: list[int] = field(init=False, repr=False)

    def __post_init__(self) -> None:
        import torch
        from peft import PeftModel
        from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

        self._rng = random.Random(self.seed)
        self.tokenizer = AutoTokenizer.from_pretrained(self.base, local_files_only=True)
        self._digits = [self.tokenizer.encode(str(i), add_special_tokens=False)[0]
                        for i in range(6)]
        base = AutoModelForCausalLM.from_pretrained(
            self.base, local_files_only=True, device_map={"": 0},
            quantization_config=BitsAndBytesConfig(load_in_4bit=True,
                bnb_4bit_quant_type="nf4", bnb_4bit_compute_dtype=torch.bfloat16),
        )
        self.model = PeftModel.from_pretrained(base, self.adapter, is_trainable=False).eval()

    def choose(self, text: str, slots: list[str]) -> str | None:
        import torch

        if not slots:
            return None
        self.calls += 1
        order = slots[:]
        self._rng.shuffle(order)
        labels = [self.ontology.slots[s].label for s in order]
        prompt = [{"role": "system", "content": _SYSTEM},
                  {"role": "user", "content": format_user(text, labels)}]
        try:
            ids = self.tokenizer.apply_chat_template(
                prompt, tokenize=True, add_generation_prompt=True,
                enable_thinking=False, return_tensors="pt",
            ).to(self.model.device)
            with torch.inference_mode():
                logits = self.model(input_ids=ids["input_ids"]).logits[0, -1, self._digits[:len(order) + 1]]
                pick = int(logits.argmax())
            self.last = f"{pick} ({float(torch.softmax(logits.float(), dim=0)[pick]):.2f})"
            if pick == 0:
                self.refusals += 1
                return None
            return order[pick - 1]
        except Exception as exc:  # noqa: BLE001 — другой voter/каскад продолжают звонок
            self.failures += 1
            self.last = f"ошибка {type(exc).__name__}: {exc}"[:200]
            return None
