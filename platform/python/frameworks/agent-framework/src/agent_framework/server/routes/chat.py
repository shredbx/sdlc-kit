"""Generic chat endpoint: POST /agents/{name}/chat — works for any RegisteredAgent the registry
discovered. No per-agent route code needed; a new agent folder is automatically served here."""

from contextlib import nullcontext
from datetime import UTC, datetime

from fastapi import APIRouter, Depends, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from agent_framework.core.cards import AgentReply, Card
from agent_framework.core.debug_model import build_debug_model
from agent_framework.core.limits import Limits, LimitStatus, evaluate_and_consume
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
    limits: LimitStatus | None = None


def build_router(
    agents: dict[str, RegisteredAgent],
    store: SessionStore,
    tool_registry: dict[str, ToolEntry] | None = None,
    debug_mode: bool = False,
    limits: Limits | None = None,
    *,
    secret_key: str,
) -> APIRouter:
    router = APIRouter()
    resolve_session = session_dependency(store, secret_key)
    registry = tool_registry or {}

    @router.post("/agents/{name}/chat", response_model=ChatResponse)
    async def chat(name: str, request: ChatRequest, resolved: ResolvedSession = Depends(resolve_session)) -> ChatResponse | JSONResponse:  # noqa: B008 — this is FastAPI's own DI idiom, not a mutable-default bug
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")

        session = resolved.session

        # limits=None means the consumer didn't opt in - no check, no field on the response,
        # unchanged behavior from before this feature existed.
        limit_status: LimitStatus | None = None
        if limits is not None:
            allowed, limit_status, updated = evaluate_and_consume(session.data.get("limits", {}), limits, datetime.now(UTC))
            if not allowed:
                # No agent.run(), no store.save() - an over-limit message costs nothing. Returned
                # as a plain JSONResponse (not ChatResponse) so the body is exactly {limits, session_id},
                # not wrapped in FastAPI's default {"detail": ...} shape.
                return JSONResponse(status_code=429, content={"limits": limit_status.model_dump(), "session_id": resolved.token})
            session.data["limits"] = updated

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
            return ChatResponse(reply=output.text, cards=output.cards or None, session_id=resolved.token, limits=limit_status)
        return ChatResponse(reply=str(output), session_id=resolved.token, limits=limit_status)

    return router
