"""Debug mode: a `tool:name {json args}` chat message runs a real registered tool with no LLM,
through the agent's own run loop via `agent.override(model=build_debug_model(registry))` — so
session history keeps working, unlike a route-level bypass. Gated by two independent flags that
must both be true (see routes/chat.py): the server started with debug_mode=True, and the
request's own `debug: true`. Neither alone is enough, so a real user's message is never parsed as
a command even if it happens to match the shape.

Two calls per debug turn, mirroring PydanticAI's own tool-calling loop: the first sees the raw
`tool:name {json}` command and emits a real ToolCallPart (PydanticAI then actually runs that
tool); the second sees the ToolReturnPart with the real result and formats the final reply."""

import json
import re

from pydantic_ai import ModelRequest, ModelResponse, ToolCallPart, ToolReturnPart, UserPromptPart
from pydantic_ai.models.function import AgentInfo, FunctionModel

from agent_framework.core.cards import AgentReply
from agent_framework.core.tool_registry import ToolEntry, default_to_reply

_COMMAND = re.compile(r"^(?P<kind>tool):(?P<name>[a-zA-Z_]\w*)\s+(?P<args>\{.*\})\s*$", re.DOTALL)


def build_debug_model(registry: dict[str, ToolEntry]) -> FunctionModel:
    def _model(messages: list[ModelRequest | ModelResponse], info: AgentInfo) -> ModelResponse:
        last_request = next(m for m in reversed(messages) if isinstance(m, ModelRequest))

        tool_return = next((p for p in last_request.parts if isinstance(p, ToolReturnPart)), None)
        if tool_return is not None:
            entry = registry.get(tool_return.tool_name)
            to_reply = (entry.to_reply if entry else None) or default_to_reply
            return _final_result(to_reply(tool_return.content), info)

        text = next(p.content for p in last_request.parts if isinstance(p, UserPromptPart))
        match = _COMMAND.match(text.strip()) if isinstance(text, str) else None
        if match is None:
            return _final_result(AgentReply(text=f"[debug mode] {text}"), info)

        name = match.group("name")
        real_tool_names = {tool.name for tool in info.function_tools}
        if name not in real_tool_names:
            available = ", ".join(sorted(real_tool_names)) or "(none registered on this agent)"
            return _final_result(AgentReply(text=f"No tool named {name!r}. Available: {available}"), info)

        try:
            args = json.loads(match.group("args"))
        except json.JSONDecodeError as exc:
            return _final_result(AgentReply(text=f"Invalid JSON args for {name!r}: {exc}"), info)

        return ModelResponse(parts=[ToolCallPart(tool_name=name, args=args)])

    return FunctionModel(_model)


def _final_result(reply: AgentReply, info: AgentInfo) -> ModelResponse:
    final_result_tool = info.output_tools[0]
    return ModelResponse(parts=[ToolCallPart(tool_name=final_result_tool.name, args=reply.model_dump())])
