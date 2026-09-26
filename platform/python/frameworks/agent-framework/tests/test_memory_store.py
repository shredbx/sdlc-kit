"""MemoryStore-level check that link_user rotates the session id (OWASP: regenerate the session
identifier on a privilege-level change) rather than just tagging the same id with a user_id -
mirrors test_postgres_store.py's equivalent, but runs without docker."""

from agent_framework.core.store.memory_store import MemoryStore


async def test_link_user_rotates_the_id_and_carries_state_forward() -> None:
    store = MemoryStore()
    session = await store.get_or_create("anon-1")
    session.data["contact_name"] = "Andrei"
    await store.save(session)

    linked = await store.link_user("anon-1", "user-123")

    assert linked.id != "anon-1"
    assert linked.user_id == "user-123"
    assert linked.data == {"contact_name": "Andrei"}  # state carried over, not dropped

    # The pre-link id is gone, not still pointing at the (now identified) session.
    orphan = await store.get_or_create("anon-1")
    assert orphan.user_id is None
    assert orphan.data == {}
