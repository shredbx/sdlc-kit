"""The per-turn record: what it contains (and never contains), how it is written, and that both chat
routes write one for every turn however it ends. Stub models only."""

import asyncio
import io
import json
import logging
from datetime import UTC, datetime
from decimal import Decimal

import pytest
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from agent_framework.server.turn_log import LOGGER_NAME, configure_turn_log, log_turn, turn_record
from agent_framework.server.usage import TokenUsage
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic_ai import Agent, ModelResponse, TextPart
from pydantic_ai.capabilities.hooks import Hooks
from pydantic_ai.exceptions import ModelHTTPError
from pydantic_ai.models.function import FunctionModel

NOW = datetime(2026, 10, 2, 14, 8, 3, tzinfo=UTC)


def record(**overrides):  # noqa: ANN003, ANN201
    args = {
        "agent": "assistant",
        "session_id": "0123456789abcdef",
        "user_id": "user-1",
        "client_context": {"user_source": "facebook", "user_name": "Anna", "user_contact": "https://x"},
        "streamed": True,
        "latency_seconds": 1.2345,
        "status": "ok",
        "usage": TokenUsage(requests=3, input_tokens=12_000, output_tokens=400, tool_calls=2),
        "cost_usd": 0.0068,
        "now": NOW,
    }
    return turn_record(**{**args, **overrides})


class TestTurnRecord:
    def test_the_fields(self) -> None:
        assert record() == {
            "event": "chat_turn",
            "ts": "2026-10-02T14:08:03+00:00",
            "agent": "assistant",
            "session": "01234567",
            "user_id": "user-1",
            "source": "facebook",
            "stream": True,
            "status": "ok",
            "code": None,
            "latency_ms": 1234,
            "requests": 3,
            "tool_calls": 2,
            "input_tokens": 12_000,
            "output_tokens": 400,
            "cost_usd": 0.0068,
        }

    def test_nothing_of_the_conversation_is_in_it(self) -> None:
        text = json.dumps(record())

        assert "Anna" not in text and "https://x" not in text

    def test_a_failed_turn_has_a_code_and_zero_usage(self) -> None:
        failed = record(status="error", code="unavailable", usage=None, cost_usd=None)

        assert (failed["status"], failed["code"], failed["input_tokens"], failed["cost_usd"]) == ("error", "unavailable", 0, None)

    def test_an_unknown_cost_stays_unknown(self) -> None:
        assert record(cost_usd=None)["cost_usd"] is None

    def test_a_missing_or_odd_source_is_none(self) -> None:
        assert record(client_context=None)["source"] is None
        assert record(client_context={"user_source": 5})["source"] is None


class TestLogging:
    def test_one_json_line_on_the_turn_logger(self, caplog: pytest.LogCaptureFixture) -> None:
        caplog.set_level(logging.INFO, logger=LOGGER_NAME)

        log_turn(record())

        [line] = [r.getMessage() for r in caplog.records if r.name == LOGGER_NAME]
        assert json.loads(line)["event"] == "chat_turn"

    def test_configure_writes_bare_json_lines_to_the_stream(self) -> None:
        stream = io.StringIO()
        logger = logging.getLogger(LOGGER_NAME)
        try:
            configure_turn_log(stream)
            log_turn(record())
            log_turn(record(status="error", code="timeout"))
        finally:
            logger.handlers.clear()
            logger.propagate = True

        lines = stream.getvalue().strip().split("\n")
        assert [json.loads(line)["status"] for line in lines] == ["ok", "error"]


def priced(cost: Decimal) -> Hooks:
    """A model that costs `cost` per request, the way a real provider's answer carries its price."""
    hooks = Hooks()

    @hooks.on.after_model_request
    async def set_cost(ctx, *, request_context, response):  # noqa: ANN001, ANN202, ARG001
        response.usage.cost = cost
        return response

    return hooks


def build_client(model: FunctionModel) -> TestClient:
    registered = RegisteredAgent(name="chat", agent=Agent(model, name="t", capabilities=[priced(Decimal("0.0012"))]), build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="k"))
    return TestClient(app, raise_server_exceptions=False)


def turn_lines(caplog: pytest.LogCaptureFixture) -> list[dict]:
    return [json.loads(r.getMessage()) for r in caplog.records if r.name == LOGGER_NAME]


def ok_model(messages: list, info: object) -> ModelResponse:
    return ModelResponse(parts=[TextPart(content="ok")])


async def ok_stream(messages: list, info: object):  # noqa: ANN201
    yield "ok"


async def busy_stream(messages: list, info: object):  # noqa: ANN201
    raise ModelHTTPError(503, "m", body="busy")
    yield  # pragma: no cover


class TestRoutesWriteARecordPerTurn:
    def test_plain_ok(self, caplog: pytest.LogCaptureFixture) -> None:
        caplog.set_level(logging.INFO, logger=LOGGER_NAME)

        build_client(FunctionModel(ok_model)).post("/agents/chat/chat", json={"message": "hi"})

        [line] = turn_lines(caplog)
        assert (
            (line["agent"], line["stream"], line["status"], line["requests"]) == ("chat", False, "ok", 1)
            and line["input_tokens"] > 0
            and line["cost_usd"] == 0.0012
        )

    def test_plain_model_failure(self, caplog: pytest.LogCaptureFixture) -> None:
        def busy(messages: list, info: object) -> ModelResponse:
            raise ModelHTTPError(503, "m", body="busy")

        caplog.set_level(logging.INFO, logger=LOGGER_NAME)

        build_client(FunctionModel(busy)).post("/agents/chat/chat", json={"message": "hi"})

        [line] = turn_lines(caplog)
        assert (line["status"], line["code"], line["input_tokens"]) == ("error", "unavailable", 0)

    def test_stream_ok_and_failure(self, caplog: pytest.LogCaptureFixture) -> None:
        caplog.set_level(logging.INFO, logger=LOGGER_NAME)

        build_client(FunctionModel(stream_function=ok_stream)).post("/agents/chat/chat/stream", json={"message": "hi"})
        build_client(FunctionModel(stream_function=busy_stream)).post("/agents/chat/chat/stream", json={"message": "hi"})

        ok, failed = turn_lines(caplog)
        assert (ok["stream"], ok["status"]) == (True, "ok")
        assert (failed["status"], failed["code"]) == ("error", "unavailable")

    @pytest.mark.asyncio
    async def test_a_stream_the_client_walks_away_from_is_recorded_as_cancelled(self, caplog: pytest.LogCaptureFixture) -> None:
        async def stalls(messages: list, info: object):  # noqa: ANN202
            yield "partial "
            await asyncio.sleep(30)
            yield "never"

        caplog.set_level(logging.INFO, logger=LOGGER_NAME)
        app = build_client(FunctionModel(stream_function=stalls)).app
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
        disconnect, sent_request = asyncio.Event(), False

        async def receive() -> dict:
            nonlocal sent_request
            if not sent_request:
                sent_request = True
                return {"type": "http.request", "body": body, "more_body": False}
            await disconnect.wait()
            return {"type": "http.disconnect"}

        async def send(message: dict) -> None:
            if message["type"] == "http.response.body" and b"partial" in message.get("body", b""):
                disconnect.set()

        await asyncio.wait_for(app(scope, receive, send), timeout=5)  # type: ignore[arg-type]

        [line] = turn_lines(caplog)
        assert (line["stream"], line["status"]) == (True, "cancelled")
