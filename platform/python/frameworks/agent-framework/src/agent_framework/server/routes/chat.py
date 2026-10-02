"""Generic chat endpoints: POST /agents/{name}/chat (one JSON reply) and POST
/agents/{name}/chat/stream (the same turn as Server-Sent Events) - work for any RegisteredAgent the
registry discovered. No per-agent route code needed; a new agent folder is automatically served
here. Both routes share the same pre-run steps (agent lookup, client_context, auth, limits) and the
same post-run steps (save the session, add up token usage)."""

import json
import time
from collections.abc import AsyncIterator
from contextlib import nullcontext
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any

import logfire
from fastapi import APIRouter, Depends, Header, HTTPException
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel
from pydantic_ai import AgentRunResult, AgentRunResultEvent

from agent_framework.core.auth import UserVerifier
from agent_framework.core.cards import AgentReply, Card, flatten_cards_to_text
from agent_framework.core.debug_model import build_debug_model
from agent_framework.core.limits import Limits, LimitStatus, evaluate_and_consume
from agent_framework.core.store.base import Session, SessionStore
from agent_framework.core.tool_registry import ToolEntry
from agent_framework.core.types import RegisteredAgent
from agent_framework.server.middleware.session import ResolvedSession, session_dependency, sign_session_token
from agent_framework.server.model_errors import INTERNAL, ModelFailure, classify_model_error
from agent_framework.server.streaming import frames_for, sse
from agent_framework.server.turn_log import log_turn, turn_record
from agent_framework.server.usage import Pricing, TokenUsage, add_turn_usage, summarize

_MAX_CONTEXT_CHARS = 4096


class ChatRequest(BaseModel):
    message: str
    debug: bool = False
    # Opaque client-supplied state (e.g. "which page is the visitor currently on") - the framework
    # never interprets this, just stashes it at session.data["client_context"] (below) for the
    # consumer's own dynamic-instructions hook (agent.instructions(...)) to read and give meaning
    # to. Sent fresh on every request, not merged/accumulated - a consumer that wants a field to
    # persist once seen should copy it into its OWN typed context (e.g. session.data["context"]),
    # not rely on this surviving past the request that sent it.
    client_context: dict[str, Any] | None = None


class ChatResponse(BaseModel):
    reply: str
    cards: list[Card] | None = None
    session_id: str
    limits: LimitStatus | None = None


@dataclass
class _Turn:
    """What the pre-run steps resolved for one request."""

    registered: RegisteredAgent
    session: Session
    session_token: str
    limit_status: LimitStatus | None


def build_router(
    agents: dict[str, RegisteredAgent],
    store: SessionStore,
    tool_registry: dict[str, ToolEntry] | None = None,
    debug_mode: bool = False,
    limits: Limits | None = None,
    user_verifier: UserVerifier | None = None,
    *,
    secret_key: str,
    pricing: Pricing | None = None,
    # Size caps, checked before anything else: None = no cap on the message. The context cap is fixed -
    # client_context is a few small fields (a name, a source, a link), and it is saved on the session.
    max_message_chars: int | None = None,
) -> APIRouter:
    router = APIRouter()
    resolve_session = session_dependency(store, secret_key)
    registry = tool_registry or {}

    async def begin_turn(name: str, request: ChatRequest, resolved: ResolvedSession, authorization: str | None) -> _Turn | JSONResponse:
        """Everything before the model runs. Raises HTTPException (404/401/500) or returns the 429
        JSONResponse - in both cases nothing has been sent to the model and nothing is saved."""
        registered = agents.get(name)
        if registered is None:
            raise HTTPException(status_code=404, detail=f"no agent named {name!r}")

        session = resolved.session
        session_token = resolved.token

        too_big = _size_problem(request, max_message_chars)
        if too_big is not None:
            return _failure_response(too_big, session_token)

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

        # Raw passthrough, overwritten (not merged) every request - see ChatRequest.client_context's
        # own comment for why staleness across turns is the consumer's problem to solve, not this
        # route's. Set AFTER the auth step above: linking a session to a user can hand back a different
        # Session object (a store may reload it), and a value set on the old one would silently be lost
        # on exactly the first request of every signed-in conversation.
        if request.client_context is not None:
            session.data["client_context"] = request.client_context

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

        return _Turn(registered=registered, session=session, session_token=session_token, limit_status=limit_status)

    async def complete_turn(turn: _Turn, result: AgentRunResult[Any]) -> tuple[str, list[Card] | None, TokenUsage]:
        """Everything after a run that SUCCEEDED: persist the history and the running token totals,
        and shape the output. A failed run never reaches this, so a failed turn leaves the session
        exactly as it was and re-sending the same message starts from the same point."""
        run_usage = result.usage
        turn_usage = TokenUsage(
            requests=run_usage.requests,
            input_tokens=run_usage.input_tokens,
            output_tokens=run_usage.output_tokens,
            tool_calls=run_usage.tool_calls,
        )
        turn.session.messages = result.all_messages()
        turn.session.data["usage"] = add_turn_usage(turn.session.data.get("usage", {}), turn_usage)
        await store.save(turn.session)
        text, cards = _render_output(turn.registered, result.output)
        return text, cards, turn_usage

    def record_turn(turn: _Turn, *, started: float, streamed: bool, status: str, code: str | None = None, usage: TokenUsage | None = None) -> None:
        log_turn(
            turn_record(
                agent=turn.registered.name,
                session_id=turn.session.id,
                user_id=turn.session.user_id,
                client_context=turn.session.data.get("client_context"),
                streamed=streamed,
                latency_seconds=time.monotonic() - started,
                status=status,
                code=code,
                usage=usage,
                pricing=pricing,
            )
        )

    @router.post("/agents/{name}/chat", response_model=ChatResponse)
    async def chat(
        name: str,
        request: ChatRequest,
        resolved: ResolvedSession = Depends(resolve_session),  # noqa: B008 — this is FastAPI's own DI idiom, not a mutable-default bug
        authorization: str | None = Header(default=None),
    ) -> ChatResponse | JSONResponse:
        begun = await begin_turn(name, request, resolved, authorization)
        if isinstance(begun, JSONResponse):
            return begun
        turn = begun
        registered, session = turn.registered, turn.session

        # Both the server (debug_mode) and the request (request.debug) must opt in - neither
        # alone parses a message as a command, so a real user's text is never misread as one.
        override = registered.agent.override(model=build_debug_model(registry)) if debug_mode and request.debug else nullcontext()

        deps = registered.build_deps(session)
        started = time.monotonic()
        # A no-op span attribute when logfire isn't configured (agent-framework always imports it -
        # it's a bundled part of the full pydantic-ai package, not a separate install) - tags every
        # model/tool span nested under it with the real user id once a session is authenticated, so
        # a Logfire trace for an identified user (e.g. the Chrome extension) is filterable by who
        # was actually chatting, not just which session. Untouched (no span) for a public session -
        # nothing meaningful to tag an anonymous visitor with.
        with logfire.span("chat", user_id=session.user_id) if session.user_id else nullcontext():
            with override:
                try:
                    result = await registered.agent.run(request.message, deps=deps, message_history=session.messages)
                except Exception as exc:
                    # A model/network failure (rate limit, 503 "high demand", bad key, connection
                    # error, timeout, ...) becomes a real HTTP error with a machine-readable code
                    # (server/model_errors.py) - never a 200 that looks like a reply. Anything that
                    # is not one (a bug in a tool) is re-raised: the caller still gets a plain 500.
                    # message_history/session are left untouched either way, so re-sending the same
                    # message starts from the same point.
                    failure = classify_model_error(exc)
                    if failure is None:
                        raise
                    _log_failure(failure, exc)
                    record_turn(turn, started=started, streamed=False, status="error", code=failure.code)
                    return _failure_response(failure, turn.session_token)

        text, cards, turn_usage = await complete_turn(turn, result)
        record_turn(turn, started=started, streamed=False, status="ok", usage=turn_usage)
        return ChatResponse(reply=text, cards=cards, session_id=turn.session_token, limits=turn.limit_status)

    @router.post("/agents/{name}/chat/stream", response_model=None)
    async def chat_stream(
        name: str,
        request: ChatRequest,
        resolved: ResolvedSession = Depends(resolve_session),  # noqa: B008 — same FastAPI DI idiom as above
        authorization: str | None = Header(default=None),
    ) -> StreamingResponse | JSONResponse:
        if request.debug:
            # Debug commands (`tool:name {...}`, no model) have nothing to stream - they stay on /chat.
            raise HTTPException(status_code=400, detail="debug commands are only available on /agents/{name}/chat, not /chat/stream")
        begun = await begin_turn(name, request, resolved, authorization)
        if isinstance(begun, JSONResponse):
            return begun
        return StreamingResponse(
            _event_stream(begun, request.message),
            media_type="text/event-stream",
            # no-cache: never serve a stale stream; X-Accel-Buffering: tell nginx-style proxies not
            # to hold the response back until it ends (the whole point of streaming).
            headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
        )

    async def _event_stream(turn: _Turn, message: str) -> AsyncIterator[str]:
        registered, session = turn.registered, turn.session
        deps = registered.build_deps(session)
        started = time.monotonic()
        outcome, code, turn_usage = "cancelled", None, None  # what the turn record says unless the run finishes or fails first
        try:
            with logfire.span("chat", user_id=session.user_id) if session.user_id else nullcontext():
                # The run happens in a background task that is cancelled when this block exits - on
                # normal completion, on an error, or when the client disconnects (Starlette cancels
                # this generator). complete_turn() below is only reached for a finished run, so a
                # cancelled or failed turn is never saved.
                async with registered.agent.run_stream_events(message, deps=deps, message_history=session.messages) as events:
                    async for event in events:
                        if isinstance(event, AgentRunResultEvent):
                            text, cards, turn_usage = await complete_turn(turn, event.result)
                            outcome = "ok"
                            yield sse("done", _done_payload(turn, text, cards, turn_usage, pricing))
                            return
                        for frame_name, data in frames_for(event):
                            yield sse(frame_name, data)
        except Exception as exc:
            failure = classify_model_error(exc)
            if failure is None:
                logfire.exception("chat stream failed")
                failure = INTERNAL
            else:
                _log_failure(failure, exc)
            outcome, code = "error", failure.code
            yield sse("error", {"code": failure.code, "retryable": failure.retryable, "detail": failure.message, "session_id": turn.session_token})
        finally:
            record_turn(turn, started=started, streamed=True, status=outcome, code=code, usage=turn_usage)

    return router


def _done_payload(turn: _Turn, text: str, cards: list[Card] | None, turn_usage: TokenUsage, pricing: Pricing | None) -> dict[str, Any]:
    return {
        "reply": text,
        "cards": [card.model_dump(mode="json") for card in cards] if cards else None,
        "session_id": turn.session_token,
        "limits": turn.limit_status.model_dump(mode="json") if turn.limit_status else None,
        "usage": summarize(turn.session.data["usage"], turn_usage, pricing).model_dump(mode="json"),
        "state": _state_summary(turn.registered, turn.session),
    }


def _state_summary(registered: RegisteredAgent, session: Session) -> dict[str, Any] | None:
    if registered.state_summary is None:
        return None
    try:
        return registered.state_summary(session)
    except Exception:
        # A bug in the consumer's summary must not cost the customer their reply.
        logfire.exception("state_summary failed", agent=registered.name)
        return None


def _render_output(registered: RegisteredAgent, output: object) -> tuple[str, list[Card] | None]:
    if isinstance(output, AgentReply):
        text = output.text
        cards = output.cards or None
        # A plain_text channel (core/types.py's OutputProtocol) can't render Card objects at
        # all - fold them into the text instead of silently dropping them, and don't also hand
        # back raw cards a consumer explicitly can't use.
        if registered.output_protocol == "plain_text" and cards:
            return f"{text}\n\n{flatten_cards_to_text(cards)}", None
        return text, cards
    return str(output), None


def _size_problem(request: ChatRequest, max_message_chars: int | None) -> ModelFailure | None:
    if max_message_chars is not None and len(request.message) > max_message_chars:
        return ModelFailure(
            status=413,
            code="message_too_long",
            retryable=False,
            message=f"The message is too long (more than {max_message_chars} characters). Shorten it and send again.",
        )
    if request.client_context is not None and len(json.dumps(request.client_context)) > _MAX_CONTEXT_CHARS:
        return ModelFailure(status=413, code="context_too_large", retryable=False, message="The request's context is too large.")
    return None


def _failure_response(failure: ModelFailure, session_token: str) -> JSONResponse:
    headers = {"Retry-After": str(failure.retry_after)} if failure.retry_after is not None else None
    return JSONResponse(
        status_code=failure.status,
        content={"detail": failure.message, "code": failure.code, "retryable": failure.retryable, "session_id": session_token},
        headers=headers,
    )


def _log_failure(failure: ModelFailure, exc: BaseException) -> None:
    # The provider's own detail stays in the logs only - never in the response.
    logfire.error("model call failed", code=failure.code, status=failure.status, error_type=type(exc).__name__, error=str(exc)[:500])


def _bearer_token(authorization: str | None) -> str | None:
    if not authorization or not authorization.lower().startswith("bearer "):
        return None
    return authorization[7:].strip() or None
