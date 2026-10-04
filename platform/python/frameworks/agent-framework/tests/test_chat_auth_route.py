"""Route-level proof that RegisteredAgent.auth="required" is actually enforced (core/auth.py's
UserVerifier was previously wired up nowhere - the field existed but did nothing): a missing or
invalid token is rejected before the model ever runs, a valid one links the session to that user
and rotates its id (rotate_for_user's own OWASP-motivated contract - the new row is saved with the finished turn), and a "public" agent's
behavior is completely unchanged (the default, and today's only real caller, the website widget)."""

from datetime import UTC, datetime

from agent_framework.core.auth import AuthUnavailable
from agent_framework.core.limits import Limits, evaluate_and_consume
from agent_framework.core.store.base import Session
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.middleware.session import sign_session_token
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from itsdangerous import URLSafeTimedSerializer
from pydantic_ai import Agent, ModelRequest, ModelResponse, TextPart
from pydantic_ai.exceptions import ModelHTTPError
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


def test_client_context_is_on_the_session_the_first_signed_in_turn_runs_and_saves() -> None:
    seen: list[dict] = []
    agent = Agent(FunctionModel(_counting_model), name="test_chat")
    registered = RegisteredAgent(
        name="chat",
        agent=agent,
        build_deps=lambda session: seen.append(dict(session.data.get("client_context") or {})),
        auth="required",
    )
    store = MemoryStore()
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store, user_verifier=_verify_good_token, secret_key="test-secret"))
    client = TestClient(app)

    response = client.post(
        "/agents/chat/chat",
        json={"message": "hi", "client_context": {"user_name": "Anna", "user_source": "facebook"}},
        headers={"Authorization": "Bearer good-token"},
    )

    assert response.status_code == 200
    assert seen == [{"user_name": "Anna", "user_source": "facebook"}]  # the first request of a signed-in conversation had its context
    assert [s.data["client_context"] for s in store._sessions.values()] == [{"user_name": "Anna", "user_source": "facebook"}]  # noqa: SLF001


# ---- the id rotation at login, and what a turn that does not finish leaves behind ------------------------------------------------


class _CountingStore(MemoryStore):
    """MemoryStore that counts the requests a real store would make."""

    def __init__(self) -> None:
        super().__init__()
        self.reads = 0
        self.writes = 0

    async def get_or_create(self, session_id: str) -> Session:
        self.reads += 1
        return await super().get_or_create(session_id)

    async def save(self, session: Session) -> None:
        self.writes += 1
        await super().save(session)


_SECRET = "test-secret"


def _client_with(store: MemoryStore, model_fn: object = _counting_model, **router: object) -> TestClient:
    _calls.clear()
    registered = RegisteredAgent(name="chat", agent=Agent(FunctionModel(model_fn), name="test_chat"), build_deps=lambda session: None, auth="required")  # type: ignore[arg-type]
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store, user_verifier=_verify_good_token, secret_key=_SECRET, **router))  # type: ignore[arg-type]
    return TestClient(app, raise_server_exceptions=False)


def _anonymous_chat(store: MemoryStore) -> str:
    """An anonymous session that already has content, as a store holds it before the person signs in; returns the token the client holds."""
    store._sessions["anon"] = Session(id="anon", data={"booking": "villa"})  # noqa: SLF001
    return sign_session_token(_SECRET, "anon")


def _id_of(token: str) -> str:
    return URLSafeTimedSerializer(_SECRET, salt="agent-framework.session").loads(token)


def test_a_finished_first_signed_in_turn_hands_back_the_new_token_and_leaves_only_the_new_row() -> None:
    store = MemoryStore()
    old_token = _anonymous_chat(store)
    client = _client_with(store)

    response = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token", "x-session-id": old_token})

    new_id = _id_of(response.json()["session_id"])
    assert response.status_code == 200 and new_id != "anon"
    assert set(store._sessions) == {new_id}  # noqa: SLF001 - the old row is gone
    assert store._sessions[new_id].user_id == "user-42" and store._sessions[new_id].data["booking"] == "villa"  # noqa: SLF001


def test_a_turn_that_fails_after_the_rotation_leaves_the_old_token_and_the_old_row_valid() -> None:
    def busy(messages: list, info: object) -> ModelResponse:
        raise ModelHTTPError(503, "some-model", body="busy")

    store = MemoryStore()
    old_token = _anonymous_chat(store)
    client = _client_with(store, busy)

    response = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token", "x-session-id": old_token})

    assert response.status_code == 503
    assert response.json()["session_id"] == old_token  # not a token for an id that was never saved
    assert set(store._sessions) == {"anon"}  # noqa: SLF001 - nothing new, nothing deleted


def test_a_throttled_first_signed_in_turn_also_hands_back_the_old_token() -> None:
    store = MemoryStore()
    old_token = _anonymous_chat(store)
    _, _, used = evaluate_and_consume({}, Limits(throttle_seconds=3600, quota_per_hour=40), datetime.now(UTC))
    store._sessions["anon"].data["limits"] = used  # noqa: SLF001 - a message was accepted a moment ago
    client = _client_with(store, limits=Limits(throttle_seconds=3600, quota_per_hour=40))

    response = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token", "x-session-id": old_token})

    assert response.status_code == 429 and response.json()["session_id"] == old_token
    assert set(store._sessions) == {"anon"} and _calls == []  # noqa: SLF001


def test_a_fresh_signed_in_turn_costs_one_read_and_one_write() -> None:
    store = _CountingStore()
    client = _client_with(store)

    response = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token"})

    assert response.status_code == 200
    assert (store.reads, store.writes) == (1, 1)  # a fresh session used to cost six requests against Supabase
    assert len(store._sessions) == 1  # noqa: SLF001 - and no empty row is left behind


def test_a_steady_turn_costs_one_read_and_one_write() -> None:
    store = _CountingStore()
    client = _client_with(store)
    token = client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token"}).json()["session_id"]
    store.reads = store.writes = 0

    client.post("/agents/chat/chat", json={"message": "again"}, headers={"Authorization": "Bearer good-token", "x-session-id": token})

    assert (store.reads, store.writes) == (1, 1)


def test_the_stored_history_keeps_no_copy_of_the_instructions() -> None:
    store = MemoryStore()
    agent = Agent(FunctionModel(_counting_model), name="test_chat", instructions="Be brief. " * 500)
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None, auth="required")  # type: ignore[arg-type]
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store, user_verifier=_verify_good_token, secret_key=_SECRET))
    client = TestClient(app)

    client.post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token"})

    (session,) = store._sessions.values()  # noqa: SLF001
    assert [m.instructions for m in session.messages if isinstance(m, ModelRequest)] == [None]


def test_an_identity_provider_that_cannot_answer_is_a_503_not_a_sign_out_and_the_model_is_not_run() -> None:
    async def provider_down(token: str) -> str | None:
        raise AuthUnavailable("Supabase Auth answered 503")

    store = MemoryStore()
    old_token = _anonymous_chat(store)
    registered = RegisteredAgent(name="chat", agent=Agent(FunctionModel(_counting_model), name="test_chat"), build_deps=lambda session: None, auth="required")  # type: ignore[arg-type]
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store, user_verifier=provider_down, secret_key=_SECRET))
    _calls.clear()

    response = TestClient(app).post("/agents/chat/chat", json={"message": "hi"}, headers={"Authorization": "Bearer good-token", "x-session-id": old_token})

    assert response.status_code == 503
    assert (response.json()["code"], response.json()["retryable"], response.json()["session_id"]) == ("auth_unavailable", True, old_token)
    assert _calls == [] and set(store._sessions) == {"anon"}  # noqa: SLF001
