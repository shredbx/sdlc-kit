"""Route-level proof that build_router(..., limits=...) actually wires evaluate_and_consume in:
a throttled/over-quota request gets a 429 with {limits, session_id} and the model is never called
- not just that the pure function in core/limits.py is correct in isolation (see test_limits.py)."""

from agent_framework.core.limits import Limits
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic_ai import Agent, ModelResponse, TextPart
from pydantic_ai.models.function import FunctionModel

_calls: list[str] = []


def _counting_model(messages: list, info: object) -> ModelResponse:
    _calls.append("called")
    return ModelResponse(parts=[TextPart(content="ok")])


def _build_client(limits: Limits) -> TestClient:
    _calls.clear()
    agent = Agent(FunctionModel(_counting_model), name="test_chat")
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), limits=limits, secret_key="test-secret"))
    return TestClient(app)


def test_first_message_is_allowed_and_carries_limits() -> None:
    client = _build_client(Limits(quota_per_hour=40))

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 200
    assert response.json()["limits"] == {"remaining": 39, "resets_at": response.json()["limits"]["resets_at"]}
    assert _calls == ["called"]


def test_second_message_within_throttle_window_gets_429_and_skips_the_model() -> None:
    client = _build_client(Limits(throttle_seconds=60, quota_per_hour=40))
    r1 = client.post("/agents/chat/chat", json={"message": "hi"})
    token = r1.json()["session_id"]

    r2 = client.post("/agents/chat/chat", json={"message": "again"}, headers={"x-session-id": token})

    assert r2.status_code == 429
    assert r2.json()["limits"]["remaining"] == 39  # unchanged - the second message never consumed
    assert r2.json()["session_id"] == token
    assert _calls == ["called"]  # only the first request ever reached the model


def test_quota_exhausted_gets_429_with_zero_remaining() -> None:
    client = _build_client(Limits(throttle_seconds=0, quota_per_hour=1))
    r1 = client.post("/agents/chat/chat", json={"message": "hi"})
    token = r1.json()["session_id"]

    r2 = client.post("/agents/chat/chat", json={"message": "one too many"}, headers={"x-session-id": token})

    assert r2.status_code == 429
    assert r2.json()["limits"] == {"remaining": 0, "resets_at": r2.json()["limits"]["resets_at"]}


def test_no_limits_configured_means_no_check_at_all() -> None:
    _calls.clear()
    agent = Agent(FunctionModel(_counting_model), name="test_chat")
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="test-secret"))
    client = TestClient(app)

    r1 = client.post("/agents/chat/chat", json={"message": "hi"})
    token = r1.json()["session_id"]
    r2 = client.post("/agents/chat/chat", json={"message": "again, immediately"}, headers={"x-session-id": token})

    assert r1.status_code == 200 and r1.json()["limits"] is None
    assert r2.status_code == 200 and r2.json()["limits"] is None
    assert _calls == ["called", "called"]
