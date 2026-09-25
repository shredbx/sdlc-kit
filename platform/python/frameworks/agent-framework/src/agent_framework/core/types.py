"""Shared types for the framework's agent registry."""

from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from pydantic_ai import Agent


@dataclass(frozen=True)
class RegisteredAgent:
    """One agent discovered under an `agents/` folder: its built PydanticAI Agent, a factory for
    its per-request deps, and the auth mode gating access to it. `name` matches the agent's own
    folder name, not necessarily `agent.name` (which is only a Logfire trace label)."""

    name: str
    agent: Agent[Any, Any]
    build_deps: Callable[[Any], Any]
    auth: str = "public"
