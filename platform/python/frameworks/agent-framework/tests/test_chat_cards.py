"""Proves the chat route serializes an agent's structured AgentReply (text + cards) as well as
its plain-str fallback — no real model, no API key. This deterministic FunctionModel harness is
also the reusable "send command + params, inspect the response" pattern for exercising the full
request/response cycle before swapping in a real model."""

from unittest.mock import ANY

from agent_framework.core.cards import AgentReply, Card, CardAction
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.routes.chat import build_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic_ai import Agent, ModelResponse, TextPart, ToolCallPart
from pydantic_ai.models.function import AgentInfo, FunctionModel


def _plain_text_model(messages: list, info: AgentInfo) -> ModelResponse:
    return ModelResponse(parts=[TextPart(content="just chatting")])


async def test_plain_str_output_leaves_cards_none() -> None:
    agent = Agent(FunctionModel(_plain_text_model), name="test_chat", output_type=str | AgentReply)
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="test-secret"))
    client = TestClient(app)

    response = client.post("/agents/chat/chat", json={"message": "hello"}, headers={"x-session-id": "s1"})

    assert response.json() == {"reply": "just chatting", "cards": None, "session_id": ANY, "limits": None}


def _structured_reply_model(messages: list, info: AgentInfo) -> ModelResponse:
    reply = AgentReply(
        text="Here's what I found:",
        cards=[Card(id="123", title="Sea View Villa", subtitle="villa · Srithanu", price_display="฿45,000", link="/p/123")],
    )
    final_result_tool = info.output_tools[0]
    return ModelResponse(parts=[ToolCallPart(tool_name=final_result_tool.name, args=reply.model_dump())])


async def test_structured_reply_serializes_text_and_cards() -> None:
    agent = Agent(FunctionModel(_structured_reply_model), name="test_chat", output_type=str | AgentReply)
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="test-secret"))
    client = TestClient(app)

    response = client.post("/agents/chat/chat", json={"message": "find me a villa"}, headers={"x-session-id": "s1"})

    assert response.json() == {
        "reply": "Here's what I found:",
        "cards": [
            {
                "id": "123",
                "title": "Sea View Villa",
                "subtitle": "villa · Srithanu",
                "price_display": "฿45,000",
                "image_url": None,
                "link": "/p/123",
                "actions": [],
            }
        ],
        "session_id": ANY,
        "limits": None,
    }


def _card_with_action_model(messages: list, info: AgentInfo) -> ModelResponse:
    reply = AgentReply(
        text="Here's what I found:",
        cards=[Card(id="123", title="Sea View Villa", link="/p/123", actions=[CardAction(label="I'm interested", text="I'm interested in Sea View Villa — /p/123")])],
    )
    final_result_tool = info.output_tools[0]
    return ModelResponse(parts=[ToolCallPart(tool_name=final_result_tool.name, args=reply.model_dump())])


async def test_card_actions_round_trip_on_a_rich_cards_channel() -> None:
    agent = Agent(FunctionModel(_card_with_action_model), name="test_chat", output_type=str | AgentReply)
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None)
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="test-secret"))
    client = TestClient(app)

    response = client.post("/agents/chat/chat", json={"message": "find me a villa"}, headers={"x-session-id": "s1"})

    assert response.json()["cards"][0]["actions"] == [{"label": "I'm interested", "text": "I'm interested in Sea View Villa — /p/123"}]


async def test_plain_text_channel_drops_card_actions_not_just_prints_them() -> None:
    # A CardAction is a button with nothing to click in a paste-back Messenger flow - dropped
    # entirely, not printed as a confusing extra text line duplicating the card's own title/link.
    agent = Agent(FunctionModel(_card_with_action_model), name="test_chat", output_type=str | AgentReply)
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None, output_protocol="plain_text")
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="test-secret"))
    client = TestClient(app)

    response = client.post("/agents/chat/chat", json={"message": "find me a villa"}, headers={"x-session-id": "s1"})

    assert "I'm interested" not in response.json()["reply"]


async def test_plain_text_channel_folds_cards_into_the_reply_text_instead_of_dropping_them() -> None:
    agent = Agent(FunctionModel(_structured_reply_model), name="test_chat", output_type=str | AgentReply)
    registered = RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None, output_protocol="plain_text")
    app = FastAPI()
    app.include_router(build_router({"chat": registered}, MemoryStore(), secret_key="test-secret"))
    client = TestClient(app)

    response = client.post("/agents/chat/chat", json={"message": "find me a villa"}, headers={"x-session-id": "s1"})

    body = response.json()
    assert body["cards"] is None  # not handed back raw - the consumer can't render them anyway
    assert body["reply"] == "Here's what I found:\n\nSea View Villa (villa · Srithanu) — ฿45,000 — /p/123"
