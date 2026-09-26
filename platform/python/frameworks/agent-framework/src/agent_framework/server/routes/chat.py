"""Generic chat endpoint: POST /agents/{name}/chat — works for any RegisteredAgent the registry
discovered. No per-agent route code needed; a new agent folder is automatically served here."""

from contextlib import nullcontext

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from agent_framework.core.cards import AgentReply, Card
from agent_framework.core.debug_model import build_debug_model
from agent_framework.core.store.base import Session, SessionStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.middleware.session import session_dependency


class ChatRequest(BaseModel):
    message: str
    debug: bool = False


class ChatResponse(BaseModel):
    reply: str
    cards: list[Card] | None = None


def build_router(
    agents: dict[str, RegisteredAgent],
    store: SessionStore,
    tool_registry: dict[str, ToolEntry] | None = None,
    debug_mode: bool = False,
) -> APIRouter:
    router = APIRouter()
    resolve_session = session_dependency(store)
    registry = tool_registry or {}

    @router.post("/agents/{name}/chat", response_model=ChatResponse)
    async def chat(name: str, request: ChatRequest, session: Session = Depends(resolve_session)) -> ChatResponse:  # noqa: B008 — this is FastAPI's own DI idiom, not a mutable-default bug
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")

        # Both the server (debug_mode) and the request (request.debug) must opt in - neither
        # alone parses a message as a command, so a real user's text is never misread as one.
        override = registered.agent.override(model=build_debug_model(registry)) if debug_mode and request.debug else nullcontext()

        deps = registered.build_deps(session)
        with override:
            result = await registered.agent.run(request.message, deps=deps, message_history=session.messages)
        session.messages = result.all_messages()
        await store.save(session)
        output = result.output
        if isinstance(output, AgentReply):
            return ChatResponse(reply=output.text, cards=output.cards or None)
        return ChatResponse(reply=str(output))

    return router
