"""The key-usage reader, proven against a fake provider endpoint (httpx.MockTransport): no network, no real key. Each test
counts how many times the fake provider was asked and what it was sent."""

import asyncio
from datetime import UTC, datetime

import httpx
import pytest
from agent_framework.core.providers.credits import (
    DEFAULT_CACHE_SECONDS,
    OPENROUTER_KEY_URL,
    CreditsUnavailable,
    OpenRouterKeyUsage,
    load_credits_from_env,
    parse_key_usage,
)

SECRET = "sk-or-test-not-a-real-key"
NOW = datetime(2026, 10, 2, 11, 0, 0, tzinfo=UTC)


def key_body(**data: object) -> dict:
    """What GET /api/v1/key returns for a key with a credit limit, with any field overridden (None removes it)."""
    body = {
        **dict(label="a-project-key", limit=20.0, limit_remaining=18.77, limit_reset=None, usage=1.23),
        **dict(usage_daily=0.05, usage_weekly=0.4, usage_monthly=1.1, is_free_tier=False),
    }
    body.update(data)
    return {"data": {name: value for name, value in body.items() if value is not None}}


class Clock:
    def __init__(self) -> None:
        self.now = 1000.0

    def __call__(self) -> float:
        return self.now


class FakeProvider:
    """Answers GET /key from a queue of responses (the last one repeats), and remembers every request."""

    def __init__(self, *responses: httpx.Response | Exception) -> None:
        self.responses = list(responses) or [httpx.Response(200, json=key_body())]
        self.requests: list[httpx.Request] = []

    async def __call__(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        await asyncio.sleep(0)  # a real provider takes time: other callers get to run while this one waits
        answer = self.responses.pop(0) if len(self.responses) > 1 else self.responses[0]
        if isinstance(answer, Exception):
            raise answer
        return answer


def source(provider: FakeProvider, clock: Clock | None = None, cache_seconds: float = 30) -> OpenRouterKeyUsage:
    return OpenRouterKeyUsage(SECRET, cache_seconds=cache_seconds, transport=httpx.MockTransport(provider), clock=clock or Clock(), now=lambda: NOW)


class TestParsing:
    def test_a_key_with_a_credit_limit_reports_what_is_left_under_it(self) -> None:
        usage = parse_key_usage(key_body(), as_of="t")

        assert (usage.used, usage.used_today, usage.used_week, usage.used_month) == (1.23, 0.05, 0.4, 1.1)
        assert (usage.limit, usage.remaining, usage.currency, usage.provider, usage.stale) == (20.0, 18.77, "USD", "openrouter", False)

    def test_a_key_without_a_limit_has_no_remaining_to_show(self) -> None:
        usage = parse_key_usage(key_body(limit=None, limit_remaining=None), as_of="t")

        assert usage.limit is None and usage.remaining is None and usage.used == 1.23

    def test_periods_the_provider_does_not_send_are_left_out_not_zero(self) -> None:
        usage = parse_key_usage(key_body(usage_daily=None, usage_weekly=None, usage_monthly=None), as_of="t")

        assert (usage.used_today, usage.used_week, usage.used_month) == (None, None, None)

    @pytest.mark.parametrize("body", [{}, {"data": None}, {"data": {}}, {"data": {"usage": "1.2"}}, {"data": {"usage": True}}, [], "nope"])
    def test_an_answer_without_a_usage_figure_is_unusable(self, body: object) -> None:
        with pytest.raises(CreditsUnavailable):
            parse_key_usage(body, as_of="t")


@pytest.mark.asyncio
class TestReading:
    async def test_asks_the_provider_for_the_calling_key_only(self) -> None:
        provider = FakeProvider()

        usage = await source(provider).current()

        [request] = provider.requests
        assert (request.method, str(request.url)) == ("GET", OPENROUTER_KEY_URL)
        assert request.headers["authorization"] == f"Bearer {SECRET}"
        assert (usage.used, usage.remaining, usage.as_of) == (1.23, 18.77, "2026-10-02T11:00:00+00:00")

    async def test_the_reading_is_kept_for_the_cache_time_and_then_asked_again(self) -> None:
        provider, clock = FakeProvider(), Clock()
        reader = source(provider, clock, cache_seconds=30)

        await reader.current()
        clock.now += 29
        await reader.current()
        assert len(provider.requests) == 1

        clock.now += 2
        await reader.current()
        assert len(provider.requests) == 2

    async def test_zero_cache_seconds_asks_every_time(self) -> None:
        provider = FakeProvider()
        reader = source(provider, cache_seconds=0)

        await reader.current()
        await reader.current()

        assert len(provider.requests) == 2

    async def test_many_askers_at_once_cost_one_call(self) -> None:
        provider = FakeProvider()
        reader = source(provider)

        readings = await asyncio.gather(*(reader.current() for _ in range(8)))

        assert len(provider.requests) == 1
        assert {r.used for r in readings} == {1.23}


@pytest.mark.asyncio
class TestWhenTheProviderFails:
    @pytest.mark.parametrize(
        "failure",
        [
            httpx.Response(401, json={"error": "no"}),
            httpx.Response(429),
            httpx.Response(500, text="oops"),
            httpx.Response(200, text="<html>"),  # not JSON
            httpx.ConnectError("down"),
            httpx.ReadTimeout("slow"),
        ],
    )
    async def test_with_no_earlier_reading_the_failure_is_raised(self, failure: httpx.Response | Exception) -> None:
        with pytest.raises(CreditsUnavailable):
            await source(FakeProvider(failure)).current()

    async def test_with_an_earlier_reading_that_reading_is_returned_marked_stale(self) -> None:
        provider, clock = FakeProvider(httpx.Response(200, json=key_body()), httpx.Response(500)), Clock()
        reader = source(provider, clock)
        first = await reader.current()

        clock.now += 60
        second = await reader.current()

        assert (first.stale, second.stale) == (False, True)
        assert (second.used, second.remaining, second.as_of) == (first.used, first.remaining, first.as_of)

    async def test_a_failing_provider_is_not_hammered_and_recovery_clears_the_mark(self) -> None:
        provider, clock = FakeProvider(httpx.Response(200, json=key_body()), httpx.Response(500), httpx.Response(200, json=key_body(usage=2.0))), Clock()
        reader = source(provider, clock)
        await reader.current()

        clock.now += 60
        await reader.current()  # the failing call
        clock.now += 1
        await reader.current()  # inside the back-off: no new call
        assert len(provider.requests) == 2

        clock.now += 10
        recovered = await reader.current()
        assert len(provider.requests) == 3
        assert (recovered.stale, recovered.used) == (False, 2.0)

    async def test_the_key_is_in_no_error_message(self) -> None:
        for failure in (httpx.Response(401, text=f"bad key {SECRET}"), httpx.ConnectError(f"cannot reach with {SECRET}")):
            with pytest.raises(CreditsUnavailable) as raised:
                await source(FakeProvider(failure)).current()
            assert SECRET not in str(raised.value) and SECRET not in repr(raised.value)

    async def test_the_key_is_in_no_reading(self) -> None:
        reading = await source(FakeProvider()).current()

        assert SECRET not in reading.model_dump_json()


class TestFromTheEnvironment:
    def test_no_key_no_source(self) -> None:
        assert load_credits_from_env({}) is None
        assert load_credits_from_env({"OPENROUTER_API_KEY": "  "}) is None

    def test_a_key_gives_a_source_with_the_default_cache_time(self) -> None:
        reader = load_credits_from_env({"OPENROUTER_API_KEY": SECRET})

        assert isinstance(reader, OpenRouterKeyUsage) and reader._ttl == DEFAULT_CACHE_SECONDS  # noqa: SLF001

    def test_the_cache_time_can_be_set(self) -> None:
        reader = load_credits_from_env({"OPENROUTER_API_KEY": SECRET, "CREDITS_CACHE_SECONDS": " 120 "})

        assert reader is not None and reader._ttl == 120  # noqa: SLF001

    @pytest.mark.parametrize("value", ["soon", "1.5", "-1"])
    def test_a_bad_cache_time_stops_startup_and_names_the_variable(self, value: str) -> None:
        with pytest.raises(ValueError, match="CREDITS_CACHE_SECONDS"):
            load_credits_from_env({"OPENROUTER_API_KEY": SECRET, "CREDITS_CACHE_SECONDS": value})
