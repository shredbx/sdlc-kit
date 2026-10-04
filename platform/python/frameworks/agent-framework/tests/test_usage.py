"""Token totals and cost are pure arithmetic - table-tested without a model or a store."""

from decimal import Decimal

import pytest
from agent_framework.server.usage import (
    TokenLimit,
    TokenUsage,
    TurnCost,
    add_turn_usage,
    load_token_limit_from_env,
    summarize,
    turn_cost,
)

NO_COST = TurnCost()


def test_a_fresh_session_starts_from_zero_and_the_input_is_not_mutated() -> None:
    stored: dict[str, int | float] = {}

    updated = add_turn_usage(stored, TokenUsage(requests=2, input_tokens=100, output_tokens=12, tool_calls=1), TurnCost(usd=0.002))

    assert updated == {"requests": 2, "input_tokens": 100, "output_tokens": 12, "tool_calls": 1, "cost_usd": 0.002, "unpriced_requests": 0}
    assert stored == {}


def test_turns_accumulate() -> None:
    first = add_turn_usage({}, TokenUsage(requests=1, input_tokens=100, output_tokens=10), TurnCost(usd=0.001))

    second = add_turn_usage(first, TokenUsage(requests=3, input_tokens=300, output_tokens=30, tool_calls=2), TurnCost(usd=0.0025))

    assert second == {"requests": 4, "input_tokens": 400, "output_tokens": 40, "tool_calls": 2, "cost_usd": 0.0035, "unpriced_requests": 0}


def test_the_cost_of_a_turn_is_what_each_request_cost_added_up() -> None:
    assert turn_cost([Decimal("0.001"), Decimal("0.0025")]) == TurnCost(usd=0.0035, unpriced_requests=0)


def test_a_request_with_no_known_price_is_counted_not_guessed() -> None:
    assert turn_cost([Decimal("0.001"), None, None]) == TurnCost(usd=0.001, unpriced_requests=2)
    assert turn_cost([]) == TurnCost(usd=0.0, unpriced_requests=0)


def test_a_turn_with_a_request_of_unknown_cost_reports_no_cost_and_a_known_one_is_rounded() -> None:
    assert TurnCost(usd=0.001, unpriced_requests=1).reported_usd is None
    assert TurnCost(usd=0.0012345678).reported_usd == 0.001235


def test_a_session_that_has_a_request_with_no_price_reports_no_cost() -> None:
    turn = TokenUsage(requests=2, input_tokens=10, output_tokens=5)
    stored = add_turn_usage({}, turn, TurnCost(usd=0.001, unpriced_requests=1))

    assert summarize(stored, turn).cost_usd is None


def test_a_session_started_before_costs_were_recorded_never_reports_a_cost() -> None:
    old = {"requests": 3, "input_tokens": 300, "output_tokens": 30, "tool_calls": 1}  # no cost keys: saved by an earlier version
    turn = TokenUsage(requests=1, input_tokens=100, output_tokens=10)

    stored = add_turn_usage(old, turn, TurnCost(usd=0.001))

    assert stored["unpriced_requests"] == 3  # the three earlier requests had no price recorded
    assert summarize(stored, turn).cost_usd is None
    assert summarize(old, turn).cost_usd is None


def test_the_cost_is_rounded_to_micro_dollars() -> None:
    turn = TokenUsage(requests=1, input_tokens=1, output_tokens=1)

    assert summarize(add_turn_usage({}, turn, TurnCost(usd=0.12345678)), turn).cost_usd == 0.123457


def test_summary_reports_the_turn_the_session_and_the_session_cost() -> None:
    first = TokenUsage(requests=1, input_tokens=1_000_000, output_tokens=0)
    second = TokenUsage(requests=1, input_tokens=1_000_000, output_tokens=0)
    stored = add_turn_usage(add_turn_usage({}, first, TurnCost(usd=0.5)), second, TurnCost(usd=0.5))

    summary = summarize(stored, second)

    assert summary.turn.input_tokens == 1_000_000
    assert summary.session.input_tokens == 2_000_000
    assert summary.cost_usd == 1.0


def test_a_session_of_unpriced_requests_has_no_cost() -> None:
    turn = TokenUsage(requests=1, input_tokens=10, output_tokens=5)

    summary = summarize(add_turn_usage({}, turn, TurnCost(unpriced_requests=1)), turn)

    assert summary.cost_usd is None


def test_the_summary_carries_the_limit_it_was_given() -> None:
    turn = TokenUsage(requests=1, input_tokens=10, output_tokens=5)

    assert summarize(add_turn_usage({}, turn, NO_COST), turn, TokenLimit(tokens=100)).limit == TokenLimit(tokens=100, rule="WARN")
    assert summarize(add_turn_usage({}, turn, NO_COST), turn).limit is None


def test_the_token_limit_is_read_from_the_environment(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("SESSION_TOKEN_LIMIT", "150000")
    monkeypatch.setenv("SESSION_TOKEN_LIMIT_RULE", " warn ")

    assert load_token_limit_from_env() == TokenLimit(tokens=150_000, rule="WARN")


def test_an_unset_or_blank_limit_means_no_limit_and_the_rule_defaults_to_warn(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("SESSION_TOKEN_LIMIT", raising=False)
    monkeypatch.delenv("SESSION_TOKEN_LIMIT_RULE", raising=False)
    assert load_token_limit_from_env() is None

    monkeypatch.setenv("SESSION_TOKEN_LIMIT", "  ")
    assert load_token_limit_from_env() is None

    monkeypatch.setenv("SESSION_TOKEN_LIMIT", "500")
    assert load_token_limit_from_env() == TokenLimit(tokens=500, rule="WARN")


@pytest.mark.parametrize("raw", ["lots", "1.5", "0", "-10"])
def test_a_limit_that_is_not_a_positive_whole_number_fails_loudly(monkeypatch: pytest.MonkeyPatch, raw: str) -> None:
    monkeypatch.setenv("SESSION_TOKEN_LIMIT", raw)

    with pytest.raises(ValueError, match="SESSION_TOKEN_LIMIT"):
        load_token_limit_from_env()


def test_a_rule_that_is_not_supported_fails_loudly_even_without_a_limit(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("SESSION_TOKEN_LIMIT", raising=False)
    monkeypatch.setenv("SESSION_TOKEN_LIMIT_RULE", "BLOCK")

    with pytest.raises(ValueError, match="SESSION_TOKEN_LIMIT_RULE"):
        load_token_limit_from_env()
