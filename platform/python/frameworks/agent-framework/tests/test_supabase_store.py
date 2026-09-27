"""Proves SupabaseSessionStore's PostgREST calls: right path, right Accept-/Content-Profile header
per verb, right payload shape - no real Supabase project needed (httpx.MockTransport, same pattern
as test_supabase_client.py)."""

import httpx
from agent_framework.core.store.base import Session
from agent_framework.core.store.supabase_store import SupabaseSessionStore
from pydantic_ai import ModelRequest, ModelResponse, TextPart, UserPromptPart

_SCHEMA = "acme_agent_framework"


async def test_get_or_create_mints_a_fresh_session_when_none_exists() -> None:
    posted = {}

    def handler(request: httpx.Request) -> httpx.Response:
        if request.method == "GET":
            assert request.headers["accept-profile"] == _SCHEMA
            assert "id=eq.s1" in str(request.url)
            return httpx.Response(200, json=[])
        assert request.method == "POST"
        assert request.headers["content-profile"] == _SCHEMA
        posted["body"] = request.read()
        return httpx.Response(201)

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    session = await store.get_or_create("s1")

    assert session.id == "s1"
    assert session.messages == []
    assert b'"id":"s1"' in posted["body"]
    await store.close()


async def test_get_or_create_returns_the_existing_row() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "GET"
        return httpx.Response(
            200,
            json=[
                {
                    "id": "s1",
                    "user_id": "user-123",
                    "messages": [{"parts": [{"content": "hi", "timestamp": "2026-01-01T00:00:00Z", "part_kind": "user-prompt"}], "kind": "request"}],
                    "data": {"contact_name": "Andrei"},
                }
            ],
        )

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    session = await store.get_or_create("s1")

    assert session.user_id == "user-123"
    assert session.data == {"contact_name": "Andrei"}
    assert len(session.messages) == 1
    await store.close()


async def test_save_upserts_with_merge_duplicates() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "POST"
        assert request.headers["content-profile"] == _SCHEMA
        assert "resolution=merge-duplicates" in request.headers["prefer"]
        return httpx.Response(201)

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    session = Session(
        id="s1",
        messages=[ModelRequest(parts=[UserPromptPart(content="hi")]), ModelResponse(parts=[TextPart(content="hello")])],
    )
    await store.save(session)
    await store.close()


async def test_link_user_rotates_the_id_and_deletes_the_old_row() -> None:
    calls: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        calls.append(request.method)
        if request.method == "GET":
            # Models realistic usage: by the time link_user runs, the session-resolution
            # middleware has already created this row earlier in the same request.
            return httpx.Response(200, json=[{"id": "s1", "user_id": None, "messages": [], "data": {}}])
        if request.method == "DELETE":
            assert "id=eq.s1" in str(request.url)
        return httpx.Response(201)

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    linked = await store.link_user("s1", "user-123")

    assert linked.id != "s1"
    assert linked.user_id == "user-123"
    assert calls == ["GET", "POST", "DELETE"]
    await store.close()
