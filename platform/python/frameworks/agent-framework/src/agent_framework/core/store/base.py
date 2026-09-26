"""Session storage: an adapter interface, plus a trivial in-memory implementation for local dev.
`postgres_store` (real deploys) implements the same interface. Async throughout — MemoryStore
does no real I/O, but a real backing store (postgres, redis) does, and the interface shouldn't
differ between them."""

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


class SessionStore(ABC):
    @abstractmethod
    async def get_or_create(self, session_id: str) -> Session: ...

    @abstractmethod
    async def save(self, session: Session) -> None: ...

    @abstractmethod
    async def link_user(self, session_id: str, user_id: str) -> Session:
        """Associates a session with an identified user — e.g. once a handoff or login resolves
        who's actually on the other end of an until-then-anonymous session — and rotates the
        session id in the process. OWASP Session Management Cheat Sheet: regenerate the session
        identifier on any privilege-level change, so an id that was ever anonymous can't be reused
        to ride into an identified session. Returns the session under its NEW id; `session_id` is
        no longer valid afterwards — the caller must hand the client a freshly signed token for
        the id on the returned `Session`."""
        ...
