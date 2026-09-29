"""Supabase provider: builds a BaseAPIClient pre-configured for Supabase's PostgREST REST API, so
a consumer's ApiToolConfig only needs a table path and schema — not hand-rolled auth headers.

Also a Supabase Auth-backed UserVerifier (core/auth.py) — GET /auth/v1/user with the caller's own
token, not build_supabase_client's fixed service-role client (that call needs the caller's token in
the Authorization header, not the service role key, and no PostgREST base path). A live round trip
to Supabase on every call, by design: the same thing Supabase's own @supabase/ssr middleware does
server-side (confirmed against bestays-web's own middleware.ts, 2026-09-29) - not local JWT
decoding, so an already-revoked or expired token is rejected correctly with no JWT secret/JWKS
handling needed here.

supabase_password_grant/supabase_refresh_grant back server/routes/auth.py's sign-in proxy — a
distributed client (a Chrome extension, a mobile app) talks ONLY to this app's own base URL, never
learns the Supabase project URL or anon key at all. Both are a pure, verbatim passthrough (same
request/response shape Supabase's own /auth/v1/token returns, same status code) - the caller's own
error-parsing logic never needs to know a proxy is even involved."""

from typing import Any

import httpx

from agent_framework.api.base import BaseAPIClient
from agent_framework.core.auth import UserVerifier


def build_supabase_client(url: str, key: str, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None) -> BaseAPIClient:
    return BaseAPIClient(
        base_url=f"{url.rstrip('/')}/rest/v1",
        headers={"apikey": key, "Authorization": f"Bearer {key}", "Content-Type": "application/json"},
        timeout=timeout,
        transport=transport,
    )


def build_supabase_user_verifier(url: str, anon_key: str, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None) -> UserVerifier:
    endpoint = f"{url.rstrip('/')}/auth/v1/user"

    async def verify(token: str) -> str | None:
        async with httpx.AsyncClient(timeout=timeout, transport=transport) as client:
            response = await client.get(endpoint, headers={"apikey": anon_key, "Authorization": f"Bearer {token}"})
        if response.status_code != 200:
            return None
        user_id = response.json().get("id")
        return user_id if isinstance(user_id, str) else None

    return verify


async def _token_grant(
    url: str, anon_key: str, grant_type: str, body: dict[str, str], timeout: float, transport: httpx.AsyncBaseTransport | None
) -> tuple[int, Any]:
    async with httpx.AsyncClient(timeout=timeout, transport=transport) as client:
        response = await client.post(
            f"{url.rstrip('/')}/auth/v1/token?grant_type={grant_type}",
            headers={"apikey": anon_key, "Content-Type": "application/json"},
            json=body,
        )
    try:
        return response.status_code, response.json()
    except ValueError:
        return response.status_code, {"msg": response.text or f"Supabase returned {response.status_code} with a non-JSON body."}


async def supabase_password_grant(
    url: str, anon_key: str, email: str, password: str, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None
) -> tuple[int, Any]:
    return await _token_grant(url, anon_key, "password", {"email": email, "password": password}, timeout, transport)


async def supabase_refresh_grant(
    url: str, anon_key: str, refresh_token: str, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None
) -> tuple[int, Any]:
    return await _token_grant(url, anon_key, "refresh_token", {"refresh_token": refresh_token}, timeout, transport)
