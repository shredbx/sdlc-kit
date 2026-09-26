"""How debug_model.py's DebugAgent formats a tool's real return value into an AgentReply for
`tool:name {json}` commands — which tools actually exist and run comes from the agent's own
registered tools (see AgentInfo.function_tools), not from this registry; this only carries the
optional per-tool card mapping. `default_to_reply` JSON-dumps a result as plain text — works for
any tool with no bespoke code, at the cost of not being card-shaped."""

import json
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from agent_framework.core.cards import AgentReply


@dataclass
class ToolEntry:
    to_reply: Callable[[Any], AgentReply] | None = None


def default_to_reply(result: Any) -> AgentReply:
    if isinstance(result, str):
        return AgentReply(text=json.dumps(result))
    if hasattr(result, "model_dump"):
        return AgentReply(text=json.dumps(result.model_dump(), indent=2, default=str))
    return AgentReply(text=json.dumps(result, indent=2, default=str))
