"""Checking an access token on this server, without asking the identity provider: for a token signed with an asymmetric key (ES256 / RS256) the provider
publishes the public keys (a JWKS), and the signature, expiry and audience can be checked here in a fraction of a millisecond. This is what Supabase's own
client does for a project with asymmetric signing keys (supabase-js getClaims), and what its docs recommend over verifying with a shared secret.

What it deliberately does:
- Accepts only the algorithms of the key types it knows (RFC 8725 3.1, 3.2). A token that says `none`, HS256 or anything else is not accepted here - the
  caller decides what to do with HS256 (ask the provider).
- Takes the keys only from the provider's own key-set URL, so a key always belongs to the issuer (RFC 8725 3.8); checks the audience (3.9) and requires
  `exp` and `sub`.
- Keeps the key set for `refresh_seconds` (Supabase's edge caches it 10 minutes and asks apps not to keep it longer - a longer cache makes key revocation slow).
  A token with a `kid` the cache does not know triggers ONE refetch (a key was just rotated in), at most once per `unknown_kid_wait_seconds`, so garbage
  tokens cannot make this server hammer the provider. If a refetch fails the old key set keeps working; with no key set at all the answer is AuthUnavailable.
- Fetches once at a time: concurrent first requests share one fetch.

The price of checking locally: a user signed out or removed at the provider keeps access until their access token expires (the provider's jwt_expiry,
one hour by default)."""

import asyncio
import time
from collections.abc import Callable

import httpx
import jwt
from jwt import PyJWK

from agent_framework.core.auth import AuthUnavailable

LOCAL_ALGORITHMS = ("ES256", "RS256")
_KEY_TYPE_OF = {"ES256": "EC", "RS256": "RSA"}  # a token must name a key of the family its algorithm belongs to


class KeySetVerifier:
    def __init__(
        self,
        keys_url: str,
        *,
        audience: str,
        client: httpx.AsyncClient,
        headers: dict[str, str] | None = None,
        refresh_seconds: float = 600,
        unknown_kid_wait_seconds: float = 30,
        leeway_seconds: int = 10,
        clock: Callable[[], float] = time.monotonic,
    ) -> None:
        self._url, self._audience, self._client, self._headers = keys_url, audience, client, headers or {}
        self._refresh, self._wait, self._leeway, self._clock = refresh_seconds, unknown_kid_wait_seconds, leeway_seconds, clock
        self._keys: dict[str, PyJWK] | None = None
        self._fetched_at: float | None = None  # when the key set was last fetched, or last tried (a failed try counts: the next one waits)
        self._lock = asyncio.Lock()

    async def user_id(self, token: str) -> str | None:
        """The token's user id when it is a valid token signed by one of the provider's keys; None when it is not. Raises AuthUnavailable
        when there is no key set to check against and none can be fetched."""
        try:
            header = jwt.get_unverified_header(token)
        except jwt.PyJWTError:
            return None
        algorithm, kid = header.get("alg"), header.get("kid")
        if algorithm not in LOCAL_ALGORITHMS or not isinstance(kid, str):
            return None
        key = await self._key(kid)
        if key is None or key.key_type != _KEY_TYPE_OF[algorithm]:
            return None
        try:
            claims = jwt.decode(token, key.key, algorithms=[algorithm], audience=self._audience, leeway=self._leeway, options={"require": ["exp", "sub"]})
        except (jwt.PyJWTError, ValueError, TypeError):  # whatever the token was made of, a bad one is a "no", never a server error
            return None
        subject = claims.get("sub")
        return subject if isinstance(subject, str) and subject else None

    def _age(self) -> float:
        return float("inf") if self._fetched_at is None else self._clock() - self._fetched_at

    async def _key(self, kid: str) -> PyJWK | None:
        # With keys in hand they are refreshed every `refresh_seconds`; with none at all a new try waits only `unknown_kid_wait_seconds`.
        if self._age() >= (self._refresh if self._keys is not None else self._wait):
            await self._fetch(self._fetched_at)
        if self._keys is None:
            raise AuthUnavailable("there is no key set to check the token against")
        key = self._keys.get(kid)
        if key is None and self._age() >= self._wait:
            await self._fetch(self._fetched_at)  # a key may have just been rotated in
            key = (self._keys or {}).get(kid)
        return key

    async def _fetch(self, seen_at: float | None) -> None:
        async with self._lock:
            if self._fetched_at != seen_at:
                return  # another request fetched while this one waited
            self._fetched_at = self._clock()  # also when it fails: the next try waits, it does not hammer
            try:
                response = await self._client.get(self._url, headers=self._headers)
                response.raise_for_status()
                entries = [entry for entry in response.json().get("keys", []) if isinstance(entry.get("kid"), str) and entry.get("kty") in ("EC", "RSA")]
                keys = {entry["kid"]: PyJWK(entry) for entry in entries}
            except (httpx.HTTPError, ValueError, KeyError, AttributeError, jwt.PyJWTError) as exc:
                if self._keys is None:
                    raise AuthUnavailable("the key set could not be fetched") from exc
                return  # keep working with the keys already held (for a full refresh period: a failed try counts as a fetch)
            self._keys = keys
