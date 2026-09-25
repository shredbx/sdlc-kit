"""Generic chat endpoint: POST /agents/{name}/chat — works for any RegisteredAgent the registry
discovered. No per-agent route code needed; a new agent folder is automatically served here."""

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from agent_framework.core.cards import AgentReply, Card
from agent_framework.core.store.base import Session, SessionStore
from agent_framework.core.tool_registry import ToolEntry, try_invoke
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.middleware.session import session_dependency


class ChatRequest(BaseModel):
    message: str


class ChatResponse(BaseModel):
    reply: str
    cards: list[Card] | None = None


def build_router(
    agents: dict[str, RegisteredAgent],
    store: SessionStore,
    tool_registry: dict[str, ToolEntry] | None = None,
) -> APIRouter:
    router = APIRouter()
    resolve_session = session_dependency(store)
    registry = tool_registry or {}

    @router.post("/agents/{name}/chat", response_model=ChatResponse)
    async def chat(name: str, request: ChatRequest, session: Session = Depends(resolve_session)) -> ChatResponse:  # noqa: B008 — this is FastAPI's own DI idiom, not a mutable-default bug
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")

        direct = await try_invoke(request.message, registry)
        if direct is not None:
            return ChatResponse(reply=direct.text, cards=direct.cards or None)

        deps = registered.build_deps(session)
        result = await registered.agent.run(request.message, deps=deps, message_history=session.messages)
        session.messages = result.all_messages()
        await store.save(session)
        output = result.output
        if isinstance(output, AgentReply):
            return ChatResponse(reply=output.text, cards=output.cards or None)
        return ChatResponse(reply=str(output))

    return router
