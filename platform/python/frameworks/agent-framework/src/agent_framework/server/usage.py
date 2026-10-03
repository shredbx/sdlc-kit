"""Token usage per turn and per session, and what it costs. The running per-session totals live in
`Session.data["usage"]` - the same free-form per-session bag core/limits.py already uses, no new
storage - and are summed here from pydantic-ai's own RunUsage after each completed run.

The cost is what each model request cost, as the model that answered it reports it (pydantic-ai fills
`ModelResponse.usage.cost` from its price table, and a consumer can set it itself for a model the table
does not know). There is no price configured here: a session can be answered by several models in turn.
A request with no known cost is counted, never guessed, and one of them makes the whole session's cost
unknown - a client then shows tokens only. All functions are pure so they are table-testable without a
model or a store."""

import os
from collections.abc import Iterable
from dataclasses import dataclass
from decimal import Decimal

from pydantic import BaseModel


@dataclass(frozen=True)
class TurnCost:
    """What a turn's model requests cost: the sum of the known costs, and how many requests had none."""

    usd: float = 0.0
    unpriced_requests: int = 0

    @property
    def reported_usd(self) -> float | None:
        """The cost to report: None as soon as one request had no known cost (a partial sum would be a guess)."""
        return None if self.unpriced_requests else round(self.usd, 6)


def turn_cost(request_costs: Iterable[Decimal | None]) -> TurnCost:
    """`request_costs` is `ModelResponse.usage.cost` of every request of the turn; None = no price known for that model."""
    costs = list(request_costs)
    known = [cost for cost in costs if cost is not None]
    return TurnCost(usd=float(sum(known, Decimal(0))), unpriced_requests=len(costs) - len(known))


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
    # Cost of the whole session so far; None when any of its requests had no known cost.
    cost_usd: float | None = None
    # The session allowance the client measures `session` against; None = no limit configured.
    limit: TokenLimit | None = None


def add_turn_usage(stored: dict[str, int | float], turn: TokenUsage, cost: TurnCost) -> dict[str, int | float]:
    """The session's running totals after adding `turn` and what it cost. `stored` is `session.data.get("usage", {})`
    - {} for a session that has not completed a run yet; it is not modified. A session saved before costs were
    recorded has no cost keys: its earlier requests count as unpriced, so its cost stays unknown."""
    session = TokenUsage(**stored)
    unpriced_before = stored.get("unpriced_requests", 0 if "cost_usd" in stored else session.requests)
    return {
        **TokenUsage(
            requests=session.requests + turn.requests,
            input_tokens=session.input_tokens + turn.input_tokens,
            output_tokens=session.output_tokens + turn.output_tokens,
            tool_calls=session.tool_calls + turn.tool_calls,
        ).model_dump(),
        "cost_usd": round(stored.get("cost_usd", 0.0) + cost.usd, 9),
        "unpriced_requests": int(unpriced_before) + cost.unpriced_requests,
    }


def session_cost(stored: dict[str, int | float]) -> float | None:
    """The session's cost so far, or None when it is not fully known."""
    if "cost_usd" not in stored or stored.get("unpriced_requests", 0):
        return None
    return round(float(stored["cost_usd"]), 6)


def summarize(stored: dict[str, int | float], turn: TokenUsage, token_limit: TokenLimit | None = None) -> UsageSummary:
    """`stored` is the session totals ALREADY including `turn` (i.e. after add_turn_usage)."""
    return UsageSummary(turn=turn, session=TokenUsage(**stored), cost_usd=session_cost(stored), limit=token_limit)
