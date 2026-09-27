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
sessions` resolves there without any per-file changes.

`grant_roles` is what makes a fresh, non-`public` schema actually reachable by Supabase's
`service_role` (or any other role) via PostgREST: Supabase's own bootstrap grants `service_role`
access to `public` automatically, but a schema we create ourselves starts with none - and
`service_role` bypassing Row Level Security is a *separate* thing from having ordinary SQL GRANT
privileges on this schema at all (confirmed the hard way: a fresh schema exposed to PostgREST but
with no grants still 403s the service_role key). Left as a generic role-name list, not hardcoded to
"service_role", so this stays usable against a plain Postgres database with no such role - pass
nothing and this step is skipped entirely."""

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


async def run_migrations(pool: asyncpg.Pool, schema: str = "public", grant_roles: list[str] | None = None) -> None:
    async with pool.acquire() as conn:
        await conn.execute("SELECT pg_advisory_lock($1)", _LOCK_KEY)
        try:
            # Quoting throughout: `schema`/`grant_roles` are operator-supplied config (env vars),
            # never end-user input, and Postgres identifiers can't be bound as query parameters.
            if schema != "public":
                await conn.execute(f'CREATE SCHEMA IF NOT EXISTS "{schema}"')
            await conn.execute(_TRACKING_TABLE)
            applied = {row["id"] for row in await conn.fetch("SELECT id FROM schema_migrations")}
            for migration_id, sql in _migration_files():
                if migration_id in applied:
                    continue
                async with conn.transaction():
                    await conn.execute(sql)
                    await conn.execute("INSERT INTO schema_migrations (id) VALUES ($1)", migration_id)
            for role in grant_roles or []:
                await conn.execute(f'GRANT USAGE ON SCHEMA "{schema}" TO "{role}"')
                await conn.execute(f'GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA "{schema}" TO "{role}"')
                await conn.execute(f'ALTER DEFAULT PRIVILEGES IN SCHEMA "{schema}" GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "{role}"')
        finally:
            await conn.execute("SELECT pg_advisory_unlock($1)", _LOCK_KEY)
