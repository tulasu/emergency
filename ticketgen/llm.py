"""OpenAI-compatible LLM client with ordered backend failover."""

from __future__ import annotations

import json
import logging
import os
import urllib.error
import urllib.request
from dataclasses import dataclass

log = logging.getLogger("ticketgen.llm")


@dataclass(frozen=True)
class Backend:
    name: str
    url: str
    model: str = ""
    priority: int = 0
    timeout: float = 60.0


def backends_from_env() -> list[Backend]:
    raw = os.environ.get("TICKETGEN_LLM_BACKENDS", "").strip()
    if raw:
        items = json.loads(raw)
        backends = [
            Backend(
                name=i.get("name") or i["url"],
                url=i["url"].rstrip("/"),
                model=i.get("model") or "",
                priority=int(i.get("priority") or 0),
                timeout=float(i.get("timeout") or 60),
            )
            for i in items
        ]
        return sorted(backends, key=lambda b: b.priority)
    url = os.environ.get("TICKETGEN_LLM_URL", "http://127.0.0.1:8081").rstrip("/")
    return [Backend(name="default", url=url, model=os.environ.get("TICKETGEN_LLM_MODEL", ""), timeout=60)]


class LLMError(Exception):
    pass


class LLMClient:
    def __init__(self, backends: list[Backend] | None = None, max_retries: int = 3):
        self.backends = backends or backends_from_env()
        self.max_retries = max_retries

    def chat_json(
        self,
        system: str,
        user: str,
        *,
        temperature: float = 0.4,
        max_tokens: int = 1024,
    ) -> dict:
        last_err: Exception | None = None
        attempts = 0
        while attempts < self.max_retries:
            for backend in self.backends:
                attempts += 1
                try:
                    content = self._call(backend, system, user, temperature, max_tokens)
                    return self._parse_json(content)
                except Exception as e:  # noqa: BLE001
                    last_err = e
                    log.warning("llm backend %s failed: %s", backend.name, e)
                    if attempts >= self.max_retries:
                        break
        raise LLMError(f"all backends failed: {last_err}")

    def _call(
        self,
        backend: Backend,
        system: str,
        user: str,
        temperature: float,
        max_tokens: int,
    ) -> str:
        body: dict = {
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
            "temperature": temperature,
            "max_tokens": max_tokens,
            "chat_template_kwargs": {"enable_thinking": False},
        }
        if backend.model:
            body["model"] = backend.model
        req = urllib.request.Request(
            backend.url + "/v1/chat/completions",
            data=json.dumps(body).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=backend.timeout) as resp:
                payload = json.loads(resp.read())
        except urllib.error.HTTPError as e:
            raise LLMError(f"http {e.code}: {e.read()[:200]!r}") from e
        except Exception as e:
            raise LLMError(str(e)) from e
        return payload["choices"][0]["message"]["content"].strip()

    @staticmethod
    def _parse_json(content: str) -> dict:
        text = content.strip()
        if text.startswith("```"):
            lines = text.splitlines()
            lines = lines[1:]
            if lines and lines[-1].strip().startswith("```"):
                lines = lines[:-1]
            text = "\n".join(lines).strip()
        start = text.find("{")
        end = text.rfind("}")
        if start < 0 or end < 0 or end <= start:
            raise LLMError(f"no json object in response: {content[:200]!r}")
        try:
            return json.loads(text[start : end + 1])
        except json.JSONDecodeError as e:
            raise LLMError(f"invalid json: {e}; body={content[:200]!r}") from e
