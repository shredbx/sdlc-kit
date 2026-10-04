"""The SSE wire format and the pydantic-ai event -> SSE mapping, built from the library's own event
classes. Pure - no run, no model."""

import json

from agent_framework.server.streaming import frames_for, sse
from pydantic_ai.messages import (
    FunctionToolCallEvent,
    FunctionToolResultEvent,
    OutputToolCallEvent,
    PartDeltaEvent,
    PartEndEvent,
    PartStartEvent,
    TextPart,
    TextPartDelta,
    ThinkingPart,
    ToolCallPart,
    ToolReturnPart,
)


def test_a_frame_is_event_line_json_data_line_and_a_blank_line() -> None:
    frame = sse("text", {"delta": "héllo\nworld"})

    assert frame == 'event: text\ndata: {"delta": "héllo\\nworld"}\n\n'
    # the newline inside the text is escaped, so the frame has exactly one data line
    assert frame.count("\n") == 3
    assert json.loads(frame.split("data: ")[1]) == {"delta": "héllo\nworld"}


def test_text_part_start_and_delta_become_text_events() -> None:
    assert frames_for(PartStartEvent(index=0, part=TextPart(content="Two places "))) == [("text", {"delta": "Two places "})]
    assert frames_for(PartDeltaEvent(index=0, delta=TextPartDelta(content_delta="are free"))) == [("text", {"delta": "are free"})]


def test_empty_text_produces_nothing() -> None:
    assert frames_for(PartStartEvent(index=0, part=TextPart(content=""))) == []
    assert frames_for(PartDeltaEvent(index=0, delta=TextPartDelta(content_delta=""))) == []


def test_function_tool_call_and_result_become_tool_events() -> None:
    call = ToolCallPart(tool_name="search", args={"q": "x"}, tool_call_id="c1")
    result = ToolReturnPart(tool_name="search", content="3 found", tool_call_id="c1")

    assert frames_for(FunctionToolCallEvent(part=call)) == [("tool", {"id": "c1", "name": "search", "status": "start"})]
    assert frames_for(FunctionToolResultEvent(part=result)) == [("tool", {"id": "c1", "name": "search", "status": "done"})]


def test_events_a_client_has_no_use_for_are_dropped() -> None:
    call = ToolCallPart(tool_name="final_result", args={}, tool_call_id="c2")

    assert frames_for(OutputToolCallEvent(part=call)) == []
    assert frames_for(PartStartEvent(index=0, part=ThinkingPart(content="hmm"))) == []
    assert frames_for(PartEndEvent(index=0, part=TextPart(content="done"))) == []
    assert frames_for("not an event") == []
