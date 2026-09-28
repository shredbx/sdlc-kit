"""Generic chat endpoint: POST /agents/{name}/chat — works for any RegisteredAgent the registry
discovered. No per-agent route code needed; a new agent folder is automatically served here."""

from contextlib import nullcontext
from datetime import UTC, datetime

import logfire
from fastapi import APIRouter, Depends, Header, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from agent_framework.core.auth import UserVerifier
from agent_framework.core.cards import AgentReply, Card, flatten_cards_to_text
from agent_framework.core.debug_model import build_debug_model
from agent_framework.core.limits import Limits, LimitStatus, evaluate_and_consume
from agent_framework.core.store.base import SessionStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.middleware.session import ResolvedSession, session_dependency, sign_session_token


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
    user_verifier: UserVerifier | None = None,
    *,
    secret_key: str,
) -> APIRouter:
    router = APIRouter()
    resolve_session = session_dependency(store, secret_key)
    registry = tool_registry or {}

    @router.post("/agents/{name}/chat", response_model=ChatResponse)
    async def chat(
        name: str,
        request: ChatRequest,
        resolved: ResolvedSession = Depends(resolve_session),  # noqa: B008 — this is FastAPI's own DI idiom, not a mutable-default bug
        authorization: str | None = Header(default=None),
    ) -> ChatResponse | JSONResponse:
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")

        session = resolved.session
        session_token = resolved.token

        if registered.auth == "required":
            if user_verifier is None:
                raise HTTPException(status_code=500, detail=f"agent {name!r} requires auth but no user_verifier is configured")
            token = _bearer_token(authorization)
            user_id = await user_verifier(token) if token else None
            if user_id is None:
                raise HTTPException(status_code=401, detail="a valid Authorization: Bearer <token> is required")
            if session.user_id != user_id:
                # A privilege-level change (first time this session is linked, or a different user
                # than before) - store.link_user() unconditionally rotates the session id (OWASP
                # Session Management Cheat Sheet, see its own docstring), so the response has to
                # carry a freshly signed token for the NEW id, not resolved.token.
                session = await store.link_user(session.id, user_id)
                session_token = sign_session_token(secret_key, session.id)

        # limits=None means the consumer didn't opt in - no check, no field on the response,
        # unchanged behavior from before this feature existed.
        limit_status: LimitStatus | None = None
        if limits is not None:
            allowed, limit_status, updated = evaluate_and_consume(session.data.get("limits", {}), limits, datetime.now(UTC))
            if not allowed:
                # No agent.run(), no store.save() - an over-limit message costs nothing. Returned
                # as a plain JSONResponse (not ChatResponse) so the body is exactly {limits, session_id},
                # not wrapped in FastAPI's default {"detail": ...} shape.
                return JSONResponse(status_code=429, content={"limits": limit_status.model_dump(), "session_id": session_token})
            session.data["limits"] = updated

        # Both the server (debug_mode) and the request (request.debug) must opt in - neither
        # alone parses a message as a command, so a real user's text is never misread as one.
        override = registered.agent.override(model=build_debug_model(registry)) if debug_mode and request.debug else nullcontext()

        deps = registered.build_deps(session)
        # A no-op span attribute when logfire isn't configured (agent-framework always imports it -
        # it's a bundled part of the full pydantic-ai package, not a separate install) - tags every
        # model/tool span nested under it with the real user id once a session is authenticated, so
        # a Logfire trace for an identified user (e.g. the Chrome extension) is filterable by who
        # was actually chatting, not just which session. Untouched (no span) for a public session -
        # nothing meaningful to tag an anonymous visitor with.
        with logfire.span("chat", user_id=session.user_id) if session.user_id else nullcontext():
            with override:
                result = await registered.agent.run(request.message, deps=deps, message_history=session.messages)
        session.messages = result.all_messages()
        await store.save(session)
        output = result.output
        if isinstance(output, AgentReply):
            text = output.text
            cards = output.cards or None
            # A plain_text channel (core/types.py's OutputProtocol) can't render Card objects at
            # all - fold them into the text instead of silently dropping them, and don't also hand
            # back raw cards a consumer explicitly can't use.
            if registered.output_protocol == "plain_text" and cards:
                text = f"{text}\n\n{flatten_cards_to_text(cards)}"
                cards = None
            return ChatResponse(reply=text, cards=cards, session_id=session_token, limits=limit_status)
        return ChatResponse(reply=str(output), session_id=session_token, limits=limit_status)

    return router


def _bearer_token(authorization: str | None) -> str | None:
    if not authorization or not authorization.lower().startswith("bearer "):
        return None
    return authorization[7:].strip() or None
