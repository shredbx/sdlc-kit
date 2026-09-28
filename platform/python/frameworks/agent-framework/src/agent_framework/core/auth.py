"""Verifying a bearer token belongs to a real user, and what an agent requires of its caller.
`RegisteredAgent.auth` (core/types.py) is "public" (no verification - today's unchanged default,
e.g. the website widget) or "required" (routes/chat.py demands a valid `Authorization: Bearer
<token>` header, verified via a UserVerifier supplied to create_app/build_router, before the agent
runs at all).

A UserVerifier is deliberately just a callable, not a class hierarchy - any async function from a
token to a user id (or None) satisfies it, so a consumer plugs in whatever identity provider it
actually uses (agent_framework.api.supabase.build_supabase_user_verifier for Supabase Auth) without
this module needing to know about it."""

from typing import Protocol


class UserVerifier(Protocol):
    async def __call__(self, token: str) -> str | None:
        """Returns the verified user's stable id, or None if the token is missing, invalid, or
        expired. Never raises for a bad token - only for a genuine infrastructure failure (the
        identity provider itself unreachable), which routes/chat.py lets propagate as a 500 rather
        than silently treating as "not authenticated"."""
        ...
