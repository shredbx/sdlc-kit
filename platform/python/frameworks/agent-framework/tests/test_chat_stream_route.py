"""Route-level proof of POST /agents/{name}/chat/stream with a stub streaming model (no network):
the event order, the final `done` payload (reply, cards, usage, state), that only a COMPLETED run is
saved, that a failure after streaming began arrives as an `error` event, and that every failure
BEFORE the first byte is still an ordinary HTTP error."""

import asyncio
import json

from agent_framework.core.cards import AgentReply, Card
from agent_framework.core.limits import Limits
from agent_framework.core.store.base import Session
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from agent_framework.server.usage import Pricing
from fastapi import FastAPI
from fastapi.testclient import TestClient
from itsdangerous import URLSafeTimedSerializer
from pydantic_ai import Agent
from pydantic_ai.exceptions import ModelHTTPError
from pydantic_ai.messages import ToolReturnPart
from pydantic_ai.models.function import AgentInfo, DeltaToolCall, FunctionModel

_SECRET = "test-secret"
_seen_history_lengths: list[int] = []


def _build(
    stream_fn: object,
    *,
    output_type: object = str,
    tools: tuple = (),
    state_summary: object = None,
    output_protocol: str = "rich_cards",
    auth: str = "public",
    user_verifier: object = None,
    limits: Limits | None = None,
    pricing: Pricing | None = None,
) -> tuple[TestClient, MemoryStore]:
    _seen_history_lengths.clear()
    store = MemoryStore()
    agent = Agent(FunctionModel(stream_function=stream_fn), name="test_chat", output_type=output_type)  # type: ignore[arg-type]
    for tool in tools:
        agent.tool_plain(tool)
    registered = RegisteredAgent(
        name="chat",
        agent=agent,
        build_deps=lambda session: None,
        auth=auth,  # type: ignore[arg-type]
        output_protocol=output_protocol,  # type: ignore[arg-type]
        state_summary=state_summary,  # type: ignore[arg-type]
    )
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store, user_verifier=user_verifier, limits=limits, secret_key=_SECRET, pricing=pricing))  # type: ignore[arg-type]
    return TestClient(app, raise_server_exceptions=False), store


def _stream(client: TestClient, message: str = "hi", token: str | None = None, **body: object) -> list[tuple[str, dict]]:
    headers = {"x-session-id": token} if token else {}
    with client.stream("POST", "/agents/chat/chat/stream", json={"message": message, **body}, headers=headers) as response:
        assert response.status_code == 200, response.read()
        assert response.headers["content-type"].startswith("text/event-stream")
        return _parse_sse(response.read().decode())


def _parse_sse(text: str) -> list[tuple[str, dict]]:
    events = []
    for frame in filter(None, text.split("\n\n")):
        name_line, data_line = frame.split("\n")
        events.append((name_line.removeprefix("event: "), json.loads(data_line.removeprefix("data: "))))
    return events


def _session(store: MemoryStore, token: str) -> Session:
    session_id = URLSafeTimedSerializer(_SECRET, salt="agent-framework.session").loads(token)
    return store._sessions[session_id]  # noqa: SLF001 - the test inspects what the route persisted


async def _two_chunks(messages: list, info: AgentInfo):  # noqa: ANN202
    _seen_history_lengths.append(len(messages))
    yield "Two places "
    yield "are free."


def test_text_arrives_in_pieces_then_done_carries_the_final_reply() -> None:
    client, _ = _build(_two_chunks)

    events = _stream(client)

    assert [name for name, _ in events] == ["text", "text", "done"]
    assert [data["delta"] for name, data in events if name == "text"] == ["Two places ", "are free."]
    done = events[-1][1]
    assert done["reply"] == "Two places are free."
    assert done["cards"] is None and done["limits"] is None and done["state"] is None
    assert done["session_id"]
    assert set(done["usage"]) == {"turn", "session", "cost_usd"} and done["usage"]["cost_usd"] is None


def _search(q: str) -> str:
    return "3 found"


async def _tool_then_text(messages: list, info: AgentInfo):  # noqa: ANN202
    if any(isinstance(part, ToolReturnPart) for message in messages for part in getattr(message, "parts", [])):
        yield "Found three."
    else:
        yield {0: DeltaToolCall(name="_search", json_args='{"q": "x"}', tool_call_id="c1")}


def test_tool_calls_are_announced_and_counted_in_the_usage() -> None:
    client, _ = _build(_tool_then_text, tools=(_search,), pricing=Pricing(input_per_mtok=1.0, output_per_mtok=2.0))

    events = _stream(client)

    assert [(name, data.get("status")) for name, data in events if name == "tool"] == [("tool", "start"), ("tool", "done")]
    assert [name for name, _ in events][-1] == "done"
    usage = events[-1][1]["usage"]
    assert usage["turn"]["tool_calls"] == 1 and usage["turn"]["requests"] == 2
    assert usage["session"] == usage["turn"]  # first turn of the session
    assert usage["cost_usd"] is not None and usage["cost_usd"] > 0


def test_session_usage_accumulates_across_turns() -> None:
    client, _ = _build(_two_chunks)
    first = _stream(client)[-1][1]

    second = _stream(client, "again", token=first["session_id"])[-1][1]

    assert second["usage"]["session"]["requests"] == 2
    assert second["usage"]["session"]["input_tokens"] > first["usage"]["session"]["input_tokens"]
    assert second["usage"]["turn"]["requests"] == 1


def test_a_completed_stream_is_saved_so_the_next_turn_sees_the_history() -> None:
    client, _ = _build(_two_chunks)
    first = _stream(client)[-1][1]

    _stream(client, "second", token=first["session_id"])

    assert _seen_history_lengths[0] == 1
    assert _seen_history_lengths[1] > 1


def test_the_state_summary_hook_feeds_the_done_event() -> None:
    client, _ = _build(_two_chunks, state_summary=lambda session: {"messages": len(session.messages)})

    done = _stream(client)[-1][1]

    assert done["state"] == {"messages": 2}


def test_a_broken_state_summary_never_costs_the_reply() -> None:
    def broken(session: Session) -> dict:
        raise RuntimeError("bug in the summary")

    client, _ = _build(_two_chunks, state_summary=broken)

    done = _stream(client)[-1][1]

    assert done["reply"] == "Two places are free." and done["state"] is None


async def _structured_reply(messages: list, info: AgentInfo):  # noqa: ANN202
    reply = AgentReply(text="Here's what I found:", cards=[Card(id="1", title="Sea View", subtitle="villa", price_display="45,000", link="/p/1")])
    yield {0: DeltaToolCall(name=info.output_tools[0].name, json_args=reply.model_dump_json(), tool_call_id="o1")}


def test_cards_come_back_in_done_on_a_rich_cards_channel() -> None:
    client, _ = _build(_structured_reply, output_type=str | AgentReply)

    done = _stream(client)[-1][1]

    assert done["reply"] == "Here's what I found:"
    assert done["cards"][0]["title"] == "Sea View"


def test_cards_are_folded_into_the_text_on_a_plain_text_channel() -> None:
    client, _ = _build(_structured_reply, output_type=str | AgentReply, output_protocol="plain_text")

    done = _stream(client)[-1][1]

    assert done["cards"] is None
    assert done["reply"] == "Here's what I found:\n\nSea View (villa) — 45,000 — /p/1"


async def _busy(messages: list, info: AgentInfo):  # noqa: ANN202
    raise ModelHTTPError(503, "some-model", body="provider body")
    yield  # pragma: no cover - makes this an async generator


def test_a_model_failure_after_the_stream_started_is_an_error_event_and_nothing_is_saved() -> None:
    client, store = _build(_busy)

    events = _stream(client)

    assert [name for name, _ in events] == ["error"]
    error = events[0][1]
    assert (error["code"], error["retryable"]) == ("unavailable", True)
    assert "provider body" not in json.dumps(error)
    assert _session(store, error["session_id"]).messages == []


async def _buggy(messages: list, info: AgentInfo):  # noqa: ANN202
    raise ValueError("bug")
    yield  # pragma: no cover


def test_an_unclassified_failure_is_an_internal_retryable_error_event() -> None:
    client, _ = _build(_buggy)

    events = _stream(client)

    assert events[-1][0] == "error"
    assert (events[-1][1]["code"], events[-1][1]["retryable"]) == ("internal", True)


def test_unknown_agent_is_a_normal_404_before_any_streaming() -> None:
    client, _ = _build(_two_chunks)

    response = client.post("/agents/nope/chat/stream", json={"message": "hi"})

    assert response.status_code == 404


def test_a_required_agent_without_a_token_is_a_normal_401() -> None:
    async def verify(token: str) -> str | None:
        return "user-1" if token == "good" else None

    client, _ = _build(_two_chunks, auth="required", user_verifier=verify)

    assert client.post("/agents/chat/chat/stream", json={"message": "hi"}).status_code == 401
    assert _stream_with_header(client, "Bearer good")[-1][0] == "done"


def _stream_with_header(client: TestClient, authorization: str) -> list[tuple[str, dict]]:
    with client.stream("POST", "/agents/chat/chat/stream", json={"message": "hi"}, headers={"authorization": authorization}) as response:
        assert response.status_code == 200
        return _parse_sse(response.read().decode())


def test_an_over_limit_message_is_a_normal_429_and_never_reaches_the_model() -> None:
    client, _ = _build(_two_chunks, limits=Limits(throttle_seconds=60, quota_per_hour=40))
    token = _stream(client)[-1][1]["session_id"]
    _seen_history_lengths.clear()

    response = client.post("/agents/chat/chat/stream", json={"message": "again"}, headers={"x-session-id": token})

    assert response.status_code == 429
    assert response.json()["limits"]["remaining"] == 39
    assert _seen_history_lengths == []


def test_debug_commands_are_not_available_on_the_stream_route() -> None:
    client, _ = _build(_two_chunks)

    response = client.post("/agents/chat/chat/stream", json={"message": "tool:x {}", "debug": True})

    assert response.status_code == 400


async def test_a_client_that_disconnects_mid_stream_cancels_the_run_and_nothing_is_saved() -> None:
    # What the extension's Stop button does: close the connection. Starlette cancels the response
    # generator, which cancels the run - complete_turn() is never reached, so the session keeps
    # neither the half-finished turn nor its token usage.
    async def stalls(messages: list, info: AgentInfo):  # noqa: ANN202
        yield "partial "
        await asyncio.sleep(30)
        yield "never reached"

    client, store = _build(stalls)
    app = client.app
    body = json.dumps({"message": "hi"}).encode()
    scope = {
        "type": "http",
        "asgi": {"version": "3.0"},
        "http_version": "1.1",
        "method": "POST",
        "scheme": "http",
        "path": "/agents/chat/chat/stream",
        "raw_path": b"/agents/chat/chat/stream",
        "query_string": b"",
        "root_path": "",
        "headers": [(b"content-type", b"application/json"), (b"content-length", str(len(body)).encode())],
        "server": ("test", 80),
        "client": ("test", 1),
    }
    disconnect = asyncio.Event()
    request_sent = False

    async def receive() -> dict:
        nonlocal request_sent
        if not request_sent:
            request_sent = True
            return {"type": "http.request", "body": body, "more_body": False}
        await disconnect.wait()
        return {"type": "http.disconnect"}

    sent: list[dict] = []

    async def send(message: dict) -> None:
        sent.append(message)
        if message["type"] == "http.response.body" and b"partial" in message.get("body", b""):
            disconnect.set()  # the user pressed Stop after seeing the first text

    await asyncio.wait_for(app(scope, receive, send), timeout=5)  # type: ignore[arg-type]

    assert any(b"partial" in m.get("body", b"") for m in sent)
    assert not any(b"event: done" in m.get("body", b"") for m in sent)
    assert store._sessions  # noqa: SLF001 - the session row exists (resolved before the run)...
    for session in store._sessions.values():  # noqa: SLF001
        assert session.messages == [] and "usage" not in session.data  # ...but nothing from the cancelled turn was saved
