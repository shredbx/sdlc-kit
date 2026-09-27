"""Applies agent_framework's own schema migrations: a numbered `.sql` file per change, tracked in
a `schema_migrations` table so each one runs exactly once, in order, whether the target database
is brand new (first run — every migration applies) or already has some of them (only what's
missing applies). Files are plain SQL, not Python — readable and applyable outside this codebase
too (a DBA reviewing them, or a deploy pipeline running them directly).

Wrapped in a Postgres advisory lock: multiple app instances can safely boot against the same empty
database at once without racing each other's `CREATE TABLE`.

`schema` lets a consumer sharing one Postgres database across multiple products (e.g. several
client apps on one Supabase project) keep its own sessions/schema_migrations tables isolated —
e.g. `bestie_agent_framework` rather than the connection's default `public`. The migration files
themselves stay schema-agnostic: this only ever runs them with the connection's `search_path`
already pointed at the target schema (see PostgresStore), so their unqualified `CREATE TABLE
sessions` resolves there without any per-file changes."""

from pathlib import Path

import asyncpg

_MIGRATIONS_DIR = Path(__file__).parent / "migrations"
_LOCK_KEY = 727433  # arbitrary, fixed — this store's own advisory-lock namespace

_TRACKING_TABLE = """
CREATE TABLE IF NOT EXISTS schema_migrations (
    id TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)
"""


def _migration_files() -> list[tuple[str, str]]:
    """(id, sql) pairs — the numeric filename prefix defines apply order."""
    return [(path.stem, path.read_text()) for path in sorted(_MIGRATIONS_DIR.glob("*.sql"))]


async def run_migrations(pool: asyncpg.Pool, schema: str = "public") -> None:
    async with pool.acquire() as conn:
        await conn.execute("SELECT pg_advisory_lock($1)", _LOCK_KEY)
        try:
            if schema != "public":
                # Quoting: `schema` is operator-supplied config (an env var), never end-user input,
                # and Postgres identifiers can't be bound as query parameters anyway.
                await conn.execute(f'CREATE SCHEMA IF NOT EXISTS "{schema}"')
            await conn.execute(_TRACKING_TABLE)
            applied = {row["id"] for row in await conn.fetch("SELECT id FROM schema_migrations")}
            for migration_id, sql in _migration_files():
                if migration_id in applied:
                    continue
                async with conn.transaction():
                    await conn.execute(sql)
                    await conn.execute("INSERT INTO schema_migrations (id) VALUES ($1)", migration_id)
        finally:
            await conn.execute("SELECT pg_advisory_unlock($1)", _LOCK_KEY)
