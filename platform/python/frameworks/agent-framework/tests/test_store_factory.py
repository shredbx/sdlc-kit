"""Pure precedence-logic tests for build_store_from_env - env vars only, no live database or
network needed."""

import pytest
from agent_framework.core.store.factory import build_store_from_env
from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.core.store.postgres_store import PostgresStore
from agent_framework.core.store.supabase_store import SupabaseSessionStore


def _clear(monkeypatch: pytest.MonkeyPatch) -> None:
    for var in ("SESSION_STORE", "SUPABASE_URL", "SUPABASE_SERVICE_ROLE_KEY", "DATABASE_URL", "AGENT_FRAMEWORK_CONSUMER_PREFIX"):
        monkeypatch.delenv(var, raising=False)


def test_defaults_to_memory_when_nothing_configured(monkeypatch: pytest.MonkeyPatch) -> None:
    _clear(monkeypatch)
    assert isinstance(build_store_from_env(), MemoryStore)


def test_infers_postgres_when_only_database_url_is_set(monkeypatch: pytest.MonkeyPatch) -> None:
    _clear(monkeypatch)
    monkeypatch.setenv("DATABASE_URL", "postgresql://x/y")
    assert isinstance(build_store_from_env(), PostgresStore)


def test_infers_supabase_over_postgres_when_both_are_configured(monkeypatch: pytest.MonkeyPatch) -> None:
    _clear(monkeypatch)
    monkeypatch.setenv("SUPABASE_URL", "https://x.supabase.co")
    monkeypatch.setenv("SUPABASE_SERVICE_ROLE_KEY", "key")
    monkeypatch.setenv("DATABASE_URL", "postgresql://x/y")
    monkeypatch.setenv("AGENT_FRAMEWORK_CONSUMER_PREFIX", "acme")
    assert isinstance(build_store_from_env(), SupabaseSessionStore)


def test_explicit_session_store_wins_over_inference(monkeypatch: pytest.MonkeyPatch) -> None:
    _clear(monkeypatch)
    monkeypatch.setenv("SESSION_STORE", "memory")
    monkeypatch.setenv("SUPABASE_URL", "https://x.supabase.co")
    monkeypatch.setenv("SUPABASE_SERVICE_ROLE_KEY", "key")
    assert isinstance(build_store_from_env(), MemoryStore)


def test_supabase_without_a_consumer_prefix_raises(monkeypatch: pytest.MonkeyPatch) -> None:
    _clear(monkeypatch)
    monkeypatch.setenv("SESSION_STORE", "supabase")
    monkeypatch.setenv("SUPABASE_URL", "https://x.supabase.co")
    monkeypatch.setenv("SUPABASE_SERVICE_ROLE_KEY", "key")
    with pytest.raises(ValueError, match="AGENT_FRAMEWORK_CONSUMER_PREFIX"):
        build_store_from_env()


def test_unknown_explicit_session_store_raises(monkeypatch: pytest.MonkeyPatch) -> None:
    _clear(monkeypatch)
    monkeypatch.setenv("SESSION_STORE", "redis")
    with pytest.raises(ValueError, match="Unknown SESSION_STORE"):
        build_store_from_env()
