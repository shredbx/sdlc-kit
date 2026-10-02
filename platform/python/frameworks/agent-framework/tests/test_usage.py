"""Token totals and cost are pure arithmetic - table-tested without a model or a store."""

import pytest
from agent_framework.server.usage import (
    Pricing,
    TokenLimit,
    TokenUsage,
    add_turn_usage,
    cost_usd,
    load_pricing_from_env,
    load_token_limit_from_env,
    summarize,
)


def test_a_fresh_session_starts_from_zero_and_the_input_is_not_mutated() -> None:
    stored: dict[str, int] = {}

    updated = add_turn_usage(stored, TokenUsage(requests=2, input_tokens=100, output_tokens=12, tool_calls=1))

    assert updated == {"requests": 2, "input_tokens": 100, "output_tokens": 12, "tool_calls": 1}
    assert stored == {}


def test_turns_accumulate() -> None:
    first = add_turn_usage({}, TokenUsage(requests=1, input_tokens=100, output_tokens=10))

    second = add_turn_usage(first, TokenUsage(requests=3, input_tokens=300, output_tokens=30, tool_calls=2))

    assert second == {"requests": 4, "input_tokens": 400, "output_tokens": 40, "tool_calls": 2}


def test_cost_needs_both_prices() -> None:
    usage = TokenUsage(input_tokens=2_000_000, output_tokens=500_000)

    assert cost_usd(usage, Pricing(input_per_mtok=0.10, output_per_mtok=0.40)) == 0.4  # 0.2 + 0.2
    assert cost_usd(usage, Pricing(input_per_mtok=0.10)) is None
    assert cost_usd(usage, Pricing(output_per_mtok=0.40)) is None
    assert cost_usd(usage, Pricing()) is None
    assert cost_usd(usage, None) is None


def test_cost_is_rounded_to_micro_dollars() -> None:
    assert cost_usd(TokenUsage(input_tokens=1_000_000, output_tokens=0), Pricing(input_per_mtok=0.1234567, output_per_mtok=0.0)) == 0.123457


def test_summary_reports_the_turn_the_session_and_the_session_cost() -> None:
    turn = TokenUsage(requests=1, input_tokens=1_000_000, output_tokens=0)
    stored = add_turn_usage({"requests": 1, "input_tokens": 1_000_000, "output_tokens": 0, "tool_calls": 0}, turn)

    summary = summarize(stored, turn, Pricing(input_per_mtok=0.5, output_per_mtok=1.0))

    assert summary.turn.input_tokens == 1_000_000
    assert summary.session.input_tokens == 2_000_000
    assert summary.cost_usd == 1.0


def test_summary_without_pricing_has_no_cost() -> None:
    turn = TokenUsage(requests=1, input_tokens=10, output_tokens=5)

    summary = summarize(add_turn_usage({}, turn), turn, None)

    assert summary.cost_usd is None


def test_prices_are_read_per_provider_prefix(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("GOOGLE_PRICE_INPUT_PER_MTOK", "0.10")
    monkeypatch.setenv("GOOGLE_PRICE_OUTPUT_PER_MTOK", "0.40")
    monkeypatch.setenv("OPENROUTER_PRICE_INPUT_PER_MTOK", "9")

    assert load_pricing_from_env("google") == Pricing(input_per_mtok=0.10, output_per_mtok=0.40)
    assert load_pricing_from_env("openrouter") == Pricing(input_per_mtok=9.0, output_per_mtok=None)


def test_unset_or_blank_prices_are_none(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("GOOGLE_PRICE_INPUT_PER_MTOK", raising=False)
    monkeypatch.setenv("GOOGLE_PRICE_OUTPUT_PER_MTOK", "  ")

    assert load_pricing_from_env("google") == Pricing()


def test_a_price_that_is_not_a_number_fails_loudly(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("GOOGLE_PRICE_INPUT_PER_MTOK", "cheap")

    with pytest.raises(ValueError, match="GOOGLE_PRICE_INPUT_PER_MTOK"):
        load_pricing_from_env("google")


def test_the_summary_carries_the_limit_it_was_given() -> None:
    turn = TokenUsage(requests=1, input_tokens=10, output_tokens=5)

    assert summarize(add_turn_usage({}, turn), turn, None, TokenLimit(tokens=100)).limit == TokenLimit(tokens=100, rule="WARN")
    assert summarize(add_turn_usage({}, turn), turn, None).limit is None


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
