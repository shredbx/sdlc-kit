"""In-memory SessionStore — local dev only, state is lost on restart."""

from agent_framework.core.store.base import Session, SessionStore


class MemoryStore(SessionStore):
    def __init__(self) -> None:
        self._sessions: dict[str, Session] = {}

    def get_or_create(self, session_id: str) -> Session:
        return self._sessions.setdefault(session_id, Session(id=session_id))

    def save(self, session: Session) -> None:
        self._sessions[session.id] = session
