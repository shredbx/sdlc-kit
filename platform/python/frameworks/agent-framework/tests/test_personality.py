"""Proves the personality loader reads real YAML correctly and the renderer produces the prose
block an Agent's instructions=[...] actually needs - not just that the dataclasses hold data."""

from pathlib import Path

import yaml
from agent_framework.core.personality.base import ConversionTechnique, Personality, PersonalityTrait
from agent_framework.core.personality.filesystem import load_personality
from agent_framework.core.personality.render import render_personality_prompt

_YAML = {
    "role": "You are Bestie's assistant.",
    "traits": [
        {"name": "Warm & friendly", "description": 'Open with "Hello!"'},
        {"name": "Concise & direct", "description": "Short messages."},
    ],
    "techniques": [
        {"name": "Scarcity Signal", "when_to_use": "When others are genuinely interested too.", "example": "A few people have asked about this one."},
    ],
}


def test_load_personality_reads_real_yaml(tmp_path: Path) -> None:
    path = tmp_path / "personality.yaml"
    path.write_text(yaml.dump(_YAML), encoding="utf-8")

    personality = load_personality(path)

    assert personality.role == "You are Bestie's assistant."
    assert personality.traits == (
        PersonalityTrait(name="Warm & friendly", description='Open with "Hello!"'),
        PersonalityTrait(name="Concise & direct", description="Short messages."),
    )
    assert personality.techniques == (
        ConversionTechnique(name="Scarcity Signal", when_to_use="When others are genuinely interested too.", example="A few people have asked about this one."),
    )


def test_load_personality_with_no_techniques_section(tmp_path: Path) -> None:
    path = tmp_path / "personality.yaml"
    path.write_text(yaml.dump({"role": "x", "traits": []}), encoding="utf-8")

    assert load_personality(path).techniques == ()


def test_render_includes_role_and_every_trait() -> None:
    personality = Personality(
        role="You are Bestie's assistant.",
        traits=(PersonalityTrait(name="Warm & friendly", description='Open with "Hello!"'),),
        techniques=(),
    )

    prompt = render_personality_prompt(personality)

    assert "You are Bestie's assistant." in prompt
    assert 'Warm & friendly — Open with "Hello!"' in prompt
    assert "Helping a hesitant customer" not in prompt  # no techniques - section omitted, not empty


def test_render_includes_techniques_when_present() -> None:
    personality = Personality(
        role="You are Bestie's assistant.",
        traits=(),
        techniques=(ConversionTechnique(name="Scarcity Signal", when_to_use="When true.", example="A few people asked."),),
    )

    prompt = render_personality_prompt(personality)

    assert "Scarcity Signal" in prompt
    assert "A few people asked." in prompt
