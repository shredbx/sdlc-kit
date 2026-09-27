"""Proves build_supabase_client wires the right base URL and PostgREST auth headers — no real
Supabase project or network needed."""

import httpx
from agent_framework.api.supabase import build_supabase_client


def _mock_transport() -> httpx.MockTransport:
    def handler(request: httpx.Request) -> httpx.Response:
        assert str(request.url) == "https://example.supabase.co/rest/v1/properties"
        assert request.headers["apikey"] == "test-key"
        assert request.headers["authorization"] == "Bearer test-key"
        assert request.headers["content-type"] == "application/json"
        return httpx.Response(200, json=[])

    return httpx.MockTransport(handler)


async def test_build_supabase_client_sets_base_url_and_auth_headers() -> None:
    client = build_supabase_client(url="https://example.supabase.co/", key="test-key", transport=_mock_transport())
    response = await client.request("GET", "/properties")
    assert response.json() == []
    await client.aclose()
