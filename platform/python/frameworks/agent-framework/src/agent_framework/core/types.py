"""Shared types for the framework's agent registry."""

from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from pydantic_ai import Agent

from agent_framework.core.personality.base import Personality


@dataclass(frozen=True)
class RegisteredAgent:
    """One agent discovered under an `agents/` folder: its built PydanticAI Agent, a factory for
    its per-request deps, and the auth mode gating access to it. `name` matches the agent's own
    folder name, not necessarily `agent.name` (which is only a Logfire trace label).

    `personality` and `prompt_fields` are optional, purely for introspection (see
    server/routes/introspect.py) - the agent itself only ever reads its own already-composed
    `instructions=[...]`, never these. A consumer that built its prompt from a Personality plus
    other named pieces (e.g. operational_instructions) can hand both back here so a dev console
    can show the individual fields, not just the final joined string. `prompt_fields` is
    deliberately a plain label->text dict, not a new framework type - there's exactly one real
    example of a second prompt piece (operational_instructions) to generalize from so far."""

    name: str
    agent: Agent[Any, Any]
    build_deps: Callable[[Any], Any]
    auth: str = "public"
    personality: Personality | None = None
    prompt_fields: dict[str, str] | None = None
