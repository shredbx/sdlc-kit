"""Sign-in proxy: POST /auth/signin and /auth/refresh, forwarding to Supabase Auth's own
/auth/v1/token grant endpoint server-side. A distributed client (a Chrome extension, a mobile app)
talks ONLY to this app's own base URL - it never learns the Supabase project URL or anon key, and
never ships them in its own build/storage. Public routes (no session/auth required) by design -
this IS how a client gets a token in the first place, the same as any login endpoint.

Pure passthrough: Supabase's own response body and status code are returned verbatim (see
api.supabase.supabase_password_grant/supabase_refresh_grant), so a client's existing
Supabase-response error-parsing logic (e.g. checking `msg`/`error_description`) needs no changes
at all - only the URL it calls changes."""

from fastapi import APIRouter
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from agent_framework.api.supabase import supabase_password_grant, supabase_refresh_grant


class SignInRequest(BaseModel):
    email: str
    password: str


class RefreshRequest(BaseModel):
    refresh_token: str


def build_auth_router(supabase_url: str, supabase_anon_key: str) -> APIRouter:
    router = APIRouter()

    @router.post("/auth/signin")
    async def signin(request: SignInRequest) -> JSONResponse:
        status, body = await supabase_password_grant(supabase_url, supabase_anon_key, request.email, request.password)
        return JSONResponse(status_code=status, content=body)

    @router.post("/auth/refresh")
    async def refresh(request: RefreshRequest) -> JSONResponse:
        status, body = await supabase_refresh_grant(supabase_url, supabase_anon_key, request.refresh_token)
        return JSONResponse(status_code=status, content=body)

    return router
