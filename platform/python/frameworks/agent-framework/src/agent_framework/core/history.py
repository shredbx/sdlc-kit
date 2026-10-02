"""Keeps a conversation's message history to a bounded size, for `pydantic_ai.capabilities.ProcessHistory`.

Why: the whole history is sent to the model with every request, so a long session pays for its old turns on every
reply (and eventually outgrows the model's window). What a consumer needs to remember from old turns belongs in the
session's own state (re-sent each request as instructions), not in a transcript the model re-reads.

The rule is deliberately plain:
- a TURN starts at a user prompt and runs up to the next one - it holds the model's tool calls and their results, so
  a cut is only ever made at the start of a turn and a tool call is never separated from its result;
- the newest turn is always kept whole, even over the limits (it may be the run in progress, mid tool call);
- older turns are kept while the number of turns and their text size both stay within the limits, newest first, and
  the first one that does not fit ends it (never a gap of dropped turns inside what is kept);
- anything before the first turn (a system prompt, say) is not part of any turn and is kept.

`ProcessHistory` runs before every model request of a run, and the history the framework saves afterwards is the
processed one - so the stored session stays bounded too, not just what is sent. Pure functions, no model needed."""

import json
from collections.abc import Callable, Sequence
from typing import Any

from pydantic_ai.messages import (
    ModelMessage,
    ModelRequest,
    TextPart,
    ToolCallPart,
    ToolReturnPart,
    UserPromptPart,
)

HistoryProcessor = Callable[[list[ModelMessage]], list[ModelMessage]]


def trim_history(max_turns: int | None, max_chars: int | None) -> HistoryProcessor:
    """A processor keeping at most `max_turns` turns and about `max_chars` characters of text (about 4 per token);
    None = no limit on that. The newest turn is always kept."""

    def process(messages: list[ModelMessage]) -> list[ModelMessage]:
        starts = [i for i, message in enumerate(messages) if starts_turn(message)]
        if len(starts) <= 1:
            return messages  # nothing older than the newest turn to drop

        # Newest first: [(start, end)] of each turn, the newest turn's end being the end of the list.
        bounds = [(start, end) for start, end in zip(starts, [*starts[1:], len(messages)], strict=True)][::-1]
        kept_from = bounds[0][0]  # the newest turn, whatever its size
        turns, chars = 1, sum(size_of(m) for m in messages[kept_from:])
        for start, end in bounds[1:]:
            size = sum(size_of(m) for m in messages[start:end])
            if (max_turns is not None and turns + 1 > max_turns) or (max_chars is not None and chars + size > max_chars):
                break
            turns, chars, kept_from = turns + 1, chars + size, start
        return messages[: starts[0]] + messages[kept_from:]

    return process


def starts_turn(message: ModelMessage) -> bool:
    """A request carrying a user prompt. A request holding only tool results continues the turn."""
    return isinstance(message, ModelRequest) and any(isinstance(part, UserPromptPart) for part in message.parts)


def size_of(message: ModelMessage) -> int:
    """Characters of text in one message: what the user and model wrote, tool arguments and tool results."""
    return sum(_part_size(part) for part in message.parts)


def _part_size(part: Any) -> int:
    if isinstance(part, UserPromptPart):
        return _content_size(part.content)
    if isinstance(part, TextPart):
        return len(part.content)
    if isinstance(part, ToolCallPart):
        return len(part.args_as_json_str())
    if isinstance(part, ToolReturnPart):
        return len(part.model_response_str())
    content = getattr(part, "content", None)  # retry prompts, system prompts, anything else with text
    return _content_size(content) if content is not None else 0


def _content_size(content: str | Sequence[Any] | Any) -> int:
    if isinstance(content, str):
        return len(content)
    if isinstance(content, Sequence):
        return sum(_content_size(item) for item in content)
    try:
        return len(json.dumps(content, default=str))
    except (TypeError, ValueError):
        return len(str(content))
