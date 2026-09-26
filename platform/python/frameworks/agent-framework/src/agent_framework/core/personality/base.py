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
    """A named way to help a hesitant customer decide, with a concrete example line - not a
    trigger/response rule, since real conversations don't reduce to fixed pattern matches. The
    model picks which (if any) technique fits, the same way it already judges tone."""

    name: str
    when_to_use: str
    example: str


@dataclass(frozen=True)
class Personality:
    role: str
    traits: tuple[PersonalityTrait, ...]
    techniques: tuple[ConversionTechnique, ...]
