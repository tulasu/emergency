"""OpenAI-compatible LLM client: instructor validation + JSON schema."""

from __future__ import annotations

import json
import logging
import os
from dataclasses import dataclass
from typing import Protocol, TypeVar

from pydantic import BaseModel

log = logging.getLogger("ticketgen.llm")

T = TypeVar("T", bound=BaseModel)


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
    return [
        Backend(
            name="default",
            url=url,
            model=os.environ.get("TICKETGEN_LLM_MODEL", ""),
            timeout=60,
        )
    ]


class LLMError(Exception):
    pass


class LLM(Protocol):
    def complete(
        self,
        system: str,
        user: str,
        response_model: type[T],
        *,
        temperature: float = 0.4,
        max_tokens: int = 1024,
    ) -> T: ...


def _v1_base(url: str) -> str:
    base = url.rstrip("/")
    if base.endswith("/v1"):
        return base
    return base + "/v1"


def _response_format(model: type[BaseModel]) -> dict:
    return {
        "type": "json_schema",
        "json_schema": {
            "name": model.__name__,
            "strict": True,
            "schema": model.model_json_schema(),
        },
    }


class LLMClient:
    def __init__(self, backends: list[Backend] | None = None, max_retries: int = 3):
        self.backends = backends or backends_from_env()
        self.max_retries = max_retries
        self._patched: dict[str, object] = {}

    def complete(
        self,
        system: str,
        user: str,
        response_model: type[T],
        *,
        temperature: float = 0.4,
        max_tokens: int = 1024,
    ) -> T:
        last_err: Exception | None = None
        for backend in self.backends:
            try:
                return self._complete_backend(
                    backend,
                    system,
                    user,
                    response_model,
                    temperature=temperature,
                    max_tokens=max_tokens,
                )
            except Exception as e:  # noqa: BLE001
                last_err = e
                log.warning("llm backend %s failed: %s", backend.name, e)
        raise LLMError(f"all backends failed: {last_err}")

    def _client(self, backend: Backend):
        cached = self._patched.get(backend.name)
        if cached is not None:
            return cached
        import instructor
        from openai import OpenAI

        raw = OpenAI(
            base_url=_v1_base(backend.url),
            api_key=os.environ.get("TICKETGEN_LLM_API_KEY", "local"),
            timeout=backend.timeout,
        )
        mode = getattr(instructor.Mode, "JSON_SCHEMA", None) or instructor.Mode.JSON
        patched = instructor.from_openai(raw, mode=mode)
        self._patched[backend.name] = patched
        return patched

    def _complete_backend(
        self,
        backend: Backend,
        system: str,
        user: str,
        response_model: type[T],
        *,
        temperature: float,
        max_tokens: int,
    ) -> T:
        client = self._client(backend)
        return client.chat.completions.create(
            model=backend.model or "local",
            response_model=response_model,
            messages=[
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
            temperature=temperature,
            max_tokens=max_tokens,
            max_retries=self.max_retries,
            extra_body={
                "chat_template_kwargs": {"enable_thinking": False},
                "response_format": _response_format(response_model),
            },
        )
