"""Per-session throttle + hourly quota (M6): checked in routes/chat.py *before* the model runs, so
a rejected message costs nothing - no model call, no extra store write. Counters live in
`Session.data["limits"]` - the same free-form per-session bag core/store/base.py already persists,
no new storage mechanism, no new table.

`evaluate_and_consume` is a pure function (takes `now` as a parameter, touches no clock, no I/O) -
the throttle/quota/reset-window logic is fully testable without real sleeps and without a store."""

import os
from datetime import datetime, timedelta
from typing import Any

from pydantic import BaseModel

_WINDOW = timedelta(hours=1)


class Limits(BaseModel):
    throttle_seconds: float = 3.0
    quota_per_hour: int = 40  # 0 = no hourly quota (the throttle still applies; no figure is reported to the client)


class LimitStatus(BaseModel):
    remaining: int
    resets_at: str  # ISO 8601, always UTC — the caller passes a timezone-aware `now`


def load_limits_from_env() -> Limits:
    """THROTTLE_SECONDS / QUOTA_PER_HOUR, both optional - same env-first pattern as
    core.store.factory.build_store_from_env. A consumer that wants limiting off entirely just
    doesn't pass a Limits to create_app(...) at all; this always returns one (falling back to
    Limits()'s own defaults), since calling this implies the consumer wants the feature on."""
    defaults = Limits()
    throttle_env = os.environ.get("THROTTLE_SECONDS")
    quota_env = os.environ.get("QUOTA_PER_HOUR")
    return Limits(
        throttle_seconds=float(throttle_env) if throttle_env else defaults.throttle_seconds,
        quota_per_hour=int(quota_env) if quota_env else defaults.quota_per_hour,
    )


def evaluate_and_consume(data: dict[str, Any], limits: Limits, now: datetime) -> tuple[bool, LimitStatus, dict[str, Any]]:
    """Checks `data` (the session's persisted `data.get("limits", {})` bag - {} for a session that
    has never sent a message) against `limits`, at time `now`. Returns
    (allowed, status-to-report-to-the-client, the bag to persist back onto the session).

    The caller only writes the returned bag back onto `session.data["limits"]` when `allowed` is
    True - a rejected request must change nothing, per the framework rule this implements ("an
    over-limit message costs nothing")."""
    window_start = _parse(data.get("window_start"))
    count = data.get("count", 0) if window_start is not None else 0
    if window_start is None or now - window_start >= _WINDOW:
        window_start, count = now, 0

    last_accepted_at = _parse(data.get("last_accepted_at"))
    throttled = last_accepted_at is not None and now - last_accepted_at < timedelta(seconds=limits.throttle_seconds)
    over_quota = limits.quota_per_hour > 0 and count >= limits.quota_per_hour
    resets_at = (window_start + _WINDOW).isoformat()

    if throttled or over_quota:
        return False, LimitStatus(remaining=max(0, limits.quota_per_hour - count), resets_at=resets_at), data

    new_count = count + 1
    new_data = {"window_start": window_start.isoformat(), "count": new_count, "last_accepted_at": now.isoformat()}
    status = LimitStatus(remaining=max(0, limits.quota_per_hour - new_count), resets_at=resets_at)
    return True, status, new_data


def _parse(value: Any) -> datetime | None:
    return datetime.fromisoformat(value) if isinstance(value, str) else None
