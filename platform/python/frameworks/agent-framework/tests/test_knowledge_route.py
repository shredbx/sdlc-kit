"""Proves GET /tools/{name}/knowledge returns a knowledge-backed tool's real entries as JSON,
404s for a tool with no knowledge_store (or no registry entry at all), and doesn't exist unless
the app was built with debug_mode=True."""

from fastapi import FastAPI
from fastapi.testclient import TestClient

from agent_framework.core.knowledge.base import KnowledgeEntry, KnowledgeStore, KnowledgeVariant
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.server.app import create_app
from agent_framework.server.routes.knowledge import build_knowledge_router


class _FakeStore(KnowledgeStore):
    def all_entries(self) -> tuple[KnowledgeEntry, ...]:
        return (
            KnowledgeEntry(
                id="pool-1",
                category="amenities",
                canonical_question="Is there a pool?",
                variations=("pool available?",),
                tags=("pool",),
                variants=(KnowledgeVariant(when="always", answer_template="Yes!", tone_note="be warm"),),
            ),
        )


def _registry() -> dict[str, ToolEntry]:
    return {
        "search_faq": ToolEntry(kind="knowledge", knowledge_store=_FakeStore()),
        "handoff": ToolEntry(kind="function"),
    }


async def test_get_knowledge_returns_real_entries() -> None:
    app = FastAPI()
    app.include_router(build_knowledge_router(_registry()))
    client = TestClient(app)

    response = client.get("/tools/search_faq/knowledge")

    assert response.status_code == 200
    assert response.json() == [
        {
            "id": "pool-1",
            "category": "amenities",
            "canonical_question": "Is there a pool?",
            "variations": ["pool available?"],
            "tags": ["pool"],
            "variants": [{"when": "always", "answer_template": "Yes!", "tone_note": "be warm"}],
        }
    ]


async def test_get_knowledge_404s_for_a_tool_with_no_store() -> None:
    app = FastAPI()
    app.include_router(build_knowledge_router(_registry()))
    client = TestClient(app)

    response = client.get("/tools/handoff/knowledge")

    assert response.status_code == 404


async def test_get_knowledge_404s_for_an_unregistered_tool() -> None:
    app = FastAPI()
    app.include_router(build_knowledge_router(_registry()))
    client = TestClient(app)

    response = client.get("/tools/nope/knowledge")

    assert response.status_code == 404


async def test_knowledge_route_absent_when_debug_mode_is_off(tmp_path, monkeypatch) -> None:
    agents_pkg = tmp_path / "agents_pkg"
    agents_pkg.mkdir()
    (agents_pkg / "__init__.py").write_text("")
    monkeypatch.syspath_prepend(str(tmp_path))

    app = create_app(agents_package="agents_pkg", debug_mode=False, secret_key="test-secret")
    client = TestClient(app)

    response = client.get("/tools/search_faq/knowledge")

    assert response.status_code == 404
    assert response.json() == {"detail": "Not Found"}
