"""Direct tool invocation: a chat message shaped `tool_name: {json args}` calls that tool
directly and returns its reply, with no model in the loop — proves every tool works end to end
(parse args, run, format a reply) without spending on or depending on an LLM. Any other message
falls through to the normal agent, unchanged."""

import inspect
import json
import re
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from agent_framework.core.cards import AgentReply

_COMMAND = re.compile(r"^(?P<name>[a-zA-Z_]\w*):\s*(?P<args>\{.*\})\s*$", re.DOTALL)


@dataclass
class ToolEntry:
    """One directly-invokable tool. `to_reply` turns its return value into an `AgentReply`; if
    omitted, the result is JSON-dumped as plain text — works for any tool with no bespoke code,
    at the cost of not being card-shaped."""

    fn: Callable[..., Any]
    to_reply: Callable[[Any], AgentReply] | None = None


def _default_to_reply(result: Any) -> AgentReply:
    if isinstance(result, str):
        return AgentReply(text=json.dumps(result))
    if hasattr(result, "model_dump"):
        return AgentReply(text=json.dumps(result.model_dump(), indent=2, default=str))
    return AgentReply(text=json.dumps(result, indent=2, default=str))


async def try_invoke(message: str, registry: dict[str, ToolEntry]) -> AgentReply | None:
    """Returns the tool's reply if `message` is a direct-invocation command, else `None` (the
    caller should fall through to the normal agent)."""
    match = _COMMAND.match(message.strip())
    if match is None:
        return None

    name = match.group("name")
    entry = registry.get(name)
    if entry is None:
        available = ", ".join(sorted(registry)) or "(none registered)"
        return AgentReply(text=f"No tool named {name!r}. Available: {available}")

    try:
        args = json.loads(match.group("args"))
    except json.JSONDecodeError as exc:
        return AgentReply(text=f"Invalid JSON args for {name!r}: {exc}")

    result = entry.fn(**args)
    if inspect.isawaitable(result):
        result = await result

    to_reply = entry.to_reply or _default_to_reply
    return to_reply(result)
