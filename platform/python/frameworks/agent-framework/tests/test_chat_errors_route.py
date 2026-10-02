"""Route-level proof of the error contract on POST /agents/{name}/chat: a model/network failure is a
real HTTP error with a machine-readable code (never a 200 that looks like a reply), the provider's
own detail never reaches the client, a bug in a tool stays a plain 500, and a failed turn leaves the
session exactly as it was so re-sending the same message starts from the same point. Stub models
only - no network."""

import httpx
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from itsdangerous import URLSafeTimedSerializer
from pydantic_ai import Agent, ModelResponse, TextPart
from pydantic_ai.exceptions import ModelHTTPError
from pydantic_ai.models.function import FunctionModel

_SECRET = "test-secret"
_seen_history_lengths: list[int] = []


def _build(model_fn: object) -> tuple[TestClient, MemoryStore]:
    _seen_history_lengths.clear()
    store = MemoryStore()
    registered = RegisteredAgent(name="chat", agent=Agent(FunctionModel(model_fn), name="test_chat"), build_deps=lambda session: None)  # type: ignore[arg-type]
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store, secret_key=_SECRET))
    return TestClient(app, raise_server_exceptions=False), store


def _ok_model(messages: list, info: object) -> ModelResponse:
    _seen_history_lengths.append(len(messages))
    return ModelResponse(parts=[TextPart(content="ok")])


def _session(store: MemoryStore, token: str):  # noqa: ANN202
    session_id = URLSafeTimedSerializer(_SECRET, salt="agent-framework.session").loads(token)
    return store._sessions[session_id]  # noqa: SLF001 - the test inspects what the route persisted


def test_a_busy_model_is_a_503_with_a_code_not_a_200_reply() -> None:
    def busy(messages: list, info: object) -> ModelResponse:
        raise ModelHTTPError(503, "some-model", body="provider body with request details", headers={"Retry-After": "7"})

    client, _ = _build(busy)

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 503
    assert response.headers["retry-after"] == "7"
    body = response.json()
    assert body["code"] == "unavailable" and body["retryable"] is True
    assert body["session_id"]
    assert "reply" not in body
    assert "request details" not in response.text


def test_a_rejected_key_is_a_502_that_says_not_to_retry() -> None:
    def rejected(messages: list, info: object) -> ModelResponse:
        raise ModelHTTPError(401, "some-model", body="invalid key")

    client, _ = _build(rejected)

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 502
    assert (response.json()["code"], response.json()["retryable"]) == ("rejected", False)


def test_a_connection_error_is_a_503() -> None:
    def unreachable(messages: list, info: object) -> ModelResponse:
        raise httpx.ConnectError("connection refused")

    client, _ = _build(unreachable)

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 503 and response.json()["code"] == "unavailable"


def test_a_bug_that_is_not_a_model_failure_stays_a_plain_500() -> None:
    def buggy(messages: list, info: object) -> ModelResponse:
        raise ValueError("bug in the app")

    client, _ = _build(buggy)

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 500
    assert "code" not in response.text


def test_a_failed_turn_is_not_saved_so_a_resend_starts_from_the_same_point() -> None:
    calls = {"n": 0}

    def flaky(messages: list, info: object) -> ModelResponse:
        calls["n"] += 1
        if calls["n"] == 1:
            raise ModelHTTPError(503, "some-model", body="busy")
        _seen_history_lengths.append(len(messages))
        return ModelResponse(parts=[TextPart(content="ok")])

    client, store = _build(flaky)
    failed = client.post("/agents/chat/chat", json={"message": "first"})
    token = failed.json()["session_id"]
    assert _session(store, token).messages == []

    resent = client.post("/agents/chat/chat", json={"message": "first"}, headers={"x-session-id": token})

    assert resent.status_code == 200
    assert _seen_history_lengths == [1]  # only the re-sent prompt, not a duplicate of the failed one


def test_a_completed_turn_adds_its_token_usage_to_the_session() -> None:
    client, store = _build(_ok_model)
    first = client.post("/agents/chat/chat", json={"message": "hi there"})
    token = first.json()["session_id"]
    after_first = dict(_session(store, token).data["usage"])

    client.post("/agents/chat/chat", json={"message": "and again"}, headers={"x-session-id": token})
    after_second = _session(store, token).data["usage"]

    assert after_first["requests"] == 1 and after_first["input_tokens"] > 0
    assert after_second["requests"] == 2
    assert after_second["input_tokens"] > after_first["input_tokens"]
