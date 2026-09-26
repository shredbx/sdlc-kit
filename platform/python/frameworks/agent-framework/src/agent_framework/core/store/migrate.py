"""Applies agent_framework's own schema migrations: a numbered `.sql` file per change, tracked in
a `schema_migrations` table so each one runs exactly once, in order, whether the target database
is brand new (first run — every migration applies) or already has some of them (only what's
missing applies). Files are plain SQL, not Python — readable and applyable outside this codebase
too (a DBA reviewing them, or a deploy pipeline running them directly).

Wrapped in a Postgres advisory lock: multiple app instances can safely boot against the same empty
database at once without racing each other's `CREATE TABLE`."""

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


async def run_migrations(pool: asyncpg.Pool) -> None:
    async with pool.acquire() as conn:
        await conn.execute("SELECT pg_advisory_lock($1)", _LOCK_KEY)
        try:
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
