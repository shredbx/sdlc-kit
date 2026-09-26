"""Generic chat endpoint: POST /agents/{name}/chat — works for any RegisteredAgent the registry
discovered. No per-agent route code needed; a new agent folder is automatically served here."""

from contextlib import nullcontext

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from agent_framework.core.cards import AgentReply, Card
from agent_framework.core.debug_model import build_debug_model
from agent_framework.core.store.base import SessionStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.middleware.session import ResolvedSession, session_dependency


class ChatRequest(BaseModel):
    message: str
    debug: bool = False


class ChatResponse(BaseModel):
    reply: str
    cards: list[Card] | None = None
    session_id: str


def build_router(
    agents: dict[str, RegisteredAgent],
    store: SessionStore,
    tool_registry: dict[str, ToolEntry] | None = None,
    debug_mode: bool = False,
    *,
    secret_key: str,
) -> APIRouter:
    router = APIRouter()
    resolve_session = session_dependency(store, secret_key)
    registry = tool_registry or {}

    @router.post("/agents/{name}/chat", response_model=ChatResponse)
    async def chat(name: str, request: ChatRequest, resolved: ResolvedSession = Depends(resolve_session)) -> ChatResponse:  # noqa: B008 — this is FastAPI's own DI idiom, not a mutable-default bug
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")

        session = resolved.session

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
            return ChatResponse(reply=output.text, cards=output.cards or None, session_id=resolved.token)
        return ChatResponse(reply=str(output), session_id=resolved.token)

    return router
