"""Postgres-backed SessionStore — real persistence behind the same interface as MemoryStore.
Messages are stored as JSONB via PydanticAI's own ModelMessagesTypeAdapter, not a hand-rolled
serialization — the same message shape the agent already produces, no translation layer.

The connection pool is created lazily on first use, not in __init__: asyncpg pools are bound to
the event loop they're created in, and there's no running loop yet at plain module-import time
(where a consumer's main.py constructs this). Lazy creation means PostgresStore(dsn) stays a
trivial, synchronous constructor — the pool is built the first time a real request needs it,
inside whatever loop is actually running by then."""

import asyncpg
from pydantic_ai import ModelMessagesTypeAdapter

from agent_framework.core.store.base import Session, SessionStore

_CREATE_TABLE = """
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    messages JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)
"""


class PostgresStore(SessionStore):
    def __init__(self, dsn: str) -> None:
        self._dsn = dsn
        self._pool: asyncpg.Pool | None = None

    async def _get_pool(self) -> asyncpg.Pool:
        if self._pool is None:
            pool = await asyncpg.create_pool(self._dsn)
            async with pool.acquire() as conn:
                await conn.execute(_CREATE_TABLE)
            self._pool = pool
        return self._pool

    async def get_or_create(self, session_id: str) -> Session:
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            row = await conn.fetchrow("SELECT id, user_id, messages FROM sessions WHERE id = $1", session_id)
            if row is None:
                await conn.execute("INSERT INTO sessions (id) VALUES ($1)", session_id)
                return Session(id=session_id)
            messages = ModelMessagesTypeAdapter.validate_json(row["messages"] or "[]")
            return Session(id=row["id"], user_id=row["user_id"], messages=messages)

    async def save(self, session: Session) -> None:
        messages_json = ModelMessagesTypeAdapter.dump_json(session.messages).decode("utf-8")
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO sessions (id, user_id, messages, updated_at)
                VALUES ($1, $2, $3::jsonb, now())
                ON CONFLICT (id) DO UPDATE SET user_id = $2, messages = $3::jsonb, updated_at = now()
                """,
                session.id,
                session.user_id,
                messages_json,
            )

    async def link_user(self, session_id: str, user_id: str) -> None:
        session = await self.get_or_create(session_id)
        session.user_id = user_id
        await self.save(session)

    async def close(self) -> None:
        if self._pool is not None:
            await self._pool.close()
