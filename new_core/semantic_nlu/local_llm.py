# -*- coding: utf-8 -*-
"""
In-process CPU LLM classifier (llama-cpp-python + GGUF).

Без LM Studio / Ollama. Первый запуск скачивает GGUF в кэш Hugging Face.
Тот же JSON-контракт keys, что у trainer LocalLlm / SYSTEM.
"""
from __future__ import annotations

import json
import os
import re
import time
from pathlib import Path

DEFAULT_REPO = "Qwen/Qwen2.5-0.5B-Instruct-GGUF"
DEFAULT_FILE = "qwen2.5-0.5b-instruct-q4_k_m.gguf"

SYSTEM = """Ты классифицируешь реплику оператора службы 112.

Заявитель знает только перечисленные ниже сведения. Определи, о каких из них
спрашивает оператор.

Сведения:
{catalog}

Верни ТОЛЬКО JSON, без пояснений:
{{"keys": ["ключ", ...]}}

Правила:
- если оператор спрашивает о том, чего в списке нет, верни пустой список;
- если в одной реплике несколько вопросов, перечисли все подходящие ключи;
- не выдумывай ключей, которых нет в списке;
- уточнение без полного смысла («какой?», «марка?») без контекста — пустой список;
- ключ про имя/ФИО — только если спрашивают имя/ФИО самого заявителя.
"""


def parse_keys(raw: str, facts: dict) -> list[str]:
    text = re.sub(r"```json|```", "", raw or "").strip()
    m = re.search(r"\{.*\}", text, re.DOTALL)
    if m:
        text = m.group(0)
    try:
        keys = json.loads(text)["keys"]
    except Exception:
        return []
    if not isinstance(keys, list):
        return []
    return [k for k in keys if isinstance(k, str) and k in facts]


def ensure_model(
    repo_id: str | None = None,
    filename: str | None = None,
    local_path: str | None = None,
) -> Path:
    """Путь к GGUF: локальный файл, env SEMANTIC_GGUF, либо download с HF."""
    env = os.environ.get("SEMANTIC_GGUF")
    path = local_path or env
    if path:
        p = Path(path)
        if p.is_file():
            return p.resolve()
        raise FileNotFoundError(f"GGUF not found: {p}")

    repo = repo_id or os.environ.get("SEMANTIC_GGUF_REPO", DEFAULT_REPO)
    fname = filename or os.environ.get("SEMANTIC_GGUF_FILE", DEFAULT_FILE)
    try:
        from huggingface_hub import hf_hub_download
    except ImportError as e:
        raise ImportError(
            "нужен huggingface_hub: pip install -r requirements-llm.txt"
        ) from e
    downloaded = hf_hub_download(repo_id=repo, filename=fname)
    return Path(downloaded)


def llama_available() -> bool:
    try:
        import llama_cpp  # noqa: F401
        return True
    except ImportError:
        return False


class GgufClassifier:
    """Чистый LLM-классификатор фактов. Без lexical fallback."""

    name = "local_llm"

    def __init__(
        self,
        sc,
        gguf_path: Path | None = None,
        n_ctx: int = 2048,
        n_threads: int | None = None,
        max_tokens: int = 120,
    ):
        self.sc = sc
        self.n_ctx = n_ctx
        self.n_threads = n_threads or max(1, (os.cpu_count() or 4))
        self.max_tokens = max_tokens
        self.gguf_path = gguf_path
        self.catalog = "\n".join(
            f"- {k}: {f.questions[0]}" for k, f in sc.facts.items()
        )
        self.system = SYSTEM.format(catalog=self.catalog)
        self._llm = None
        self.load_ms: float = 0.0
        self.last_error: str | None = None

    def load(self) -> float:
        if self._llm is not None:
            return self.load_ms
        if not llama_available():
            raise ImportError(
                "нужен llama-cpp-python: pip install -r requirements-llm.txt"
            )
        from llama_cpp import Llama

        path = self.gguf_path or ensure_model()
        self.gguf_path = path
        t0 = time.perf_counter()
        self._llm = Llama(
            model_path=str(path),
            n_ctx=self.n_ctx,
            n_threads=self.n_threads,
            verbose=False,
        )
        self.load_ms = (time.perf_counter() - t0) * 1000
        return self.load_ms

    def classify(self, text: str) -> tuple[list[str], float, str]:
        """(keys, ms, status) status: ok | error | parse_empty."""
        try:
            self.load()
        except Exception as e:
            self.last_error = str(e)
            return [], 0.0, "error"

        assert self._llm is not None
        user = f"Текущая реплика: {text}"
        t0 = time.perf_counter()
        try:
            # chat API if available
            create = getattr(self._llm, "create_chat_completion", None)
            if create is not None:
                out = create(
                    messages=[
                        {"role": "system", "content": self.system},
                        {"role": "user", "content": user},
                    ],
                    temperature=0.0,
                    max_tokens=self.max_tokens,
                )
                raw = out["choices"][0]["message"]["content"]
            else:
                prompt = (
                    f"<|im_start|>system\n{self.system}<|im_end|>\n"
                    f"<|im_start|>user\n{user}<|im_end|>\n"
                    f"<|im_start|>assistant\n"
                )
                out = self._llm(
                    prompt,
                    temperature=0.0,
                    max_tokens=self.max_tokens,
                    stop=["<|im_end|>", "</s>"],
                )
                raw = out["choices"][0]["text"]
            ms = (time.perf_counter() - t0) * 1000
            keys = parse_keys(raw, self.sc.facts)
            status = "ok" if keys or (raw and "keys" in raw) else "parse_empty"
            # empty keys with valid JSON {"keys":[]} is ok
            if '"keys"' in (raw or "") or "'keys'" in (raw or ""):
                status = "ok"
            return keys, ms, status
        except Exception as e:
            self.last_error = str(e)
            ms = (time.perf_counter() - t0) * 1000
            return [], ms, "error"
