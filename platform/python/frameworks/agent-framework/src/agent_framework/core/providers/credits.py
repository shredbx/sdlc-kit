"""What the model key has spent, read from the provider, so a client can show it.

OpenRouter answers `GET /api/v1/key` for the key that is calling it - any normal API key, no management key - with
that key's `usage`, `usage_daily`, `usage_weekly` and `usage_monthly` in USD and, only when a credit limit is set on
the key, `limit` and `limit_remaining`. The project's key is its own, so this is what belongs to the project: nothing
about the rest of the OpenRouter account is read or shown.

Many clients asking often must not become many provider calls, so one reading is kept for a few seconds
(`CREDITS_CACHE_SECONDS`, default 30) and concurrent askers share a single call. If the provider fails, the last good
reading is returned marked `stale`; with none yet, the failure is raised. The key is sent only to the provider and
appears in no response, message or log line. Providers with no such endpoint (Google) simply have no source."""

import asyncio
import os
import time
from collections.abc import Callable, Mapping
from datetime import UTC, datetime
from typing import Any, Protocol

import httpx
from pydantic import BaseModel

OPENROUTER_KEY_URL = "https://openrouter.ai/api/v1/key"
DEFAULT_CACHE_SECONDS = 30
_REQUEST_TIMEOUT_SECONDS = 8.0
_FAILURE_BACKOFF_SECONDS = 5.0  # after a failed call, don't ask the provider again for this long (callers get the stale reading)


class KeyUsage(BaseModel):
    supported: bool = True
    provider: str
    currency: str = "USD"
    used: float  # everything this key has spent
    used_today: float | None = None  # current UTC day
    used_week: float | None = None  # current UTC week, Monday to Sunday
    used_month: float | None = None  # current UTC month
    limit: float | None = None  # the key's credit limit, when one is set on it
    remaining: float | None = None  # what is left under that limit; None when the key has no limit
    as_of: str  # when the provider was asked, ISO 8601 UTC
    stale: bool = False  # True: the provider could not be reached and this is the last reading


class CreditsUnavailable(Exception):
    """The provider could not be asked, or did not answer in a way that can be read. Never carries the key."""


class CreditsSource(Protocol):
    async def current(self) -> KeyUsage: ...


def parse_key_usage(body: Any, *, as_of: str) -> KeyUsage:
    """OpenRouter's `GET /key` response -> KeyUsage. Raises CreditsUnavailable when `data.usage` is not there."""
    data = body.get("data") if isinstance(body, dict) else None
    used = _number(data.get("usage")) if isinstance(data, dict) else None
    if not isinstance(data, dict) or used is None:
        raise CreditsUnavailable("OpenRouter's answer has no usage figure")
    return KeyUsage(
        provider="openrouter",
        used=used,
        used_today=_number(data.get("usage_daily")),
        used_week=_number(data.get("usage_weekly")),
        used_month=_number(data.get("usage_monthly")),
        limit=_number(data.get("limit")),
        remaining=_number(data.get("limit_remaining")),
        as_of=as_of,
    )


def _number(value: Any) -> float | None:
    return float(value) if isinstance(value, int | float) and not isinstance(value, bool) else None


class OpenRouterKeyUsage:
    def __init__(
        self,
        api_key: str,
        *,
        cache_seconds: float = DEFAULT_CACHE_SECONDS,
        url: str = OPENROUTER_KEY_URL,
        transport: httpx.AsyncBaseTransport | None = None,
        clock: Callable[[], float] = time.monotonic,
        now: Callable[[], datetime] = lambda: datetime.now(UTC),
    ) -> None:
        self._api_key, self._ttl, self._url, self._transport = api_key, cache_seconds, url, transport
        self._clock, self._now = clock, now
        self._lock = asyncio.Lock()
        self._reading: KeyUsage | None = None
        self._read_at = 0.0
        self._failed_at: float | None = None

    async def current(self) -> KeyUsage:
        if self._fresh():
            return self._reading  # type: ignore[return-value]
        async with self._lock:
            if self._fresh():  # someone else read it while this caller waited for the lock
                return self._reading  # type: ignore[return-value]
            if self._failed_at is not None and self._clock() - self._failed_at < _FAILURE_BACKOFF_SECONDS:
                return self._last_good_or_raise(CreditsUnavailable("the provider failed a moment ago"))
            try:
                reading = await self._fetch()
            except CreditsUnavailable as failure:
                self._failed_at = self._clock()
                return self._last_good_or_raise(failure)
            self._reading, self._read_at, self._failed_at = reading, self._clock(), None
            return reading

    def _fresh(self) -> bool:
        return self._reading is not None and self._clock() - self._read_at < self._ttl

    def _last_good_or_raise(self, failure: CreditsUnavailable) -> KeyUsage:
        if self._reading is None:
            raise failure
        return self._reading.model_copy(update={"stale": True})

    async def _fetch(self) -> KeyUsage:
        try:
            async with httpx.AsyncClient(transport=self._transport, timeout=_REQUEST_TIMEOUT_SECONDS) as client:
                response = await client.get(self._url, headers={"Authorization": f"Bearer {self._api_key}"})
        except httpx.HTTPError as error:
            raise CreditsUnavailable(f"could not reach OpenRouter ({type(error).__name__})") from None
        if response.status_code != 200:
            raise CreditsUnavailable(f"OpenRouter answered {response.status_code}")
        try:
            body = response.json()
        except ValueError:
            raise CreditsUnavailable("OpenRouter's answer is not JSON") from None
        return parse_key_usage(body, as_of=self._now().isoformat(timespec="seconds"))


def load_credits_from_env(env: Mapping[str, str] | None = None) -> OpenRouterKeyUsage | None:
    """A source when OPENROUTER_API_KEY is set, else None (a Google-only deployment has nothing to show).
    CREDITS_CACHE_SECONDS (default 30, 0 = ask every time) - a value that is not a whole number raises at startup."""
    env = os.environ if env is None else env
    key = env.get("OPENROUTER_API_KEY", "").strip()
    if not key:
        return None
    raw = env.get("CREDITS_CACHE_SECONDS", "").strip()
    try:
        seconds = int(raw) if raw else DEFAULT_CACHE_SECONDS
    except ValueError as error:
        raise ValueError(f"CREDITS_CACHE_SECONDS must be a whole number of seconds, got {raw!r}") from error
    if seconds < 0:
        raise ValueError(f"CREDITS_CACHE_SECONDS must be 0 or more, got {seconds}")
    return OpenRouterKeyUsage(key, cache_seconds=seconds)
