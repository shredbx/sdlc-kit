"""Proves debug mode end to end: a `tool:name {json}` command runs a real registered tool
through the agent's own run loop, custom/default reply formatting both work, an unknown tool name
or bad JSON reports an error instead of raising, and - the actual point of the double gate - a
real model never gets swapped out unless both the server and the request opt in."""

from unittest.mock import ANY

from agent_framework.core.cards import AgentReply, Card
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic import BaseModel
from pydantic_ai import Agent, ModelResponse, TextPart, Tool


class SearchOutput(BaseModel):
    count: int


async def _search(q: str) -> SearchOutput:
    return SearchOutput(count=len(q))


async def _search_with_options(q: str, limit: int | None = None) -> SearchOutput:
    """Search for things matching a query.

    Args:
        q: the search query text.
        limit: maximum number of results to return.
    """
    return SearchOutput(count=len(q))


def _real_model_response(messages: list, info: object) -> ModelResponse:
    return ModelResponse(parts=[TextPart(content="from the real model")])


def _build_app(*, debug_mode: bool, registry: dict[str, ToolEntry] | None = None, extra_tools: list[Tool] | None = None) -> TestClient:
    from pydantic_ai.models.function import FunctionModel

    agent = Agent(
        FunctionModel(_real_model_response),
        name="test_chat",
        output_type=str | AgentReply,
        tools=[Tool(_search, name="search"), *(extra_tools or [])],
    )
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), registry, debug_mode, secret_key="test-secret"))
    return TestClient(app)


async def test_debug_command_runs_the_real_tool_with_a_custom_reply() -> None:
    client = _build_app(
        debug_mode=True,
        registry={"search": ToolEntry(to_reply=lambda out: AgentReply(text="found", cards=[Card(id="1", title=f"n={out.count}")]))},
    )

    response = client.post("/agents/chat/chat", json={"message": 'tool:search {"q": "villa"}', "debug": True})

    expected_card = {"id": "1", "title": "n=5", "subtitle": None, "price_display": None, "image_url": None, "link": None, "actions": []}
    assert response.json() == {"reply": "found", "cards": [expected_card], "session_id": ANY, "limits": None}


async def test_debug_command_with_no_registry_entry_falls_back_to_json_dump() -> None:
    client = _build_app(debug_mode=True, registry={})

    response = client.post("/agents/chat/chat", json={"message": 'tool:search {"q": "abc"}', "debug": True})

    assert response.json() == {"reply": '{\n  "count": 3\n}', "cards": None, "session_id": ANY, "limits": None}


async def test_unknown_tool_name_lists_whats_actually_registered_on_the_agent() -> None:
    client = _build_app(debug_mode=True, registry={})

    response = client.post("/agents/chat/chat", json={"message": "tool:nope {}", "debug": True})

    assert response.json() == {"reply": "No tool named 'nope'. Available: search", "cards": None, "session_id": ANY, "limits": None}


async def test_invalid_json_args_reports_the_error() -> None:
    client = _build_app(debug_mode=True, registry={})

    response = client.post("/agents/chat/chat", json={"message": "tool:search {not json}", "debug": True})

    assert "Invalid JSON args for 'search'" in response.json()["reply"]


async def test_valid_tool_name_with_a_bad_kwarg_reports_the_validation_error_instead_of_crashing() -> None:
    client = _build_app(debug_mode=True, registry={})

    response = client.post("/agents/chat/chat", json={"message": 'tool:search {"nope": "x"}', "debug": True})

    assert response.status_code == 200
    reply = response.json()["reply"]
    assert reply.startswith("[debug mode] ")
    assert "nope" in reply


async def test_second_debug_command_in_the_same_session_is_not_shadowed_by_the_first_turns_leftover_bookkeeping() -> None:
    # Reproduces a real bug found live: a completed debug turn leaves PydanticAI's own internal
    # "Final result processed." bookkeeping ToolReturnPart in session history; on the next turn
    # that stale part lands in the same merged request as the new UserPromptPart. The second
    # command here must run search-for-real again, not echo back the first turn's leftover
    # bookkeeping text.
    client = _build_app(debug_mode=True, registry={})

    r1 = client.post("/agents/chat/chat", json={"message": 'tool:search {"q": "first"}', "debug": True})
    token = r1.json()["session_id"]
    r2 = client.post("/agents/chat/chat", json={"message": 'tool:search {"q": "second!"}', "debug": True}, headers={"x-session-id": token})

    assert r2.json() == {"reply": '{\n  "count": 7\n}', "cards": None, "session_id": ANY, "limits": None}


async def test_non_command_text_in_debug_mode_echoes_instead_of_reaching_the_real_model() -> None:
    client = _build_app(debug_mode=True, registry={})

    response = client.post("/agents/chat/chat", json={"message": "are pets allowed?", "debug": True})

    assert response.json() == {"reply": "[debug mode] are pets allowed?", "cards": None, "session_id": ANY, "limits": None}


async def test_server_debug_mode_off_ignores_a_debug_request_and_uses_the_real_model() -> None:
    client = _build_app(debug_mode=False, registry={})

    response = client.post("/agents/chat/chat", json={"message": 'tool:search {"q": "x"}', "debug": True})

    assert response.json() == {"reply": "from the real model", "cards": None, "session_id": ANY, "limits": None}


async def test_request_not_opted_in_uses_the_real_model_even_if_server_debug_mode_is_on() -> None:
    client = _build_app(debug_mode=True, registry={})

    response = client.post("/agents/chat/chat", json={"message": 'tool:search {"q": "x"}'})

    assert response.json() == {"reply": "from the real model", "cards": None, "session_id": ANY, "limits": None}


async def test_tools_list_describes_every_registered_tool_with_its_real_schema() -> None:
    client = _build_app(debug_mode=True, registry={}, extra_tools=[Tool(_search_with_options, name="search_with_options")])

    response = client.post("/agents/chat/chat", json={"message": "tools:list", "debug": True})

    reply = response.json()["reply"]
    assert "tool:search {...}" in reply
    assert "tool:search_with_options {...}" in reply
    assert "Search for things matching a query." in reply
    assert "q (string, required): the search query text." in reply
    assert "limit (integer, optional): maximum number of results to return." in reply
