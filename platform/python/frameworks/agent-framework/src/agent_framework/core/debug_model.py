"""Debug mode: a `tool:name {json args}` chat message runs a real registered tool with no LLM,
through the agent's own run loop via `agent.override(model=build_debug_model(registry))` — so
session history keeps working, unlike a route-level bypass. Gated by two independent flags that
must both be true (see routes/chat.py): the server started with debug_mode=True, and the
request's own `debug: true`. Neither alone is enough, so a real user's message is never parsed as
a command even if it happens to match the shape.

Two calls per debug turn, mirroring PydanticAI's own tool-calling loop: the first sees the raw
`tool:name {json}` command and emits a real ToolCallPart (PydanticAI then actually runs that
tool); the second sees the ToolReturnPart with the real result and formats the final reply.

A `tools:list` command answers "what can I even call, and with what args?" by reading
`info.function_tools` — the exact `ToolDefinition` list (name, description, JSON schema)
PydanticAI hands a real model for tool-calling, not a parallel/hand-maintained list. Whatever a
real LLM would see is exactly what this prints."""

import json
import re
from collections.abc import Iterable
from typing import Any

from pydantic_ai import ModelRequest, ModelResponse, ToolCallPart, ToolReturnPart, UserPromptPart
from pydantic_ai.models.function import AgentInfo, FunctionModel
from pydantic_ai.tools import ToolDefinition

from agent_framework.core.cards import AgentReply
from agent_framework.core.tool_registry import ToolEntry, default_to_reply

_COMMAND = re.compile(r"^(?P<kind>tool):(?P<name>[a-zA-Z_]\w*)\s+(?P<args>\{.*\})\s*$", re.DOTALL)
_LIST_COMMAND = re.compile(r"^tools:list\s*$", re.IGNORECASE)


def build_debug_model(registry: dict[str, ToolEntry]) -> FunctionModel:
    def _model(messages: list[ModelRequest | ModelResponse], info: AgentInfo) -> ModelResponse:
        last_request = next(m for m in reversed(messages) if isinstance(m, ModelRequest))

        tool_return = next((p for p in last_request.parts if isinstance(p, ToolReturnPart)), None)
        if tool_return is not None:
            entry = registry.get(tool_return.tool_name)
            to_reply = (entry.to_reply if entry else None) or default_to_reply
            return _final_result(to_reply(tool_return.content), info)

        text = next(p.content for p in last_request.parts if isinstance(p, UserPromptPart))
        stripped = text.strip() if isinstance(text, str) else ""

        if _LIST_COMMAND.match(stripped):
            return _final_result(AgentReply(text=_describe_tools(info.function_tools)), info)

        match = _COMMAND.match(stripped)
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


def _describe_tools(tools: Iterable[ToolDefinition]) -> str:
    ordered = sorted(tools, key=lambda t: t.name)
    if not ordered:
        return "No tools registered on this agent."
    return "\n\n".join(_describe_tool(tool) for tool in ordered)


def _describe_tool(tool: ToolDefinition) -> str:
    schema = tool.parameters_json_schema or {}
    properties: dict[str, Any] = schema.get("properties", {})
    required = set(schema.get("required", []))

    lines = [f"tool:{tool.name} {{...}}"]
    if tool.description:
        lines.append(f"  {tool.description.strip()}")
    if not properties:
        lines.append("  (no parameters)")
    for param_name, param_schema in properties.items():
        marker = "required" if param_name in required else "optional"
        line = f"  - {param_name} ({_describe_type(param_schema)}, {marker})"
        description = param_schema.get("description")
        if description:
            line += f": {description}"
        lines.append(line)
    return "\n".join(lines)


def _describe_type(schema: dict[str, Any]) -> str:
    if "enum" in schema:
        return " | ".join(repr(v) for v in schema["enum"])
    if "anyOf" in schema:
        options = [_describe_type(option) for option in schema["anyOf"] if option.get("type") != "null"]
        return " | ".join(options) or "any"
    schema_type = schema.get("type")
    if isinstance(schema_type, list):
        return " | ".join(t for t in schema_type if t != "null") or "any"
    return schema_type or "any"


def _final_result(reply: AgentReply, info: AgentInfo) -> ModelResponse:
    final_result_tool = info.output_tools[0]
    return ModelResponse(parts=[ToolCallPart(tool_name=final_result_tool.name, args=reply.model_dump())])
