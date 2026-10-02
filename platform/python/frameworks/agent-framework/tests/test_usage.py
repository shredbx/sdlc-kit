"""Token totals and cost are pure arithmetic - table-tested without a model or a store."""

import pytest
from agent_framework.server.usage import Pricing, TokenUsage, add_turn_usage, cost_usd, load_pricing_from_env, summarize


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
