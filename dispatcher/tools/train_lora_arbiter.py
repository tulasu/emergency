# -*- coding: utf-8 -*-
"""QLoRA одного выбора 0..5 для контракта LlmArbiter.

Данные bench.rerank_data происходят только из scenario train: топ-5 банка
без своего сценария; отсутствие верного слота даёт метку 0. Blind/live не читаем.
"""

from __future__ import annotations

import argparse
import json
import random
from pathlib import Path

import torch

from peft import LoraConfig, get_peft_model
from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

from bench.holdout import heldout_texts, normalized
from dispatcher.data.loader import DATA
from dispatcher.data.ontology import Ontology
from dispatcher.nlu.llm_arbiter import _SYSTEM, format_user


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--base", default=str(DATA / "build" / "qwen3-1.7b"))
    ap.add_argument("--data", default=str(DATA / "build" / "rerank.jsonl"))
    ap.add_argument("--out", default=str(DATA / "build" / "lora-arbiter-clean"))
    ap.add_argument("--epochs", type=int, default=1)
    ap.add_argument("--batch", type=int, default=1)
    ap.add_argument("--accum", type=int, default=8)
    args = ap.parse_args()

    ontology = Ontology.load()
    tokenizer = AutoTokenizer.from_pretrained(args.base, local_files_only=True)
    digit = [tokenizer.encode(str(i), add_special_tokens=False)[0] for i in range(6)]
    rng = random.Random(0)
    train, held = [], []
    excluded = heldout_texts()
    skipped = 0
    with open(args.data, encoding="utf-8") as stream:
        for index, line in enumerate(stream):
            row = json.loads(line)
            if normalized(row["text"]) in excluded:
                skipped += 1
                continue
            order = list(range(len(row["cands"])))
            rng.shuffle(order)
            slots = [row["cands"][j] for j in order]
            target = 0 if row["label"] is None else order.index(row["label"]) + 1
            messages = [{"role": "system", "content": _SYSTEM},
                        {"role": "user", "content": format_user(
                            row["text"], [ontology.slots[s].label for s in slots])}]
            prefix = tokenizer.apply_chat_template(
                messages, tokenize=True, add_generation_prompt=True,
                enable_thinking=False)["input_ids"][-255:]
            (held if index % 10 == 0 else train).append((prefix, target))
    if not train or not any(label == 0 for _, label in train):
        raise ValueError("нет обучающих отказов")
    print(f"train {len(train)}; held {len(held)}; train отказов "
          f"{sum(y == 0 for _, y in train)}", flush=True)
    model = AutoModelForCausalLM.from_pretrained(
        args.base, local_files_only=True, device_map={"": 0},
        quantization_config=BitsAndBytesConfig(load_in_4bit=True,
            bnb_4bit_quant_type="nf4", bnb_4bit_compute_dtype=torch.bfloat16),
    )
    model.config.use_cache = False
    # prepare_model_for_kbit_training переводит весь embedding в fp32 и
    # запрашивает ещё >1 ГиБ на RTX 4050 рядом с работающим llama-server.
    # Оставляем замороженные bf16-параметры, обучаем только LoRA.
    for param in model.parameters():
        param.requires_grad_(False)
    model.gradient_checkpointing_enable()
    model.enable_input_require_grads()
    model = get_peft_model(model, LoraConfig(
        r=8, lora_alpha=16, lora_dropout=0.05, bias="none", task_type="CAUSAL_LM",
        target_modules=["q_proj", "v_proj"],
    ))
    optimizer = torch.optim.AdamW((p for p in model.parameters() if p.requires_grad), lr=2e-4)

    def batch_data(rows):
        length = max(len(ids) + 1 for ids, _ in rows)
        xs, ys, mask = [], [], []
        for ids, target in rows:
            pad = length - len(ids) - 1
            xs.append(ids + [digit[target]] + [tokenizer.pad_token_id] * pad)
            ys.append([-100] * len(ids) + [digit[target]] + [-100] * pad)
            mask.append([1] * (len(ids) + 1) + [0] * pad)
        return (torch.tensor(xs, device="cuda"), torch.tensor(ys, device="cuda"),
                torch.tensor(mask, device="cuda"))

    model.train()
    for epoch in range(args.epochs):
        rng.shuffle(train)
        optimizer.zero_grad(set_to_none=True)
        total = 0.0
        for start in range(0, len(train), args.batch):
            xs, ys, mask = batch_data(train[start:start + args.batch])
            with torch.autocast("cuda", dtype=torch.bfloat16):
                loss = model(input_ids=xs, attention_mask=mask, labels=ys).loss
            (loss / args.accum).backward()
            total += loss.item() * xs.shape[0]
            if ((start // args.batch + 1) % args.accum == 0 or start + args.batch >= len(train)):
                optimizer.step()
                optimizer.zero_grad(set_to_none=True)
            if start % 400 == 0:
                print(f"epoch {epoch + 1}: {start}/{len(train)} loss {loss.item():.4f}", flush=True)
        print(f"epoch {epoch + 1}: loss {total / len(train):.4f}", flush=True)
    model.eval()
    correct = {"present": 0, "absent": 0}
    count = {"present": 0, "absent": 0}
    with torch.inference_mode():
        for prefix, target in held:
            ids = torch.tensor([prefix], device="cuda")
            logits = model(input_ids=ids).logits[0, -1, digit]
            kind = "absent" if target == 0 else "present"
            count[kind] += 1
            correct[kind] += int(logits.argmax() == target)
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    model.save_pretrained(out)
    tokenizer.save_pretrained(out)
    validation = {k: f"{correct[k]}/{count[k]}" for k in count}
    (out / "training.json").write_text(json.dumps({
        "base": args.base, "data": args.data, "epochs": args.epochs,
        "train": len(train), "held": len(held), "validation": validation,
        "excluded_holdout_phrases": skipped,
    }, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"adapter {out}; validation {validation}", flush=True)


if __name__ == "__main__":
    main()
