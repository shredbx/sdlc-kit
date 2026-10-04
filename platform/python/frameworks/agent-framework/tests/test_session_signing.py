"""Proves the signing contract in server/middleware/session.py: a fresh token round-trips to the
same session, and a missing/tampered/wrong-secret token never raises or gets trusted — it always
just falls back to a brand-new session, since the server must never adopt a client-proposed id."""

from agent_framework.core.store.memory_store import MemoryStore
from agent_framework.server.middleware.session import session_dependency
from itsdangerous import URLSafeTimedSerializer


async def test_no_header_mints_a_fresh_session_with_a_valid_token() -> None:
    resolve = session_dependency(MemoryStore(), secret_key="s3cret")

    resolved = await resolve(None)

    assert resolved.session.id
    assert resolved.token  # a real signed token, not the bare id
    assert resolved.token != resolved.session.id


async def test_a_valid_token_resolves_back_to_the_same_session() -> None:
    store = MemoryStore()
    resolve = session_dependency(store, secret_key="s3cret")
    first = await resolve(None)
    await store.save(first.session)  # a session exists once a turn has saved it: resolving alone writes nothing

    second = await resolve(first.token)

    assert second.session.id == first.session.id
    assert len(store._sessions) == 1  # noqa: SLF001 — white-box check, same test module


async def test_garbage_token_falls_back_to_a_new_session_instead_of_raising() -> None:
    resolve = session_dependency(MemoryStore(), secret_key="s3cret")

    resolved = await resolve("not-a-real-token")

    assert resolved.session.id


async def test_token_signed_with_a_different_secret_is_rejected() -> None:
    other_secret_token = URLSafeTimedSerializer("a-different-secret", salt="agent-framework.session").dumps("victim-session-id")
    resolve = session_dependency(MemoryStore(), secret_key="s3cret")

    resolved = await resolve(other_secret_token)

    assert resolved.session.id != "victim-session-id"
