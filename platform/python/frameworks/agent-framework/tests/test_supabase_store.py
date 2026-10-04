"""Proves SupabaseSessionStore's PostgREST calls: right path, right Accept-/Content-Profile header
per verb, right payload shape - no real Supabase project needed (httpx.MockTransport, same pattern
as test_supabase_client.py)."""

import json
from datetime import datetime

import httpx
from agent_framework.core.store.base import Session, rotate_for_user
from agent_framework.core.store.supabase_store import SupabaseSessionStore
from pydantic_ai import ModelRequest, ModelResponse, TextPart, UserPromptPart

_SCHEMA = "acme_agent_framework"


async def test_get_or_create_on_a_miss_returns_a_fresh_session_and_writes_nothing() -> None:
    methods: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        methods.append(request.method)
        assert request.headers["accept-profile"] == _SCHEMA
        assert "id=eq.s1" in str(request.url)
        return httpx.Response(200, json=[])

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    session = await store.get_or_create("s1")

    assert session.id == "s1"
    assert session.messages == []
    assert methods == ["GET"]  # no insert: the first save creates the row
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


async def test_save_upserts_with_merge_duplicates_and_stamps_the_time() -> None:
    bodies: list[dict] = []

    def handler(request: httpx.Request) -> httpx.Response:
        assert request.method == "POST"
        assert request.headers["content-profile"] == _SCHEMA
        assert "resolution=merge-duplicates" in request.headers["prefer"]
        bodies.append(json.loads(request.read()))
        return httpx.Response(201)

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    session = Session(
        id="s1",
        messages=[ModelRequest(parts=[UserPromptPart(content="hi")]), ModelResponse(parts=[TextPart(content="hello")])],
    )
    await store.save(session)
    await store.close()

    assert len(bodies) == 1 and bodies[0]["id"] == "s1"
    assert datetime.fromisoformat(bodies[0]["updated_at"]).tzinfo is not None  # the upsert would otherwise keep the first save's time


async def test_save_of_a_rotated_session_writes_the_new_row_then_deletes_the_old_one_and_nothing_else() -> None:
    calls: list[tuple[str, str]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        calls.append((request.method, str(request.url.params.get("id", ""))))
        return httpx.Response(201)

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    rotated = rotate_for_user(Session(id="s1", data={"k": 1}), "user-123")
    await store.save(rotated)
    await store.save(rotated)  # a second save in the same request must not delete again

    assert calls == [("POST", ""), ("DELETE", "eq.s1"), ("POST", "")]
    assert rotated.replaces is None
    await store.close()


async def test_a_failed_save_deletes_nothing() -> None:
    calls: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        calls.append(request.method)
        return httpx.Response(500)

    store = SupabaseSessionStore("https://x.supabase.co", "key", schema=_SCHEMA, transport=httpx.MockTransport(handler))
    rotated = rotate_for_user(Session(id="s1", data={"k": 1}), "user-123")
    try:
        await store.save(rotated)
    except Exception:  # noqa: BLE001 - whichever error the client raises for a 500
        pass

    assert "DELETE" not in calls and rotated.replaces == "s1"
    await store.close()
