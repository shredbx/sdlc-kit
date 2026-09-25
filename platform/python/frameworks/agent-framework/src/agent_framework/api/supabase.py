"""Supabase provider: builds a BaseAPIClient pre-configured for Supabase's PostgREST REST API, so
a consumer's ApiToolConfig only needs a table path and schema — not hand-rolled auth headers."""

import httpx

from agent_framework.api.base import BaseAPIClient


def build_supabase_client(url: str, key: str, timeout: float = 10.0, transport: httpx.AsyncBaseTransport | None = None) -> BaseAPIClient:
    return BaseAPIClient(
        base_url=f"{url.rstrip('/')}/rest/v1",
        headers={"apikey": key, "Authorization": f"Bearer {key}", "Content-Type": "application/json"},
        timeout=timeout,
        transport=transport,
    )
