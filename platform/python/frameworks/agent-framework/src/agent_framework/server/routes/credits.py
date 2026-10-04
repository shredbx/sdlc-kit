"""GET /account/credits - what the model key has spent, for the signed-in apps to show. Any signed-in user may read it
(the same login the chat routes ask for); it answers `{"supported": false}` when this deployment has no provider that
reports usage, so a client simply shows nothing rather than handling an error. The provider key never leaves the
server. Mounted only when a user verifier exists - without one there is no way to tell who is asking."""

from typing import Any

from fastapi import APIRouter, Header, HTTPException
from fastapi.responses import JSONResponse

from agent_framework.core.auth import AuthUnavailable, UserVerifier
from agent_framework.core.providers.credits import CreditsSource, CreditsUnavailable
from agent_framework.server.bearer import bearer_token
from agent_framework.server.model_errors import AUTH_UNAVAILABLE

_NO_STORE = {"Cache-Control": "no-store"}  # the figures are for the signed-in user only: no shared cache keeps them


def build_credits_router(credits: CreditsSource | None, user_verifier: UserVerifier) -> APIRouter:
    router = APIRouter()

    @router.get("/account/credits", response_model=None)
    async def get_credits(authorization: str | None = Header(default=None)) -> Any:
        token = bearer_token(authorization)
        try:
            user_id = await user_verifier(token) if token else None
        except AuthUnavailable:
            body = {"detail": AUTH_UNAVAILABLE.message, "code": AUTH_UNAVAILABLE.code, "retryable": AUTH_UNAVAILABLE.retryable}
            return JSONResponse(body, status_code=AUTH_UNAVAILABLE.status, headers=_NO_STORE)
        if user_id is None:
            raise HTTPException(status_code=401, detail="a valid Authorization: Bearer <token> is required")
        if credits is None:
            return JSONResponse({"supported": False}, headers=_NO_STORE)
        try:
            reading = await credits.current()
        except CreditsUnavailable:
            return JSONResponse(
                {"detail": "The usage figures are not available right now.", "code": "credits_unavailable", "retryable": True},
                status_code=503,
                headers=_NO_STORE,
            )
        return JSONResponse(reading.model_dump(), headers=_NO_STORE)

    return router
