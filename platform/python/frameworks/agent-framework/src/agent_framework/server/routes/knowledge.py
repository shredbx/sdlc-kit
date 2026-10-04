"""GET /tools/{name}/knowledge - browses a knowledge-backed tool's real content (e.g. search_faq's
FAQ entries), for a dev console. Tool-scoped, not agent-scoped: a KnowledgeStore belongs to a
tool, not to whichever agent happens to have that tool registered. Read-only - see
core/knowledge/base.py's KnowledgeStore interface, which has no write method yet. Mounted only
when the server's own debug_mode is on (see server/app.py) - absent in prod, not merely
unauthenticated."""

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from agent_framework.core.tool_registry import ToolEntry


class KnowledgeVariantOut(BaseModel):
    when: str
    answer_template: str
    tone_note: str | None


class KnowledgeEntryOut(BaseModel):
    id: str
    category: str
    canonical_question: str
    variations: list[str]
    tags: list[str]
    variants: list[KnowledgeVariantOut]


def build_knowledge_router(tool_registry: dict[str, ToolEntry] | None) -> APIRouter:
    router = APIRouter()
    registry = tool_registry or {}

    @router.get("/tools/{name}/knowledge", response_model=list[KnowledgeEntryOut])
    async def get_knowledge(name: str) -> list[KnowledgeEntryOut]:
        entry = registry.get(name)
        if entry is None or entry.knowledge_store is None:
            raise HTTPException(status_code=404, detail=f"tool {name!r} has no browsable knowledge store")

        return [
            KnowledgeEntryOut(
                id=k.id,
                category=k.category,
                canonical_question=k.canonical_question,
                variations=list(k.variations),
                tags=list(k.tags),
                variants=[
                    KnowledgeVariantOut(when=v.when, answer_template=v.answer_template, tone_note=v.tone_note)
                    for v in k.variants
                ],
            )
            for k in entry.knowledge_store.all_entries()
        ]

    return router
