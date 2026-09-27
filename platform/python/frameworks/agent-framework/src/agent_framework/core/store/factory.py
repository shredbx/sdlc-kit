"""Chooses which SessionStore a consumer's main.py should use, purely from env vars - so a
consumer switches storage backends by editing .env, not code. Precedence: an explicit
SESSION_STORE override wins; otherwise Supabase (SUPABASE_URL + a service/secret key) if
configured, else a direct Postgres connection (DATABASE_URL) if configured, else in-memory.

Memory/Postgres/Supabase are interchangeable here because they all implement the same
SessionStore interface (store/base.py) - a future filesystem-backed store would slot into this
same precedence chain the same way, no interface changes needed."""

import os

from agent_framework.core.store.base import SessionStore
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.store.postgres_store import PostgresStore
from agent_framework.core.store.supabase_store import SupabaseSessionStore


def build_store_from_env() -> SessionStore:
    supabase_url = os.environ.get("SUPABASE_URL")
    supabase_key = os.environ.get("SUPABASE_SERVICE_ROLE_KEY")
    database_url = os.environ.get("DATABASE_URL")

    kind = os.environ.get("SESSION_STORE", "").lower() or ("supabase" if supabase_url and supabase_key else "postgres" if database_url else "memory")

    if kind == "supabase":
        if not (supabase_url and supabase_key):
            raise ValueError("SESSION_STORE=supabase requires SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY")
        return SupabaseSessionStore(supabase_url, supabase_key, schema=_agent_framework_schema())
    if kind == "postgres":
        if not database_url:
            raise ValueError("SESSION_STORE=postgres requires DATABASE_URL")
        return PostgresStore(database_url)
    if kind == "memory":
        return MemoryStore()
    raise ValueError(f"Unknown SESSION_STORE: {kind!r} (expected supabase, postgres, or memory)")


def _agent_framework_schema() -> str:
    prefix = os.environ.get("AGENT_FRAMEWORK_CONSUMER_PREFIX")
    if not prefix:
        raise ValueError("AGENT_FRAMEWORK_CONSUMER_PREFIX is required when using the Supabase session store")
    return f"{prefix}_agent_framework"
