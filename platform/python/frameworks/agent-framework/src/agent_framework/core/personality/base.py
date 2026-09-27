"""Agent personality: the always-on part of a system prompt - tone traits and conversion/
handling techniques the model applies by its own judgement, never by tool call. Distinct from
tool-operational instructions (how to call search_faq/search_properties/handoff correctly), which
stay as plain code next to the tools they describe, not data - they're tightly coupled to real
tool/output schemas in a way prose data can drift out of sync with."""

from dataclasses import dataclass


@dataclass(frozen=True)
class PersonalityTrait:
    name: str
    description: str


@dataclass(frozen=True)
class ConversionTechnique:
    """A named way to help a hesitant customer decide, with one or more concrete example lines -
    not a trigger/response rule, since real conversations don't reduce to fixed pattern matches.
    The model picks which (if any) technique fits, the same way it already judges tone. Multiple
    examples (not just one) let several worked scenarios for the same technique live together,
    e.g. different ways to phrase a soft deadline depending on how the customer hesitated."""

    name: str
    when_to_use: str
    examples: tuple[str, ...]


@dataclass(frozen=True)
class Phrases:
    """Concrete phrasing guidance, not a rule about tone in the abstract - the same "use this,
    not that" shape a staff guide gives a new hire."""

    use: tuple[str, ...] = ()
    avoid: tuple[str, ...] = ()


@dataclass(frozen=True)
class Rules:
    """Short, direct behavioral rules - the DO/DON'T list shape, distinct from `techniques` (named,
    situational, applied by judgement) and from `Phrases` (wording, not behavior)."""

    do: tuple[str, ...] = ()
    dont: tuple[str, ...] = ()


@dataclass(frozen=True)
class HesitationReply:
    """One worked example: a real thing a customer says when hesitating, and how to reply to it -
    concrete enough to use as-is, unlike a technique's more general when_to_use."""

    customer_says: str
    reply: str


@dataclass(frozen=True)
class Personality:
    role: str
    scope: str
    traits: tuple[PersonalityTrait, ...]
    techniques: tuple[ConversionTechnique, ...]
    phrases: Phrases = Phrases()
    rules: Rules = Rules()
    hesitation_replies: tuple[HesitationReply, ...] = ()
    max_techniques_per_message: int = 2
