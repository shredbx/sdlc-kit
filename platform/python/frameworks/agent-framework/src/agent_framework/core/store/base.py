"""Session storage: an adapter interface, plus a trivial in-memory implementation for local dev.
`postgres_store` (real deploys) implements the same interface. Async throughout — MemoryStore
does no real I/O, but a real backing store (postgres, redis) does, and the interface shouldn't
differ between them."""

import uuid
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any

from pydantic_ai import ModelMessage


@dataclass
class Session:
    """One conversation's persisted state. `messages` is the full PydanticAI message history —
    passed back in as `message_history=` on the next run so the agent actually remembers the
    conversation, not just a per-request-stateless echo."""

    id: str
    user_id: str | None = None
    messages: list[ModelMessage] = field(default_factory=list)
    data: dict[str, Any] = field(default_factory=dict)
    # The id of an earlier row this session replaces (see rotate_for_user): the store deletes that row in the
    # same save that writes this one, so rotating an id costs no extra request of its own.
    replaces: str | None = None


class SessionStore(ABC):
    @abstractmethod
    async def get_or_create(self, session_id: str) -> Session:
        """The stored session, or - when there is none - a fresh empty one under `session_id`. A miss
        writes nothing: the first save creates the row, so a request that never saves leaves no row behind."""
        ...

    @abstractmethod
    async def save(self, session: Session) -> None:
        """Writes the session (an upsert), and when `session.replaces` names an earlier row, deletes it
        afterwards - never before, so a failed save loses nothing - and clears `replaces`."""
        ...


def rotate_for_user(session: Session, user_id: str) -> Session:
    """The session under a NEW id, linked to `user_id`, carrying the old one's messages and data. Pure: nothing
    is written until the new session is saved, which also deletes the old row (`replaces`) when it had content.

    OWASP Session Management Cheat Sheet: regenerate the session identifier on any privilege-level change
    (an anonymous session being linked to a user, or a session meeting a different user), so an id that was
    ever anonymous can't be reused to ride into an identified session. The caller must hand the client a freshly
    signed token for the new id - but only once the turn has been saved: until then the old id is still the valid one."""
    had_content = bool(session.messages or session.data)
    return Session(id=str(uuid.uuid4()), user_id=user_id, messages=session.messages, data=session.data, replaces=session.id if had_content else None)
