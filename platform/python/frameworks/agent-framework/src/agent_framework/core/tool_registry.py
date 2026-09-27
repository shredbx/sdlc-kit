"""How debug_model.py's DebugAgent formats a tool's real return value into an AgentReply for
`tool:name {json}` commands — which tools actually exist and run comes from the agent's own
registered tools (see AgentInfo.function_tools), not from this registry; this only carries the
optional per-tool card mapping, plus metadata for introspection (see server/routes/introspect.py).
`default_to_reply` JSON-dumps a result as plain text — works for any tool with no bespoke code, at
the cost of not being card-shaped."""

import json
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from pydantic import BaseModel

from agent_framework.core.cards import AgentReply
from agent_framework.core.knowledge.base import KnowledgeStore


@dataclass
class ToolEntry:
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
    # When/how to use this tool and how to phrase a reply from its result - plain code living next
    # to the tool it describes (a module-level constant in the tool's own file), not data, for the
    # same reason personality/base.py's own docstring gives for operational instructions generally:
    # tightly coupled to the tool's real schema in a way prose data can drift out of sync with.
    # None for a tool whose usage is self-evident or has no special reply-phrasing rules.
    usage_guidance: str | None = None


def default_to_reply(result: Any) -> AgentReply:
    if isinstance(result, str):
        return AgentReply(text=json.dumps(result))
    if hasattr(result, "model_dump"):
        return AgentReply(text=json.dumps(result.model_dump(), indent=2, default=str))
    return AgentReply(text=json.dumps(result, indent=2, default=str))
