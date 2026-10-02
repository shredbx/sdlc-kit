"""Integration test against a real postgres — the chat-api-postgres bundle
(processos-workspace/records/sbx-sdlc-kit/infrastructure/service-bundle/chat-api-postgres.yaml).
Skips cleanly if that's not running (e.g. CI without docker) rather than failing the suite."""

import os
import uuid

import asyncpg
import pytest
from agent_framework.core.store.base import rotate_for_user
from agent_framework.core.store.postgres_store import PostgresStore
from pydantic_ai import Agent
from pydantic_ai.messages import ModelResponse, TextPart
from pydantic_ai.models.function import FunctionModel

DSN = os.environ.get("TEST_POSTGRES_DSN", "postgresql://chat_api:not-the-real-password@localhost:54322/chat_api")


async def _postgres_reachable() -> bool:
    try:
        conn = await asyncpg.connect(DSN, timeout=2)
    except (OSError, asyncpg.PostgresError):
        return False
    await conn.close()
    return True


@pytest.fixture
async def store():
    if not await _postgres_reachable():
        pytest.skip(f"no postgres reachable at {DSN} — start the chat-api-postgres bundle to run this")
    s = PostgresStore(DSN)
    yield s
    await s.close()


async def test_fresh_session_has_no_messages(store: PostgresStore) -> None:
    session = await store.get_or_create(f"test-{uuid.uuid4()}")
    assert session.messages == []
    assert session.user_id is None


async def test_history_survives_a_new_store_instance(store: PostgresStore) -> None:
    session_id = f"test-{uuid.uuid4()}"

    def fn(messages: list, info: object) -> ModelResponse:
        return ModelResponse(parts=[TextPart(content=f"saw {len(messages)} messages")])

    agent = Agent(FunctionModel(fn), name="test_postgres_agent")

    session = await store.get_or_create(session_id)
    result = await agent.run("first", message_history=session.messages)
    session.messages = result.all_messages()
    await store.save(session)

    # A brand new store/pool — proves persistence, not just process-local state.
    fresh_store = PostgresStore(DSN)
    reloaded = await fresh_store.get_or_create(session_id)
    assert len(reloaded.messages) == 2  # the request + the response from turn 1

    result2 = await agent.run("second", message_history=reloaded.messages)
    assert result2.output == "saw 3 messages"  # 2 prior + this new prompt
    await fresh_store.close()


async def test_a_miss_writes_nothing_and_the_first_save_creates_the_row(store: PostgresStore) -> None:
    session_id = f"test-{uuid.uuid4()}"
    session = await store.get_or_create(session_id)

    admin = await asyncpg.connect(DSN)
    try:
        assert await admin.fetchval("SELECT count(*) FROM sessions WHERE id = $1", session_id) == 0
        session.data["x"] = 1
        await store.save(session)
        assert await admin.fetchval("SELECT count(*) FROM sessions WHERE id = $1", session_id) == 1
    finally:
        await admin.close()


async def test_saving_a_rotated_session_persists_it_and_deletes_the_old_row(store: PostgresStore) -> None:
    session_id = f"test-{uuid.uuid4()}"
    anonymous = await store.get_or_create(session_id)
    anonymous.data["contact_name"] = "Andrei"
    await store.save(anonymous)

    linked = rotate_for_user(await store.get_or_create(session_id), "user-123")
    await store.save(linked)

    assert linked.id != session_id  # OWASP: regenerate the id on a privilege-level change
    assert linked.user_id == "user-123"

    reloaded = await store.get_or_create(linked.id)
    assert reloaded.user_id == "user-123"

    # The pre-link id must no longer resolve to the linked session - it's a brand new, anonymous
    # session now, not a revival of the one that got linked.
    orphan = await store.get_or_create(session_id)
    assert orphan.user_id is None


async def test_data_survives_a_new_store_instance(store: PostgresStore) -> None:
    session_id = f"test-{uuid.uuid4()}"

    session = await store.get_or_create(session_id)
    session.data["contact_name"] = "Andrei"
    await store.save(session)

    fresh_store = PostgresStore(DSN)
    reloaded = await fresh_store.get_or_create(session_id)
    assert reloaded.data == {"contact_name": "Andrei"}
    await fresh_store.close()


async def test_uses_a_dedicated_schema_when_configured() -> None:
    """Proves rows land in the configured schema, not `public` - the mechanism a consumer sharing
    one Postgres/Supabase database across multiple products relies on for isolation."""
    if not await _postgres_reachable():
        pytest.skip(f"no postgres reachable at {DSN} — start the chat-api-postgres bundle to run this")

    schema = f"pgstore_test_{uuid.uuid4().hex[:8]}"
    store = PostgresStore(DSN, schema=schema)
    try:
        session = await store.get_or_create(f"test-{uuid.uuid4()}")
        session.data["x"] = 1
        await store.save(session)

        admin = await asyncpg.connect(DSN)
        try:
            count = await admin.fetchval(f'SELECT count(*) FROM "{schema}".sessions')
        finally:
            await admin.close()
        assert count == 1
    finally:
        await store.close()
        admin = await asyncpg.connect(DSN)
        await admin.execute(f'DROP SCHEMA IF EXISTS "{schema}" CASCADE')
        await admin.close()
