"""Proves the personality loader reads real YAML correctly and the renderer produces the prose
block an Agent's instructions=[...] actually needs - not just that the dataclasses hold data."""

from pathlib import Path

import yaml
from agent_framework.core.personality.base import (
    ConversionTechnique,
    HesitationReply,
    Personality,
    PersonalityTrait,
    Phrases,
    Rules,
)
from agent_framework.core.personality.filesystem import load_personality
from agent_framework.core.personality.render import render_personality_prompt

_YAML = {
    "role": "You are Bestie's assistant.",
    "scope": "room and house rentals on Koh Phangan",
    "traits": [
        {"name": "Warm & friendly", "description": 'Open with "Hello!"'},
        {"name": "Concise & direct", "description": "Short messages."},
    ],
    "techniques": [
        {
            "name": "Scarcity Signal",
            "when_to_use": "When others are genuinely interested too.",
            "examples": ["A few people have asked about this one.", "This one tends to move quickly."],
        },
    ],
    "phrases": {"use": ["Pls"], "avoid": ["Please confirm at your earliest convenience"]},
    "rules": {"do": ["List every cost upfront"], "dont": ["Quote the lowest price first"]},
    "hesitation_replies": [{"customer_says": "I'll think about it", "reply": "No worries — I can hold it for 24 hours."}],
}


def test_load_personality_reads_real_yaml(tmp_path: Path) -> None:
    path = tmp_path / "personality.yaml"
    path.write_text(yaml.dump(_YAML), encoding="utf-8")

    personality = load_personality(path)

    assert personality.role == "You are Bestie's assistant."
    assert personality.scope == "room and house rentals on Koh Phangan"
    assert personality.traits == (
        PersonalityTrait(name="Warm & friendly", description='Open with "Hello!"'),
        PersonalityTrait(name="Concise & direct", description="Short messages."),
    )
    assert personality.techniques == (
        ConversionTechnique(
            name="Scarcity Signal",
            when_to_use="When others are genuinely interested too.",
            examples=("A few people have asked about this one.", "This one tends to move quickly."),
        ),
    )
    assert personality.phrases == Phrases(use=("Pls",), avoid=("Please confirm at your earliest convenience",))
    assert personality.rules == Rules(do=("List every cost upfront",), dont=("Quote the lowest price first",))
    assert personality.hesitation_replies == (
        HesitationReply(customer_says="I'll think about it", reply="No worries — I can hold it for 24 hours."),
    )
    assert personality.max_techniques_per_message == 2


def test_load_personality_with_only_required_fields(tmp_path: Path) -> None:
    path = tmp_path / "personality.yaml"
    path.write_text(yaml.dump({"role": "x", "scope": "y", "traits": []}), encoding="utf-8")

    personality = load_personality(path)

    assert personality.techniques == ()
    assert personality.phrases == Phrases()
    assert personality.rules == Rules()
    assert personality.hesitation_replies == ()


def test_render_includes_role_scope_and_every_trait() -> None:
    personality = Personality(
        role="You are Bestie's assistant.",
        scope="room and house rentals",
        traits=(PersonalityTrait(name="Warm & friendly", description='Open with "Hello!"'),),
        techniques=(),
    )

    prompt = render_personality_prompt(personality)

    assert "You are Bestie's assistant." in prompt
    assert "room and house rentals" in prompt
    assert 'Warm & friendly — Open with "Hello!"' in prompt
    assert "Helping a hesitant customer" not in prompt  # no techniques - section omitted, not empty


def test_render_includes_techniques_when_present() -> None:
    personality = Personality(
        role="You are Bestie's assistant.",
        scope="rentals",
        traits=(),
        techniques=(ConversionTechnique(name="Scarcity Signal", when_to_use="When true.", examples=("A few people asked.", "It moves quickly.")),),
    )

    prompt = render_personality_prompt(personality)

    assert "Scarcity Signal" in prompt
    assert "A few people asked." in prompt
    assert "It moves quickly." in prompt


def test_render_includes_phrases_rules_and_hesitation_replies() -> None:
    personality = Personality(
        role="You are Bestie's assistant.",
        scope="rentals",
        traits=(),
        techniques=(),
        phrases=Phrases(use=("Pls",), avoid=("Please confirm at your earliest convenience",)),
        rules=Rules(do=("List every cost upfront",), dont=("Quote the lowest price first",)),
        hesitation_replies=(HesitationReply(customer_says="I'll think about it", reply="I can hold it for 24 hours."),),
    )

    prompt = render_personality_prompt(personality)

    assert '"Pls"' in prompt
    assert "List every cost upfront" in prompt
    assert "Quote the lowest price first" in prompt
    assert "I'll think about it" in prompt
    assert "I can hold it for 24 hours." in prompt


def test_render_omits_phrases_rules_and_hesitation_sections_when_absent() -> None:
    personality = Personality(role="x", scope="y", traits=(), techniques=())

    prompt = render_personality_prompt(personality)

    assert "Phrasing:" not in prompt
    assert "Rules:" not in prompt
    assert "hesitates" not in prompt
