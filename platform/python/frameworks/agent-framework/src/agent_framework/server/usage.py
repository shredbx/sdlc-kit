"""Token usage per turn and per session, and what it costs. The running per-session totals live in
`Session.data["usage"]` - the same free-form per-session bag core/limits.py already uses, no new
storage - and are summed here from pydantic-ai's own RunUsage after each completed run.

Prices are optional configuration the consumer passes in (`Pricing`, USD per million tokens). When
either price is missing the cost is None - never a guess - and a client shows tokens only. All
functions are pure so they are table-testable without a model or a store."""

import os
from dataclasses import dataclass

from pydantic import BaseModel


@dataclass(frozen=True)
class Pricing:
    """USD per million tokens. None = not configured."""

    input_per_mtok: float | None = None
    output_per_mtok: float | None = None


def load_pricing_from_env(prefix: str) -> Pricing:
    """`{PREFIX}_PRICE_INPUT_PER_MTOK` / `{PREFIX}_PRICE_OUTPUT_PER_MTOK`, both optional - the
    consumer passes the prefix of the provider that is active (e.g. "GOOGLE"), so swapping providers
    swaps the prices with it. Unset or blank -> None for that price. A value that is not a number
    raises at startup rather than silently reporting no cost."""
    return Pricing(
        input_per_mtok=_float_env(f"{prefix.upper()}_PRICE_INPUT_PER_MTOK"),
        output_per_mtok=_float_env(f"{prefix.upper()}_PRICE_OUTPUT_PER_MTOK"),
    )


def _float_env(name: str) -> float | None:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return None
    try:
        return float(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be a number (USD per million tokens), got {raw!r}") from exc


class TokenUsage(BaseModel):
    requests: int = 0
    input_tokens: int = 0
    output_tokens: int = 0
    tool_calls: int = 0


class UsageSummary(BaseModel):
    turn: TokenUsage
    session: TokenUsage
    # Cost of the whole session so far; None unless both prices are configured.
    cost_usd: float | None = None


def add_turn_usage(stored: dict[str, int], turn: TokenUsage) -> dict[str, int]:
    """The session's running totals after adding `turn`. `stored` is `session.data.get("usage", {})`
    - {} for a session that has not completed a run yet; it is not modified."""
    session = TokenUsage(**stored)
    return TokenUsage(
        requests=session.requests + turn.requests,
        input_tokens=session.input_tokens + turn.input_tokens,
        output_tokens=session.output_tokens + turn.output_tokens,
        tool_calls=session.tool_calls + turn.tool_calls,
    ).model_dump()


def cost_usd(usage: TokenUsage, pricing: Pricing | None) -> float | None:
    if pricing is None or pricing.input_per_mtok is None or pricing.output_per_mtok is None:
        return None
    return round((usage.input_tokens * pricing.input_per_mtok + usage.output_tokens * pricing.output_per_mtok) / 1_000_000, 6)


def summarize(stored: dict[str, int], turn: TokenUsage, pricing: Pricing | None) -> UsageSummary:
    """`stored` is the session totals ALREADY including `turn` (i.e. after add_turn_usage)."""
    session = TokenUsage(**stored)
    return UsageSummary(turn=turn, session=session, cost_usd=cost_usd(session, pricing))
