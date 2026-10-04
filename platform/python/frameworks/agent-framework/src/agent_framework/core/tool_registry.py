"""ToolEntry is a consumer's one declaration per tool - not just debug/introspection metadata.
`build_tools()` is the single place that turns a whole registry into the real PydanticAI tool list,
so a consumer's agents/chat/config.py no longer hand-writes a second, parallel `tools=[Tool(fn),
...]` list that has to be kept in sync with this one by hand (the "registered twice" problem: see
docs/plans/2026-09-28-chat-agent-system-review-and-delivery-plan.md S2, and the follow-up design
doc's §5). debug_model.py's `tool:name {json}` commands still run through the agent's own
registered tools (see AgentInfo.function_tools) - this registry supplies the optional per-tool card
mapping, plus metadata for introspection (see server/routes/introspect.py). `default_to_reply`
JSON-dumps a result as plain text — works for any tool with no bespoke code, at the cost of not
being card-shaped."""

import json
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from pydantic import BaseModel
from pydantic_ai import Tool

from agent_framework.core.cards import AgentReply
from agent_framework.core.knowledge.base import KnowledgeStore


@dataclass
class ToolEntry:
    # The tool's real callable. None for an entry that's pure debug/introspection metadata for a
    # tool registered some other way (e.g. directly on the Agent in a test fixture) - build_tools()
    # skips those, since they were never meant to produce a real Tool from this registry.
    fn: Callable[..., Any] | None = None
    # Actually respected by build_tools() - a disabled tool is left out of the real tools=[...]
    # list entirely, not just hidden from the model by prose. Sourced from the tool's own YAML
    # (core.tool_config.ToolConfig.enabled) via a consumer's registry.py.
    enabled: bool = True
    to_reply: Callable[[Any], AgentReply] | None = None
    # How the tool was built, for a dev console to show, not anything the framework branches on:
    # "function" (hand-written, arbitrary logic - the default, and everything today), "api_config"
    # (built by api.tool_factory.build_api_tool from a declarative ApiToolConfig - no such tool
    # exists yet, the mechanism just already does), "knowledge" (backed by core.knowledge's
    # KnowledgeStore + search, like search_faq).
    kind: str = "function"
    # Set only for kind="knowledge" tools - the same store instance the tool itself searches, so
    # GET /tools/{name}/knowledge (see server/routes/knowledge.py) can browse its real content.
    knowledge_store: KnowledgeStore | None = None
    # The tool's real return type, when it's a Pydantic model - PydanticAI's own ToolDefinition
    # never carries this (a model doesn't need to know a tool's output shape to call it), so a dev
    # console has no other way to see it. None for a tool with no clean structured output (e.g.
    # agent_guide returns list[dict] | str) - that's honest information, not a gap to paper over.
    output_schema: type[BaseModel] | None = None
    # When/how to use this tool and how to phrase a reply from its result - this becomes the real
    # Tool's `description` in build_tools(), the exact text PydanticAI sends the model alongside
    # the tool's schema. Sourced from the tool's own YAML (core.tool_config), so a console can edit
    # it as data. None for a tool whose usage is self-evident or has no special reply-phrasing rules
    # - PydanticAI then falls back to the function's own docstring, same as before this existed.
    usage_guidance: str | None = None


def build_tools(entries: dict[str, ToolEntry]) -> list[Tool]:
    """The real PydanticAI tool list, built from a consumer's whole ToolEntry registry in one
    place - name, callable, enabled flag, and description (usage_guidance, not the function's
    docstring - see ToolEntry.usage_guidance) all come from the same entry. An entry with `fn=None`
    (pure debug/introspection metadata for a tool registered elsewhere) is skipped, and a disabled
    entry never reaches the model at all, not merely told not to use it."""
    return [
        Tool(entry.fn, name=name, description=entry.usage_guidance)
        for name, entry in entries.items()
        if entry.fn is not None and entry.enabled
    ]


def default_to_reply(result: Any) -> AgentReply:
    if isinstance(result, str):
        return AgentReply(text=json.dumps(result))
    if hasattr(result, "model_dump"):
        return AgentReply(text=json.dumps(result.model_dump(), indent=2, default=str))
    return AgentReply(text=json.dumps(result, indent=2, default=str))
