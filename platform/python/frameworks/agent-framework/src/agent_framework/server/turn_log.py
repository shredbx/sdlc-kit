"""One compact, structured record per chat turn - the line you read to see what the service is doing and
what it costs, instead of digging through full trace spans (which hold prompts and customer text and
are an order of magnitude bigger). Never contains message text, replies, prompts or tool results: only
who/what/how long/how many tokens/how it ended.

`turn_record` is pure (testable without a run); `log_turn` writes it as one JSON line to the
`agent_framework.turn` logger; `configure_turn_log` points that logger at stdout (a container's log
stream) - a consumer calls it once at startup."""

import json
import logging
import sys
from datetime import UTC, datetime
from typing import Any, TextIO

from agent_framework.server.usage import Pricing, TokenUsage, cost_usd

LOGGER_NAME = "agent_framework.turn"


def turn_record(
    *,
    agent: str,
    session_id: str,
    user_id: str | None,
    client_context: dict[str, Any] | None,
    streamed: bool,
    latency_seconds: float,
    status: str,  # ok | error | cancelled | rejected
    code: str | None = None,
    usage: TokenUsage | None = None,
    pricing: Pricing | None = None,
    now: datetime | None = None,
) -> dict[str, Any]:
    source = (client_context or {}).get("user_source")
    return {
        "event": "chat_turn",
        "ts": (now or datetime.now(UTC)).isoformat(timespec="seconds"),
        "agent": agent,
        "session": session_id[:8],  # enough to follow one conversation in the logs, not the full id
        "user_id": user_id,
        "source": source if isinstance(source, str) else None,
        "stream": streamed,
        "status": status,
        "code": code,
        "latency_ms": round(latency_seconds * 1000),
        "requests": usage.requests if usage else 0,
        "tool_calls": usage.tool_calls if usage else 0,
        "input_tokens": usage.input_tokens if usage else 0,
        "output_tokens": usage.output_tokens if usage else 0,
        "cost_usd": cost_usd(usage, pricing) if usage else None,
    }


def log_turn(record: dict[str, Any]) -> None:
    logging.getLogger(LOGGER_NAME).info(json.dumps(record, ensure_ascii=False))


def configure_turn_log(stream: TextIO | None = None) -> None:
    """Send the turn records, bare (one JSON object per line), to `stream` (default stdout)."""
    logger = logging.getLogger(LOGGER_NAME)
    logger.setLevel(logging.INFO)
    logger.propagate = False  # don't also print them through the root logger
    logger.handlers.clear()
    handler = logging.StreamHandler(stream or sys.stdout)
    handler.setFormatter(logging.Formatter("%(message)s"))
    logger.addHandler(handler)
