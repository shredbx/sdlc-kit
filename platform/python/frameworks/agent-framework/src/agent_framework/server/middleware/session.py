"""Resolves a request's session id from a signed, server-minted token carried in the
`X-Session-Id` header. Missing, invalid, or expired -> a fresh session is minted server-side (the
server never adopts a client-proposed id, which is what actually prevents session fixation); the
signed token for whichever session was resolved is returned alongside it so the caller
(routes/chat.py) can hand it back to the client to carry on the next request. Transport is the
caller's choice - this only produces/verifies the token, it doesn't set cookies or decide how the
client stores it."""

import uuid
from collections.abc import Awaitable, Callable
from dataclasses import dataclass

from fastapi import Header
from itsdangerous import BadSignature, SignatureExpired, URLSafeTimedSerializer

from agent_framework.core.store.base import Session, SessionStore

_SALT = "agent-framework.session"
_MAX_AGE_SECONDS = 60 * 60 * 24 * 30  # 30-day idle expiry


@dataclass
class ResolvedSession:
    session: Session
    token: str  # signed token for `session.id` - hand this back to the client


def session_dependency(store: SessionStore, secret_key: str) -> Callable[[str | None], Awaitable[ResolvedSession]]:
    serializer = URLSafeTimedSerializer(secret_key, salt=_SALT)

    async def _resolve(x_session_id: str | None = Header(default=None)) -> ResolvedSession:
        session_id = _verify(serializer, x_session_id) or str(uuid.uuid4())
        session = await store.get_or_create(session_id)
        return ResolvedSession(session=session, token=serializer.dumps(session.id))

    return _resolve


def _verify(serializer: URLSafeTimedSerializer, token: str | None) -> str | None:
    if not token:
        return None
    try:
        session_id = serializer.loads(token, max_age=_MAX_AGE_SECONDS)
    except (BadSignature, SignatureExpired):
        return None
    return session_id if isinstance(session_id, str) else None
