"""Proves the direct-invocation path: `name: {json}` calls a registered tool with no model
involved, sync or async tools both work, and anything else falls through untouched."""

from agent_framework.core.cards import AgentReply
from agent_framework.core.tool_registry import ToolEntry, try_invoke


async def _add(a: int, b: int) -> int:
    return a + b


def _shout(text: str) -> str:
    return text.upper()


async def test_invokes_a_registered_async_tool_with_parsed_json_args() -> None:
    reply = await try_invoke('add: {"a": 2, "b": 3}', {"add": ToolEntry(fn=_add)})

    assert reply == AgentReply(text="5")


async def test_invokes_a_registered_sync_tool() -> None:
    reply = await try_invoke('shout: {"text": "hi"}', {"shout": ToolEntry(fn=_shout)})

    assert reply == AgentReply(text='"HI"')


async def test_unregistered_tool_name_lists_whats_available() -> None:
    reply = await try_invoke("nope: {}", {"add": ToolEntry(fn=_add)})

    assert reply is not None
    assert "No tool named 'nope'" in reply.text
    assert "add" in reply.text


async def test_invalid_json_args_reports_the_error_instead_of_raising() -> None:
    reply = await try_invoke("add: {not json}", {"add": ToolEntry(fn=_add)})

    assert reply is not None
    assert "Invalid JSON args for 'add'" in reply.text


async def test_a_message_that_does_not_match_the_command_shape_falls_through() -> None:
    reply = await try_invoke("are pets allowed?", {"add": ToolEntry(fn=_add)})

    assert reply is None


async def test_custom_to_reply_overrides_the_default_json_dump() -> None:
    entry = ToolEntry(fn=_add, to_reply=lambda total: AgentReply(text=f"total={total}"))

    reply = await try_invoke('add: {"a": 1, "b": 1}', {"add": entry})

    assert reply == AgentReply(text="total=2")
