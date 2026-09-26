"""In-memory SessionStore — local dev only, state is lost on restart."""

import uuid

from agent_framework.core.store.base import Session, SessionStore


class MemoryStore(SessionStore):
    def __init__(self) -> None:
        self._sessions: dict[str, Session] = {}

    async def get_or_create(self, session_id: str) -> Session:
        return self._sessions.setdefault(session_id, Session(id=session_id))

    async def save(self, session: Session) -> None:
        self._sessions[session.id] = session

    async def link_user(self, session_id: str, user_id: str) -> Session:
        old = await self.get_or_create(session_id)
        new = Session(id=str(uuid.uuid4()), user_id=user_id, messages=old.messages, data=old.data)
        del self._sessions[session_id]
        self._sessions[new.id] = new
        return new
