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


class AuthUnavailable(Exception):
    """The identity provider could not answer (it is unreachable, timed out, or is rate limiting or failing). Not a verdict on the token: the routes
    answer a retryable 503 - never the 401 that tells a client to sign in again, which a provider hiccup must not cause."""


class UserVerifier(Protocol):
    async def __call__(self, token: str) -> str | None:
        """Returns the verified user's stable id, or None if the token is missing, invalid, or
        expired. Never raises for a bad token. Raises AuthUnavailable when the identity provider
        cannot give an answer, which the routes turn into a retryable 503."""
        ...
