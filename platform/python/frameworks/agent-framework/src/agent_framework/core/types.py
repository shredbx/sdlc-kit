"""Shared types for the framework's agent registry."""

from collections.abc import Callable
from dataclasses import dataclass
from typing import Any, Literal

from pydantic_ai import Agent

from agent_framework.core.personality.base import Personality

# The channels this framework has an actual entry point for. A closed Literal, not a free string -
# adding a new channel is a one-line change here, and every place that switches on channel (e.g.
# output_protocol below, or a future channel-aware handoff) gets a typo-checked exhaustiveness
# reminder from the type checker instead of a silent no-op on a misspelled string.
Channel = Literal["website", "facebook_personal", "whatsapp_personal", "facebook_business", "whatsapp_business"]

# rich_cards: the consumer's own UI renders Card objects (property listings, a handoff link) -
# today's website widget. plain_text: the consumer can only display/paste plain text (a Messenger
# or WhatsApp compose box), so only AgentReply.text is usable - Card data still comes back from a
# tool call, the *consumer* just knows not to render it. This is a declared capability of the
# channel, not a per-request choice.
OutputProtocol = Literal["rich_cards", "plain_text"]

# public: no verification (today's unchanged default - e.g. the website widget, open to anonymous
# visitors). required: routes/chat.py demands a valid Authorization: Bearer <token> header,
# verified via the UserVerifier (core/auth.py) supplied to create_app/build_router, before the
# agent runs - e.g. facebook_personal, where a real Bestays manager (not an anonymous visitor) is
# the one using the Chrome extension, and per-user attribution (Session.user_id) actually matters.
AuthMode = Literal["public", "required"]


@dataclass(frozen=True)
class RegisteredAgent:
    """One agent discovered under an `agents/` folder: its built PydanticAI Agent, a factory for
    its per-request deps, and the auth mode gating access to it. `name` matches the agent's own
    folder name, not necessarily `agent.name` (which is only a Logfire trace label).

    `channel` and `output_protocol` exist so a consumer can register the *same* underlying agent
    behavior under several channel-specific entry points (e.g. `agents/chat/` for the website vs
    `agents/facebook_personal/` for the Chrome extension) and have each one declare what it is and
    what it can render, without the client having to know or guess. Whether a channel needs its own
    tool set (e.g. disabling `handoff` where a human is already the handoff, see tools/handoff/) is
    handled today by giving that channel its own RegisteredAgent with its own `tools=build_tools(...)`
    call - not a new mechanism, reuse of the one ToolEntry.enabled already provides.

    `personality` and `prompt_fields` are optional, purely for introspection (see
    server/routes/introspect.py) - the agent itself only ever reads its own already-composed
    `instructions=[...]`, never these. A consumer that built its prompt from a Personality plus
    other named pieces (e.g. operational_instructions) can hand both back here so a dev console
    can show the individual fields, not just the final joined string. `prompt_fields` is
    deliberately a plain label->text dict, not a new framework type - there's exactly one real
    example of a second prompt piece (operational_instructions) to generalize from so far."""

    name: str
    agent: Agent[Any, Any]
    build_deps: Callable[[Any], Any]
    auth: AuthMode = "public"
    personality: Personality | None = None
    prompt_fields: dict[str, str] | None = None
    channel: Channel = "website"
    output_protocol: OutputProtocol = "rich_cards"
    # Optional: given the finished turn's Session, returns a small JSON-able summary of what the
    # agent has captured so far (e.g. booking and visit preferences), or None. The streaming route
    # hands it to the client in the final `done` event so a UI can show it; the framework never
    # interprets it. None -> the `done` event's `state` is null.
    state_summary: Callable[[Any], dict[str, Any] | None] | None = None
