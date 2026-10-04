"""Chooses which SessionStore a consumer's main.py should use, purely from env vars - so a
consumer switches storage backends by editing .env, not code. Precedence: an explicit
SESSION_STORE override wins; otherwise Supabase (SUPABASE_URL + a service/secret key) if
configured, else a direct Postgres connection (FRAMEWORK_SESSION_DATABASE_URL) if configured, else
in-memory.

Memory/Postgres/Supabase are interchangeable here because they all implement the same
SessionStore interface (store/base.py) - a future filesystem-backed store would slot into this
same precedence chain the same way, no interface changes needed."""

import os

from agent_framework.core.store.base import SessionStore
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.store.postgres_store import PostgresStore
from agent_framework.core.store.supabase_store import SupabaseSessionStore

# Named for what it is and who reads it: a generic DATABASE_URL in the same environment (a platform's own,
# another tool's) must never silently become this framework's session database.
SESSION_DATABASE_URL = "FRAMEWORK_SESSION_DATABASE_URL"
_RENAMED_FROM = "DATABASE_URL"


def build_store_from_env() -> SessionStore:
    supabase_url = os.environ.get("SUPABASE_URL")
    supabase_key = os.environ.get("SUPABASE_SERVICE_ROLE_KEY")
    database_url = os.environ.get(SESSION_DATABASE_URL)

    explicit = os.environ.get("SESSION_STORE", "").lower()
    kind = explicit or ("supabase" if supabase_url and supabase_key else "postgres" if database_url else "memory")

    # The old name used to pick the Postgres store by itself. Falling through to memory now would lose every
    # session on the next restart without a word, so that one case is an error that names the new variable.
    if kind == "memory" and not explicit and not database_url and os.environ.get(_RENAMED_FROM):
        raise ValueError(
            f"{_RENAMED_FROM} was renamed {SESSION_DATABASE_URL}. Set that to use the direct Postgres session store, "
            "or set SESSION_STORE=memory to run without one."
        )

    if kind == "supabase":
        if not (supabase_url and supabase_key):
            raise ValueError("SESSION_STORE=supabase requires SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY")
        return SupabaseSessionStore(supabase_url, supabase_key, schema=_agent_framework_schema())
    if kind == "postgres":
        if not database_url:
            hint = f" ({_RENAMED_FROM} was renamed {SESSION_DATABASE_URL})" if os.environ.get(_RENAMED_FROM) else ""
            raise ValueError(f"SESSION_STORE=postgres requires {SESSION_DATABASE_URL}{hint}")
        return PostgresStore(database_url)
    if kind == "memory":
        return MemoryStore()
    raise ValueError(f"Unknown SESSION_STORE: {kind!r} (expected supabase, postgres, or memory)")


def _agent_framework_schema() -> str:
    prefix = os.environ.get("AGENT_FRAMEWORK_CONSUMER_PREFIX")
    if not prefix:
        raise ValueError("AGENT_FRAMEWORK_CONSUMER_PREFIX is required when using the Supabase session store")
    return f"{prefix}_agent_framework"
