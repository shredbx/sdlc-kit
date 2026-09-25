"""A provider resolver's shape: takes a bare model name, returns the PydanticAI model string."""

from typing import Protocol


class ProviderResolver(Protocol):
    def __call__(self, model: str) -> str: ...
