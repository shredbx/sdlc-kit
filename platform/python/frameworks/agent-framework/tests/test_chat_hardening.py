"""Size caps (413 before any model call, in the error contract's shape) and the docs switch."""

import pytest
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server import app as app_module
from agent_framework.server.app import create_app
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic_ai import Agent, ModelResponse, TextPart
from pydantic_ai.models.function import FunctionModel

calls: list[str] = []


def model(messages: list, info: object) -> ModelResponse:
    calls.append("called")
    return ModelResponse(parts=[TextPart(content="ok")])


async def stream_model(messages: list, info: object):  # noqa: ANN201
    calls.append("called")
    yield "ok"


def client(max_message_chars: int | None) -> TestClient:
    calls.clear()
    registered = RegisteredAgent(name="chat", agent=Agent(FunctionModel(model, stream_function=stream_model), name="t"), build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="k", max_message_chars=max_message_chars))
    return TestClient(app)


@pytest.mark.parametrize("path", ["/agents/chat/chat", "/agents/chat/chat/stream"])
class TestMessageCap:
    def test_a_message_over_the_cap_is_refused_before_the_model(self, path: str) -> None:
        response = client(10).post(path, json={"message": "x" * 11})

        assert response.status_code == 413
        body = response.json()
        assert (body["code"], body["retryable"]) == ("message_too_long", False) and "10 characters" in body["detail"] and body["session_id"]
        assert calls == []

    def test_a_message_at_the_cap_goes_through(self, path: str) -> None:
        assert client(10).post(path, json={"message": "x" * 10}).status_code == 200
        assert calls == ["called"]

    def test_no_cap_means_any_length(self, path: str) -> None:
        assert client(None).post(path, json={"message": "x" * 50_000}).status_code == 200

    def test_an_oversized_context_is_refused(self, path: str) -> None:
        response = client(None).post(path, json={"message": "hi", "client_context": {"blob": "x" * 5000}})

        assert response.status_code == 413 and response.json()["code"] == "context_too_large"
        assert calls == []

    def test_a_normal_context_is_fine(self, path: str) -> None:
        assert client(None).post(path, json={"message": "hi", "client_context": {"user_name": "Anna", "user_source": "facebook"}}).status_code == 200


class TestDocsSwitch:
    @pytest.fixture(autouse=True)
    def no_agents(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(app_module, "discover", lambda _package: [])

    def test_docs_are_on_by_default(self) -> None:
        test_client = TestClient(create_app("none", secret_key="k"))

        assert test_client.get("/docs").status_code == 200 and test_client.get("/openapi.json").status_code == 200

    def test_docs_can_be_switched_off_and_health_still_works(self) -> None:
        test_client = TestClient(create_app("none", secret_key="k", docs_enabled=False))

        assert [test_client.get(path).status_code for path in ("/docs", "/redoc", "/openapi.json")] == [404, 404, 404]
        assert test_client.get("/health").json() == {"status": "ok"}
