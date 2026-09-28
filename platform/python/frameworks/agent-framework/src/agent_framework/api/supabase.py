"""Supabase provider: builds a BaseAPIClient pre-configured for Supabase's PostgREST REST API, so
a consumer's ApiToolConfig only needs a table path and schema — not hand-rolled auth headers.

Also a Supabase Auth-backed UserVerifier (core/auth.py) — GET /auth/v1/user with the caller's own
token, not build_supabase_client's fixed service-role client (that call needs the caller's token in
the Authorization header, not the service role key, and no PostgREST base path). A live round trip
to Supabase on every call, by design: the same thing Supabase's own @supabase/ssr middleware does
server-side (confirmed against bestays-web's own middleware.ts, 2026-09-29) - not local JWT
decoding, so an already-revoked or expired token is rejected correctly with no JWT secret/JWKS
handling needed here."""

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
