"""Integration tests for migrate.py against the live chat-api-postgres bundle. Runs the real
migration files inside an isolated Postgres schema per test (created and dropped around it) - a
long-lived, shared dev database (the same one manual browser testing writes real session rows
into) is not a safe place to be dropping/recreating the actual `sessions` table from a test."""

import os
import uuid

import asyncpg
import pytest
from agent_framework.core.store.migrate import run_migrations

DSN = os.environ.get("TEST_POSTGRES_DSN", "postgresql://chat_api:not-the-real-password@localhost:54322/chat_api")


async def _postgres_reachable() -> bool:
    try:
        conn = await asyncpg.connect(DSN, timeout=2)
    except (OSError, asyncpg.PostgresError):
        return False
    await conn.close()
    return True


@pytest.fixture
async def isolated_pool():
    if not await _postgres_reachable():
        pytest.skip(f"no postgres reachable at {DSN} — start the chat-api-postgres bundle to run this")

    schema = f"migrate_test_{uuid.uuid4().hex[:8]}"
    admin = await asyncpg.connect(DSN)
    await admin.execute(f'CREATE SCHEMA "{schema}"')
    await admin.close()

    pool = await asyncpg.create_pool(DSN, server_settings={"search_path": schema})
    yield pool
    await pool.close()

    admin = await asyncpg.connect(DSN)
    await admin.execute(f'DROP SCHEMA "{schema}" CASCADE')
    await admin.close()


async def test_fresh_database_gets_the_full_schema(isolated_pool: asyncpg.Pool) -> None:
    await run_migrations(isolated_pool)

    async with isolated_pool.acquire() as conn:
        applied = {row["id"] for row in await conn.fetch("SELECT id FROM schema_migrations")}
        columns = {
            row["column_name"]
            for row in await conn.fetch("SELECT column_name FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'sessions'")
        }

    assert applied == {"0001_create_sessions_table", "0002_add_session_data_column"}
    assert {"id", "user_id", "messages", "data", "created_at", "updated_at"} <= columns


async def test_running_migrations_twice_is_a_noop(isolated_pool: asyncpg.Pool) -> None:
    await run_migrations(isolated_pool)
    await run_migrations(isolated_pool)  # must not raise, must not double-apply

    async with isolated_pool.acquire() as conn:
        count = await conn.fetchval("SELECT count(*) FROM schema_migrations")
    assert count == 2


async def test_resumes_from_a_partially_migrated_database(isolated_pool: asyncpg.Pool) -> None:
    # Simulate a database that already has migration 0001 applied (the pre-`data`-column shape)
    # but not 0002 - proves resumption applies only what's actually missing.
    async with isolated_pool.acquire() as conn:
        await conn.execute("""
            CREATE TABLE sessions (
                id TEXT PRIMARY KEY, user_id TEXT, messages JSONB NOT NULL DEFAULT '[]'::jsonb,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
            )
        """)
        await conn.execute("CREATE TABLE schema_migrations (id TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())")
        await conn.execute("INSERT INTO schema_migrations (id) VALUES ('0001_create_sessions_table')")

    await run_migrations(isolated_pool)

    async with isolated_pool.acquire() as conn:
        applied = {row["id"] for row in await conn.fetch("SELECT id FROM schema_migrations")}
        has_data_column = await conn.fetchval(
            "SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'sessions' AND column_name = 'data'"
        )

    assert applied == {"0001_create_sessions_table", "0002_add_session_data_column"}
    assert has_data_column == 1
