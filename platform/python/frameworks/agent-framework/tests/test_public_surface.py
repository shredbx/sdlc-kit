"""What a deployment exposes. A server built the way production builds it (docs off, debug off, a list of enabled
agents) has an exact, short list of routes, and an agent that is not on the list does not exist for callers - the
anonymous one included. Adding a route to the app makes the list test fail, on purpose: a new public door should be
a decision, not a side effect."""

import pytest
from agent_framework.core.types import RegisteredAgent
from agent_framework.server import app as app_module
from agent_framework.server.app import create_app
from fastapi.testclient import TestClient
from pydantic_ai import Agent
from pydantic_ai.models.test import TestModel


def _agent(name: str, auth: str) -> RegisteredAgent:
    return RegisteredAgent(name=name, agent=Agent(TestModel(), name=name), build_deps=lambda session: None, auth=auth)  # type: ignore[arg-type]


async def _verify(token: str) -> str | None:
    return "staff-1" if token == "staff" else None


@pytest.fixture(autouse=True)
def package(monkeypatch: pytest.MonkeyPatch) -> None:
    """A package with the signed-in assistant and, next to it, an anonymous public agent and an older one."""
    agents = [_agent("assistant", "required"), _agent("chat", "public"), _agent("legacy", "required")]
    monkeypatch.setattr(app_module, "discover", lambda _package: agents)


def production(**overrides: object) -> TestClient:
    options: dict[str, object] = {"docs_enabled": False, "debug_mode": False, "user_verifier": _verify, "secret_key": "k", "enabled_agents": ["assistant"]}
    return TestClient(create_app("none", **{**options, **overrides}), raise_server_exceptions=False)  # type: ignore[arg-type]


def routes_of(client: TestClient) -> set[tuple[str, str]]:
    """Every (method, path) the app answers, from its route table - includes routes hidden from the OpenAPI schema,
    which the schema itself would not list. Newer FastAPI keeps an included router behind a wrapper."""

    def flatten(routes: list) -> list:
        found = []
        for route in routes:
            inner = getattr(route, "original_router", None) or getattr(route, "router", None)
            found.extend(flatten(inner.routes) if inner is not None else [route])
        return found

    return {(method, route.path) for route in flatten(client.app.routes) for method in getattr(route, "methods", None) or ()}  # type: ignore[attr-defined]


class TestTheRoutes:
    def test_production_has_exactly_the_health_check_the_chat_routes_and_the_signed_in_usage_figures(self) -> None:
        expected = {("GET", "/health"), ("POST", "/agents/{name}/chat"), ("POST", "/agents/{name}/chat/stream"), ("GET", "/account/credits")}

        assert routes_of(production()) == expected

    def test_the_usage_figures_need_a_login_like_everything_else_but_the_health_check(self) -> None:
        client = production()

        assert client.get("/account/credits").status_code == 401
        assert client.get("/account/credits", headers={"authorization": "Bearer staff"}).json() == {"supported": False}

    def test_nothing_that_describes_the_api_or_its_agents_is_there(self) -> None:
        client = production()
        paths = [
            *("/", "/docs", "/redoc", "/openapi.json"),  # the app itself and its self-description
            *("/agents", "/agents/assistant", "/agents/assistant/tools", "/tools/search_faq/knowledge"),  # the dev console's views of the agents
            *("/metrics", "/.env"),
        ]

        assert {path: client.get(path).status_code for path in paths} == dict.fromkeys(paths, 404)

    def test_debug_mode_is_what_adds_the_dev_console_routes(self) -> None:
        paths = {path for _, path in routes_of(production(debug_mode=True))}

        assert {"/agents", "/agents/{name}", "/tools/{name}/knowledge"} <= paths


class TestWhichAgentsAreServed:
    def test_a_listed_agent_is_served_and_still_needs_a_login(self) -> None:
        client = production()

        assert client.post("/agents/assistant/chat", json={"message": "hi"}).status_code == 401
        assert client.post("/agents/assistant/chat", json={"message": "hi"}, headers={"authorization": "Bearer staff"}).status_code == 200

    @pytest.mark.parametrize("name", ["chat", "legacy", "nope"])
    @pytest.mark.parametrize("suffix", ["/chat", "/chat/stream"])
    def test_an_agent_that_is_not_listed_does_not_exist_for_callers(self, name: str, suffix: str) -> None:
        response = production().post(f"/agents/{name}{suffix}", json={"message": "hi"})

        assert response.status_code == 404
        assert response.json() == {"detail": f"no agent named {name!r}"}  # the same answer as a name that never existed

    def test_the_anonymous_agent_answers_without_a_login_only_when_it_is_listed(self) -> None:
        listed = production(enabled_agents=["assistant", "chat"])

        assert listed.post("/agents/chat/chat", json={"message": "hi"}).status_code == 200
        assert production().post("/agents/chat/chat", json={"message": "hi"}).status_code == 404

    def test_unset_serves_everything_found_so_the_framework_default_is_unchanged(self) -> None:
        client = production(enabled_agents=None)

        assert client.post("/agents/chat/chat", json={"message": "hi"}).status_code == 200

    def test_a_name_that_matches_no_agent_stops_startup_and_says_what_exists(self) -> None:
        with pytest.raises(ValueError, match=r"\['asistant'\].*found: \['assistant', 'chat', 'legacy'\]"):
            create_app("none", secret_key="k", enabled_agents=["asistant"])

    def test_an_empty_list_serves_no_agent(self) -> None:
        assert production(enabled_agents=[]).post("/agents/assistant/chat", json={"message": "hi"}).status_code == 404
