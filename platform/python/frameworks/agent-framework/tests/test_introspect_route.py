"""Proves GET /agents lists the registered agent names, GET /agents/{name} returns the agent's
real resolved system prompt (info.instructions) plus its real tool schema as JSON - the same data
tools:list already reads, structured, plus the individual personality/prompt_fields a consumer
optionally provides and each tool's registry-declared kind/card-mapping flag - and that neither
route exists when the app wasn't built with debug_mode=True."""

from agent_framework.core.cards import AgentReply
from agent_framework.core.personality.base import ConversionTechnique, Personality, PersonalityTrait
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.app import create_app
from agent_framework.server.routes.introspect import build_introspection_router
from fastapi import FastAPI
from fastapi.testclient import TestClient
from pydantic import BaseModel
from pydantic_ai import Agent, ModelResponse, TextPart, Tool
from pydantic_ai.models.function import FunctionModel


class SearchOutput(BaseModel):
    count: int


async def _search(q: str) -> SearchOutput:
    """Search for things.

    Args:
        q: the search query text.
    """
    return SearchOutput(count=len(q))


async def _lookup(q: str) -> SearchOutput:
    return SearchOutput(count=len(q))


def _real_model_response(messages: list, info: object) -> ModelResponse:
    return ModelResponse(parts=[TextPart(content="unused")])


_PERSONALITY = Personality(
    role="You are a helpful test assistant.",
    scope="testing things",
    traits=(PersonalityTrait(name="Concise", description="Keep it short."),),
    techniques=(ConversionTechnique(name="One CTA", when_to_use="Always.", examples=("Want to proceed?",)),),
)


def _build_agents(*, with_metadata: bool = False) -> dict[str, RegisteredAgent]:
    agent = Agent(
        FunctionModel(_real_model_response),
        name="test_chat",
        output_type=str | AgentReply,
        instructions="Be a helpful assistant. Never make up prices.",
        tools=[Tool(_search, name="search"), Tool(_lookup, name="lookup")],
    )
    kwargs = {}
    if with_metadata:
        kwargs = {"personality": _PERSONALITY, "prompt_fields": {"operational_instructions": "Call search first."}}
    return {"chat": RegisteredAgent(name="chat", agent=agent, build_deps=lambda session: None, **kwargs)}


def _registry() -> dict[str, ToolEntry]:
    return {
        "search": ToolEntry(
            to_reply=lambda out: AgentReply(text="found"),
            kind="function",
            output_schema=SearchOutput,
            usage_guidance="Use this when the customer wants to find something.",
        ),
        "lookup": ToolEntry(kind="knowledge"),
    }


async def test_list_agents_returns_registered_names() -> None:
    app = FastAPI()
    app.include_router(build_introspection_router(_build_agents()))
    client = TestClient(app)

    response = client.get("/agents")

    assert response.status_code == 200
    assert response.json() == ["chat"]


async def test_get_agent_returns_instructions_and_tools_with_kind_and_card_mapping() -> None:
    app = FastAPI()
    app.include_router(build_introspection_router(_build_agents(), _registry()))
    client = TestClient(app)

    response = client.get("/agents/chat")

    assert response.status_code == 200
    body = response.json()
    assert body["name"] == "chat"
    assert body["instructions"] == "Be a helpful assistant. Never make up prices."
    assert body["personality"] is None
    assert body["prompt_fields"] is None
    assert body["tools"] == [
        {
            "name": "lookup",
            "description": None,
            "parameters_json_schema": {
                "additionalProperties": False,
                "properties": {"q": {"type": "string"}},
                "required": ["q"],
                "type": "object",
            },
            "kind": "knowledge",
            "has_debug_card_mapping": False,
            "output_json_schema": None,
            "usage_guidance": None,
        },
        {
            "name": "search",
            "description": "Search for things.",
            "parameters_json_schema": {
                "additionalProperties": False,
                "properties": {"q": {"description": "the search query text.", "type": "string"}},
                "required": ["q"],
                "type": "object",
            },
            "kind": "function",
            "has_debug_card_mapping": True,
            "output_json_schema": {
                "properties": {"count": {"title": "Count", "type": "integer"}},
                "required": ["count"],
                "title": "SearchOutput",
                "type": "object",
            },
            "usage_guidance": "Use this when the customer wants to find something.",
        },
    ]


async def test_tool_with_no_registry_entry_defaults_to_function_kind_and_no_card_mapping() -> None:
    app = FastAPI()
    app.include_router(build_introspection_router(_build_agents()))  # no tool_registry passed at all
    client = TestClient(app)

    response = client.get("/agents/chat")

    tools = {t["name"]: t for t in response.json()["tools"]}
    assert tools["search"]["kind"] == "function"
    assert tools["search"]["has_debug_card_mapping"] is False
    assert tools["search"]["output_json_schema"] is None


async def test_get_agent_returns_personality_fields_and_prompt_fields_when_provided() -> None:
    app = FastAPI()
    app.include_router(build_introspection_router(_build_agents(with_metadata=True)))
    client = TestClient(app)

    response = client.get("/agents/chat")

    body = response.json()
    assert body["personality"] == {
        "role": "You are a helpful test assistant.",
        "scope": "testing things",
        "traits": [{"name": "Concise", "description": "Keep it short."}],
        "techniques": [{"name": "One CTA", "when_to_use": "Always.", "examples": ["Want to proceed?"]}],
        "phrases": {"use": [], "avoid": []},
        "rules": {"do": [], "dont": []},
        "hesitation_replies": [],
        "max_techniques_per_message": 2,
    }
    assert body["prompt_fields"] == {"operational_instructions": "Call search first."}


async def test_get_agent_404s_for_an_unknown_agent() -> None:
    app = FastAPI()
    app.include_router(build_introspection_router(_build_agents()))
    client = TestClient(app)

    response = client.get("/agents/nope")

    assert response.status_code == 404


async def test_introspection_routes_absent_when_debug_mode_is_off(tmp_path, monkeypatch) -> None:
    agents_pkg = tmp_path / "agents_pkg"
    agents_pkg.mkdir()
    (agents_pkg / "__init__.py").write_text("")
    monkeypatch.syspath_prepend(str(tmp_path))

    app = create_app(agents_package="agents_pkg", debug_mode=False, secret_key="test-secret")
    client = TestClient(app)

    response = client.get("/agents")

    assert response.status_code == 404
    assert response.json() == {"detail": "Not Found"}


async def test_introspection_routes_present_when_debug_mode_is_on(tmp_path, monkeypatch) -> None:
    agents_pkg = tmp_path / "agents_pkg2"
    agents_pkg.mkdir()
    (agents_pkg / "__init__.py").write_text("")
    monkeypatch.syspath_prepend(str(tmp_path))

    app = create_app(agents_package="agents_pkg2", debug_mode=True, secret_key="test-secret")
    client = TestClient(app)

    response = client.get("/agents")

    assert response.status_code == 200
    assert response.json() == []
