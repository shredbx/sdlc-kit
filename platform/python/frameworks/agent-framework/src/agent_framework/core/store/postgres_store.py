"""Postgres-backed SessionStore — real persistence behind the same interface as MemoryStore.
Messages are stored as JSONB via PydanticAI's own ModelMessagesTypeAdapter, not a hand-rolled
serialization — the same message shape the agent already produces, no translation layer. `data`
(the free-form per-session fact bag) is plain JSONB via stdlib `json` — it's just a dict, no
PydanticAI-specific shape to preserve.

Schema is owned by migrate.py's numbered `.sql` files, not inline DDL here — see that module for
why. The connection pool is created lazily on first use, not in __init__: asyncpg pools are bound
to the event loop they're created in, and there's no running loop yet at plain module-import time
(where a consumer's main.py constructs this). Lazy creation means PostgresStore(dsn) stays a
trivial, synchronous constructor — the pool is built, and migrations applied, the first time a
real request needs it, inside whatever loop is actually running by then."""

import json
import uuid

import asyncpg
from pydantic_ai import ModelMessagesTypeAdapter

from agent_framework.core.store.base import Session, SessionStore
from agent_framework.core.store.migrate import run_migrations


class PostgresStore(SessionStore):
    def __init__(self, dsn: str) -> None:
        self._dsn = dsn
        self._pool: asyncpg.Pool | None = None

    async def _get_pool(self) -> asyncpg.Pool:
        if self._pool is None:
            self._pool = await asyncpg.create_pool(self._dsn)
            await run_migrations(self._pool)
        return self._pool

    async def migrate(self) -> None:
        """Explicitly ensures the schema is up to date. Idempotent, and this is exactly what
        happens automatically the first time the store is used — this only matters if you want
        migrations to run as their own explicit step (e.g. a deploy pipeline, or initializing a
        brand new database before the app's first boot) rather than implicitly on first request.
        See chat-api-python/scripts/migrate_db.py for that standalone use."""
        await self._get_pool()

    async def get_or_create(self, session_id: str) -> Session:
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            row = await conn.fetchrow("SELECT id, user_id, messages, data FROM sessions WHERE id = $1", session_id)
            if row is None:
                await conn.execute("INSERT INTO sessions (id) VALUES ($1)", session_id)
                return Session(id=session_id)
            messages = ModelMessagesTypeAdapter.validate_json(row["messages"] or "[]")
            data = json.loads(row["data"] or "{}")
            return Session(id=row["id"], user_id=row["user_id"], messages=messages, data=data)

    async def save(self, session: Session) -> None:
        messages_json = ModelMessagesTypeAdapter.dump_json(session.messages).decode("utf-8")
        data_json = json.dumps(session.data)
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO sessions (id, user_id, messages, data, updated_at)
                VALUES ($1, $2, $3::jsonb, $4::jsonb, now())
                ON CONFLICT (id) DO UPDATE SET user_id = $2, messages = $3::jsonb, data = $4::jsonb, updated_at = now()
                """,
                session.id,
                session.user_id,
                messages_json,
                data_json,
            )

    async def link_user(self, session_id: str, user_id: str) -> Session:
        old = await self.get_or_create(session_id)
        new = Session(id=str(uuid.uuid4()), user_id=user_id, messages=old.messages, data=old.data)
        await self.save(new)
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            await conn.execute("DELETE FROM sessions WHERE id = $1", session_id)
        return new

    async def close(self) -> None:
        if self._pool is not None:
            await self._pool.close()
