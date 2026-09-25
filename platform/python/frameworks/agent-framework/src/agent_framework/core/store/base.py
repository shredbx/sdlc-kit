"""Session storage: an adapter interface, plus a trivial in-memory implementation for local dev.
`redis_store`/`postgres_store` (later milestones) implement the same interface for real deploys."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any


@dataclass
class Session:
    """One conversation's persisted state."""

    id: str
    data: dict[str, Any] = field(default_factory=dict)


class SessionStore(ABC):
    @abstractmethod
    def get_or_create(self, session_id: str) -> Session: ...

    @abstractmethod
    def save(self, session: Session) -> None: ...
