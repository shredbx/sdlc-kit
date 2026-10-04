"""The cap on model requests per turn (build_router's max_requests_per_turn): a model that keeps calling tools is stopped at the cap with a clear
error on BOTH routes, a turn within the cap is untouched, and a stopped turn is not saved. Stub models only - no network."""

import json

from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic_ai import Agent, ModelResponse, TextPart, ToolCallPart
from pydantic_ai.models.function import AgentInfo, DeltaToolCall, FunctionModel

_requests: list[int] = []


def _ping() -> str:
    return "pong"


def _calls_a_tool_forever(messages: list, info: AgentInfo) -> ModelResponse:
    _requests.append(1)
    return ModelResponse(parts=[ToolCallPart(tool_name="_ping", args={}, tool_call_id=f"c{len(_requests)}")])


async def _streams_a_tool_call_forever(messages: list, info: AgentInfo):  # noqa: ANN202
    _requests.append(1)
    yield {0: DeltaToolCall(name="_ping", json_args="{}", tool_call_id=f"c{len(_requests)}")}


def _answers_after(tool_calls: int):  # noqa: ANN202
    def model(messages: list, info: AgentInfo) -> ModelResponse:
        _requests.append(1)
        if len(_requests) <= tool_calls:
            return ModelResponse(parts=[ToolCallPart(tool_name="_ping", args={}, tool_call_id=f"c{len(_requests)}")])
        return ModelResponse(parts=[TextPart(content="done")])

    return model


def _build(model: FunctionModel, cap: int | None) -> tuple[TestClient, MemoryStore]:
    _requests.clear()
    store = MemoryStore()
    agent = Agent(model, name="test_chat")
    agent.tool_plain(_ping)
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store, secret_key="test-secret", max_requests_per_turn=cap))
    return TestClient(app, raise_server_exceptions=False), store


def test_a_model_that_keeps_calling_tools_is_stopped_at_the_cap_with_a_retryable_error() -> None:
    client, store = _build(FunctionModel(_calls_a_tool_forever), cap=6)

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 502
    body = response.json()
    assert (body["code"], body["retryable"]) == ("too_many_steps", True) and body["session_id"]
    assert len(_requests) == 6  # not 50
    assert store._sessions == {}  # noqa: SLF001 - a stopped turn is not saved


def test_a_turn_within_the_cap_is_not_touched() -> None:
    client, _ = _build(FunctionModel(_answers_after(tool_calls=5)), cap=6)  # five tool requests and the answer: exactly six

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 200 and response.json()["reply"] == "done"
    assert len(_requests) == 6


def test_one_request_over_the_cap_is_stopped() -> None:
    client, _ = _build(FunctionModel(_answers_after(tool_calls=6)), cap=6)

    assert client.post("/agents/chat/chat", json={"message": "hi"}).status_code == 502


def test_without_a_cap_the_library_default_applies() -> None:
    client, _ = _build(FunctionModel(_calls_a_tool_forever), cap=None)

    assert client.post("/agents/chat/chat", json={"message": "hi"}).status_code == 502
    assert len(_requests) == 50


def test_the_stream_route_stops_at_the_cap_too_and_says_so_in_an_error_event() -> None:
    client, store = _build(FunctionModel(stream_function=_streams_a_tool_call_forever), cap=6)

    with client.stream("POST", "/agents/chat/chat/stream", json={"message": "hi"}) as response:
        text = response.read().decode()

    frames = [f.split("\n") for f in filter(None, text.split("\n\n"))]
    events = [(f[0].removeprefix("event: "), json.loads(f[1].removeprefix("data: "))) for f in frames]
    name, data = events[-1]
    assert name == "error" and (data["code"], data["retryable"]) == ("too_many_steps", True)
    assert len(_requests) == 6 and store._sessions == {}  # noqa: SLF001
