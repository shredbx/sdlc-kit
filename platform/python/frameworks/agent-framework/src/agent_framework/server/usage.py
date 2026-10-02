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


# The rules a consumer can ask for. WARN only reports the limit to the client, which decides what to
# show - nothing here refuses or cuts a turn. A blocking rule is added to this tuple later without
# touching clients that already understand WARN.
SUPPORTED_LIMIT_RULES = ("WARN",)
DEFAULT_LIMIT_RULE = "WARN"


class TokenLimit(BaseModel):
    """A session's token allowance: total input + output tokens over all its model requests."""

    tokens: int
    rule: str = DEFAULT_LIMIT_RULE


def load_token_limit_from_env() -> TokenLimit | None:
    """`SESSION_TOKEN_LIMIT` (a whole number >= 1; unset or blank = no limit) and
    `SESSION_TOKEN_LIMIT_RULE` (default WARN, the only rule supported so far). A value that is not
    understood raises at startup rather than silently running without the limit."""
    rule = os.environ.get("SESSION_TOKEN_LIMIT_RULE", "").strip().upper() or DEFAULT_LIMIT_RULE
    if rule not in SUPPORTED_LIMIT_RULES:
        raise ValueError(f"SESSION_TOKEN_LIMIT_RULE must be one of {', '.join(SUPPORTED_LIMIT_RULES)}, got {rule!r}")

    raw = os.environ.get("SESSION_TOKEN_LIMIT", "").strip()
    if not raw:
        return None
    try:
        tokens = int(raw)
    except ValueError as exc:
        raise ValueError(f"SESSION_TOKEN_LIMIT must be a whole number of tokens, got {raw!r}") from exc
    if tokens < 1:
        raise ValueError(f"SESSION_TOKEN_LIMIT must be at least 1 (leave it empty for no limit), got {raw!r}")
    return TokenLimit(tokens=tokens, rule=rule)


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
    # The session allowance the client measures `session` against; None = no limit configured.
    limit: TokenLimit | None = None


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


def summarize(stored: dict[str, int], turn: TokenUsage, pricing: Pricing | None, token_limit: TokenLimit | None = None) -> UsageSummary:
    """`stored` is the session totals ALREADY including `turn` (i.e. after add_turn_usage)."""
    session = TokenUsage(**stored)
    return UsageSummary(turn=turn, session=session, cost_usd=cost_usd(session, pricing), limit=token_limit)
