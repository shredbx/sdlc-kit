"""build_tools() is the single place a consumer's ToolEntry registry becomes the real PydanticAI
tool list - proves the three things the old two-registrations-by-hand setup got wrong: a disabled
entry is actually left out (not just told-not-to-use-me in prose), the description comes from
usage_guidance (not the function's docstring), and an entry with no `fn` (pure debug/introspection
metadata) is skipped rather than erroring."""

from agent_framework.core.tool_registry import ToolEntry, build_tools


def _search(region: str | None = None) -> str:
    """Docstring summary that must NOT become the tool's description once usage_guidance is set."""
    return "ok"


def test_enabled_entry_becomes_a_real_tool_with_usage_guidance_as_description() -> None:
    entries = {"search": ToolEntry(fn=_search, usage_guidance="Ask about the region first.")}

    tools = build_tools(entries)

    assert len(tools) == 1
    assert tools[0].name == "search"
    assert tools[0].description == "Ask about the region first."


def test_disabled_entry_is_left_out_entirely() -> None:
    entries = {
        "search": ToolEntry(fn=_search, enabled=True),
        "handoff": ToolEntry(fn=lambda: "ok", enabled=False),
    }

    tools = build_tools(entries)

    assert [t.name for t in tools] == ["search"]


def test_entry_with_no_fn_is_skipped_not_erroring() -> None:
    entries = {"lookup": ToolEntry(kind="knowledge")}  # pure debug/introspection metadata

    assert build_tools(entries) == []


def test_no_usage_guidance_falls_back_to_the_function_docstring() -> None:
    entries = {"search": ToolEntry(fn=_search)}  # usage_guidance left unset

    tools = build_tools(entries)

    assert tools[0].description is not None
    assert "Docstring summary" in tools[0].description
