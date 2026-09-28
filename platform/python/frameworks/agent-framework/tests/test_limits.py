"""evaluate_and_consume is pure (takes `now` as a parameter) - every case below is deterministic,
no real sleeps, no store, no route."""

from datetime import UTC, datetime, timedelta

from agent_framework.core.limits import Limits, evaluate_and_consume

_NOW = datetime(2026, 9, 28, 12, 0, 0, tzinfo=UTC)


def test_first_message_ever_is_allowed_and_consumes_one() -> None:
    allowed, status, data = evaluate_and_consume({}, Limits(quota_per_hour=40), _NOW)

    assert allowed is True
    assert status.remaining == 39
    assert data == {"window_start": _NOW.isoformat(), "count": 1, "last_accepted_at": _NOW.isoformat()}


def test_second_message_within_throttle_window_is_rejected_and_changes_nothing() -> None:
    limits = Limits(throttle_seconds=3.0, quota_per_hour=40)
    _, _, after_first = evaluate_and_consume({}, limits, _NOW)

    allowed, status, data = evaluate_and_consume(after_first, limits, _NOW + timedelta(seconds=1))

    assert allowed is False
    assert status.remaining == 39  # unchanged - the first message's count, not decremented again
    assert data == after_first  # nothing persisted for a rejected request


def test_message_after_throttle_window_elapses_is_allowed() -> None:
    limits = Limits(throttle_seconds=3.0, quota_per_hour=40)
    _, _, after_first = evaluate_and_consume({}, limits, _NOW)

    allowed, status, _ = evaluate_and_consume(after_first, limits, _NOW + timedelta(seconds=3))

    assert allowed is True
    assert status.remaining == 38


def test_quota_cap_rejects_once_reached_even_outside_the_throttle_window() -> None:
    limits = Limits(throttle_seconds=0, quota_per_hour=2)
    data: dict = {}
    for i in range(2):
        allowed, status, data = evaluate_and_consume(data, limits, _NOW + timedelta(minutes=i))
        assert allowed is True

    allowed, status, data = evaluate_and_consume(data, limits, _NOW + timedelta(minutes=5))

    assert allowed is False
    assert status.remaining == 0


def test_quota_resets_one_hour_after_the_window_started() -> None:
    limits = Limits(throttle_seconds=0, quota_per_hour=1)
    _, _, after_first = evaluate_and_consume({}, limits, _NOW)
    rejected, rejected_status, _ = evaluate_and_consume(after_first, limits, _NOW + timedelta(minutes=59))
    assert rejected is False
    assert rejected_status.remaining == 0

    allowed, status, data = evaluate_and_consume(after_first, limits, _NOW + timedelta(hours=1, seconds=1))

    assert allowed is True
    assert status.remaining == 0  # quota_per_hour=1, just consumed
    assert data["window_start"] == (_NOW + timedelta(hours=1, seconds=1)).isoformat()


def test_resets_at_reflects_the_current_window_end() -> None:
    limits = Limits(quota_per_hour=40)

    _, status, _ = evaluate_and_consume({}, limits, _NOW)

    assert status.resets_at == (_NOW + timedelta(hours=1)).isoformat()
