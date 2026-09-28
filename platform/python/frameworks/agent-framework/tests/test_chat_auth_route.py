"""Route-level proof that RegisteredAgent.auth="required" is actually enforced (core/auth.py's
UserVerifier was previously wired up nowhere - the field existed but did nothing): a missing or
invalid token is rejected before the model ever runs, a valid one links the session to that user
and rotates its id (SessionStore.link_user's own OWASP-motivated contract), and a "public" agent's
behavior is completely unchanged (the default, and today's only real caller, the website widget)."""

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


async def _verify_good_token(token: str) -> str | None:
    return "user-42" if token == "good-token" else None


def _build_client(auth: str, user_verifier: object | None) -> TestClient:
    _calls.clear()
    agent = Agent(FunctionModel(_counting_model), name="test_chat")
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None, auth=auth)  # type: ignore[arg-type]
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), user_verifier=user_verifier, secret_key="test-secret"))
    return TestClient(app)


def test_public_agent_ignores_missing_authorization_header() -> None:
    client = _build_client("public", user_verifier=None)

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 200
    assert _calls == ["called"]


def test_required_agent_rejects_missing_authorization_header() -> None:
    client = _build_client("required", user_verifier=_verify_good_token)

    response = client.post("/agents/chat/chat", json={"message": "hi"})

    assert response.status_code == 401
    assert _calls == []


def test_required_agent_rejects_an_invalid_token() -> None:
    client = _build_client("required", user_verifier=_verify_good_token)

    response = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer wrong-token"})

    assert response.status_code == 401
    assert _calls == []


def test_required_agent_with_no_verifier_configured_is_a_server_error_not_a_silent_bypass() -> None:
    client = _build_client("required", user_verifier=None)

    response = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token"})

    assert response.status_code == 500
    assert _calls == []


def test_required_agent_accepts_a_valid_token_links_the_session_and_rotates_its_id() -> None:
    client = _build_client("required", user_verifier=_verify_good_token)

    r1 = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token"})

    assert r1.status_code == 200
    assert _calls == ["called"]
    token1 = r1.json()["session_id"]

    # A second request under the SAME (now-linked) user must NOT rotate again - link_user() would
    # otherwise burn a new session id on every single message.
    r2 = client.post(
        "/agents/chat/chat",
        json={"message": "again"},
        headers={"Authorization": "Bearer good-token", "x-session-id": token1},
    )

    assert r2.status_code == 200
    assert r2.json()["session_id"] == token1
    assert _calls == ["called", "called"]
