"""Regression test for a real gap: routes/chat.py used to call agent.run() with no
message_history, so every request was a fresh, memoryless conversation despite Session/
SessionStore existing. This proves history actually round-trips across two requests."""

from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic_ai import Agent, ModelRequest, ModelResponse, TextPart, UserPromptPart
from pydantic_ai.models.function import FunctionModel

_seen_history_lengths: list[int] = []


def _recording_model(messages: list, info: object) -> ModelResponse:
    _seen_history_lengths.append(len(messages))
    last_request = next(m for m in reversed(messages) if isinstance(m, ModelRequest))
    text = next(part.content for part in last_request.parts if isinstance(part, UserPromptPart))
    return ModelResponse(parts=[TextPart(content=f"got: {text}")])


async def test_second_request_carries_first_requests_history_as_context() -> None:
    _seen_history_lengths.clear()
    agent = Agent(FunctionModel(_recording_model), name="test_chat")
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    store = MemoryStore()

    app = FastAPI()
    app.include_router(build_router({"chat": registered}, store))
    client = TestClient(app)

    r1 = client.post("/agents/chat/chat", json={"message": "first"}, headers={"x-session-id": "s1"})
    r2 = client.post("/agents/chat/chat", json={"message": "second"}, headers={"x-session-id": "s1"})

    assert r1.json() == {"reply": "got: first"}
    assert r2.json() == {"reply": "got: second"}
    # The second call's model function must see more messages than the first — proof the prior
    # turn's history was actually passed in, not a fresh conversation each time.
    assert _seen_history_lengths[1] > _seen_history_lengths[0]

    session = await store.get_or_create("s1")
    assert len(session.messages) >= 4  # 2 requests + 2 responses, at minimum
