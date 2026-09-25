"""Integration test against a real postgres — the chat-api-postgres bundle
(processos-workspace/records/sbx-sdlc-kit/infrastructure/service-bundle/chat-api-postgres.yaml).
Skips cleanly if that's not running (e.g. CI without docker) rather than failing the suite."""

import os
import uuid

import asyncpg
import pytest
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


async def test_link_user_persists(store: PostgresStore) -> None:
    session_id = f"test-{uuid.uuid4()}"
    await store.get_or_create(session_id)
    await store.link_user(session_id, "user-123")

    reloaded = await store.get_or_create(session_id)
    assert reloaded.user_id == "user-123"
