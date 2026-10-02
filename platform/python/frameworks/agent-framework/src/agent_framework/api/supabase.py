"""Supabase provider: builds a BaseAPIClient pre-configured for Supabase's PostgREST REST API, so
a consumer's ApiToolConfig only needs a table path and schema — not hand-rolled auth headers.

Also a Supabase Auth-backed UserVerifier (core/auth.py): an asymmetric token is verified locally against the project's key set (api/jwt_keys.py),
a legacy HS256 token by asking Supabase Auth (GET /auth/v1/user with the caller's own token, not build_supabase_client's fixed service-role client).
Which path a token takes is read from the token's own `alg` header, so no Supabase setting needs to change - a project that signs with asymmetric
keys gets zero Auth calls per request, one that does not gets one call over a shared connection.

supabase_password_grant/supabase_refresh_grant back server/routes/auth.py's sign-in proxy — a
distributed client (a Chrome extension, a mobile app) talks ONLY to this app's own base URL, never
learns the Supabase project URL or anon key at all. Both are a pure, verbatim passthrough (same
request/response shape Supabase's own /auth/v1/token returns, same status code) - the caller's own
error-parsing logic never needs to know a proxy is even involved."""

from typing import Any

import httpx
import jwt

from agent_framework.api.base import BaseAPIClient
from agent_framework.api.jwt_keys import LOCAL_ALGORITHMS, KeySetVerifier
from agent_framework.core.auth import AuthUnavailable, UserVerifier


def build_supabase_client(url: str, key: str, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None) -> BaseAPIClient:
    return BaseAPIClient(
        base_url=f"{url.rstrip('/')}/rest/v1",
        headers={"apikey": key, "Authorization": f"Bearer {key}", "Content-Type": "application/json"},
        timeout=timeout,
        transport=transport,
    )


def build_supabase_user_verifier(
    url: str, anon_key: str, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None, *, key_refresh_seconds: float = 600
) -> UserVerifier:
    """A token signed with an asymmetric key is checked here against the project's published key set (api/jwt_keys.py: no network per request); a
    legacy HS256 token is checked by Supabase Auth itself - the way Supabase recommends for a shared-secret project, and the only way to see a revoked
    session at once. Both use one keep-alive client. Anything else (an unreadable token, `none`) is invalid without a call."""
    base = url.rstrip("/")
    endpoint = f"{base}/auth/v1/user"
    client = httpx.AsyncClient(timeout=timeout, transport=transport)
    keys = KeySetVerifier(
        f"{base}/auth/v1/.well-known/jwks.json", audience="authenticated", client=client, headers={"apikey": anon_key}, refresh_seconds=key_refresh_seconds
    )

    async def ask_auth_server(token: str) -> str | None:
        try:
            response = await client.get(endpoint, headers={"apikey": anon_key, "Authorization": f"Bearer {token}"})
        except httpx.HTTPError as exc:  # unreachable or timed out: not a verdict on the token
            raise AuthUnavailable("Supabase Auth could not be reached") from exc
        if response.status_code == 429 or response.status_code >= 500:
            raise AuthUnavailable(f"Supabase Auth answered {response.status_code}")
        if response.status_code != 200:
            return None
        user_id = response.json().get("id")
        return user_id if isinstance(user_id, str) else None

    async def verify(token: str) -> str | None:
        try:
            algorithm = jwt.get_unverified_header(token).get("alg")
        except jwt.PyJWTError:
            return None
        if algorithm in LOCAL_ALGORITHMS:
            return await keys.user_id(token)
        if algorithm == "HS256":
            return await ask_auth_server(token)
        return None

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
