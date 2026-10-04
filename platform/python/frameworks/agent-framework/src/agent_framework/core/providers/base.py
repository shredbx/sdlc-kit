"""A provider resolver's shape: takes a bare model name and how many times a failed call may be retried (see
retry.py), returns the PydanticAI model."""

from typing import Protocol

from pydantic_ai.models import Model


class ProviderResolver(Protocol):
    def __call__(self, model: str, max_retries: int = ...) -> Model: ...
