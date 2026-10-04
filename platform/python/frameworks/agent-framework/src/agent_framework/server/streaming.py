"""The wire format of the streaming chat route: Server-Sent Events, one JSON object per event, and
the mapping from pydantic-ai's run events to those events. Pure functions - the route owns the
run itself (routes/chat.py).

Events a client can receive, in this order of appearance:
- `tool`  {id, name, status: "start" | "done"}  - a function tool is being called / has returned.
- `text`  {delta}                                - the next piece of the model's reply text. Text that
          arrives before a `tool` event belongs to a step that is not the final answer, so a client
          clears its partial text whenever a `tool` event starts.
- `done`  {reply, cards, session_id, limits, usage, state} - the finished turn; `reply` is the
          authoritative final text (after any output validation), not the concatenated deltas.
- `error` {code, retryable, detail, session_id}  - the run failed after streaming began (a failure
          before the first byte is a normal HTTP error response instead). Nothing was saved.
"""

import json
from typing import Any

from pydantic_ai.messages import (
    FunctionToolCallEvent,
    FunctionToolResultEvent,
    PartDeltaEvent,
    PartStartEvent,
    TextPart,
    TextPartDelta,
)


def sse(event: str, data: dict[str, Any]) -> str:
    """One SSE frame. JSON on a single `data:` line (json.dumps never emits a raw newline), so a
    frame is always `event: x\\ndata: {...}\\n\\n`."""
    return f"event: {event}\ndata: {json.dumps(data, ensure_ascii=False)}\n\n"


def frames_for(event: object) -> list[tuple[str, dict[str, Any]]]:
    """The SSE (name, data) pairs for one pydantic-ai stream event - empty for events a client has
    no use for (thinking, output-tool calls, part ends, ...)."""
    if isinstance(event, PartStartEvent) and isinstance(event.part, TextPart) and event.part.content:
        return [("text", {"delta": event.part.content})]
    if isinstance(event, PartDeltaEvent) and isinstance(event.delta, TextPartDelta) and event.delta.content_delta:
        return [("text", {"delta": event.delta.content_delta})]
    if isinstance(event, FunctionToolCallEvent):
        return [("tool", {"id": event.tool_call_id, "name": event.part.tool_name, "status": "start"})]
    if isinstance(event, FunctionToolResultEvent):
        return [("tool", {"id": event.tool_call_id, "name": getattr(event.part, "tool_name", None), "status": "done"})]
    return []
