"""Resolves a request's session id into a `Session`. Header-based for now; swapping in real auth
(cookie, bearer token) later only changes this file — routes/chat.py doesn't change."""

import uuid
from collections.abc import Awaitable, Callable

from fastapi import Header

from agent_framework.core.store.base import Session, SessionStore


def session_dependency(store: SessionStore) -> Callable[[str | None], Awaitable[Session]]:
    async def _resolve(x_session_id: str | None = Header(default=None)) -> Session:
        return await store.get_or_create(x_session_id or str(uuid.uuid4()))

    return _resolve
