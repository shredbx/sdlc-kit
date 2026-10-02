"""In-memory SessionStore — local dev only, state is lost on restart."""

from agent_framework.core.store.base import Session, SessionStore


class MemoryStore(SessionStore):
    def __init__(self) -> None:
        self._sessions: dict[str, Session] = {}

    async def get_or_create(self, session_id: str) -> Session:
        return self._sessions.get(session_id) or Session(id=session_id)

    async def save(self, session: Session) -> None:
        self._sessions[session.id] = session
        if session.replaces:
            self._sessions.pop(session.replaces, None)
            session.replaces = None
