"""MemoryStore-level checks of how a login rotates the session id (OWASP: regenerate the session identifier on a
privilege-level change) - a new id, the old row gone once the new one is saved, a miss writing nothing - mirroring
test_postgres_store.py's equivalents, but running without docker."""

from agent_framework.core.store.base import Session, rotate_for_user
from agent_framework.core.store.memory_store import MemoryStore


async def test_a_miss_writes_nothing() -> None:
    store = MemoryStore()

    session = await store.get_or_create("never-saved")

    assert session.id == "never-saved" and session.messages == [] and session.data == {}
    assert store._sessions == {}  # noqa: SLF001 - white-box: nothing was stored


async def test_rotating_a_user_in_gives_a_new_id_carries_the_state_and_deletes_the_old_row_on_save() -> None:
    store = MemoryStore()
    session = await store.get_or_create("anon-1")
    session.data["contact_name"] = "Andrei"
    await store.save(session)

    linked = rotate_for_user(await store.get_or_create("anon-1"), "user-123")

    assert linked.id != "anon-1"
    assert linked.user_id == "user-123"
    assert linked.data == {"contact_name": "Andrei"}  # state carried over, not dropped
    assert linked.replaces == "anon-1"
    assert (await store.get_or_create("anon-1")).data == {"contact_name": "Andrei"}  # nothing is written until the new session is saved

    await store.save(linked)

    assert linked.replaces is None  # the removal happened once
    orphan = await store.get_or_create("anon-1")  # the pre-link id is gone, not still pointing at the identified session
    assert orphan.user_id is None and orphan.data == {}
    assert (await store.get_or_create(linked.id)).user_id == "user-123"


def test_a_session_that_was_never_filled_has_no_old_row_to_delete() -> None:
    assert rotate_for_user(Session(id="fresh"), "user-123").replaces is None
