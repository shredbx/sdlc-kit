"""Loads a Personality from one YAML file - unlike knowledge/filesystem_store.py's one-file-per-
entry, personality is a single cohesive document (role, traits, techniques all describe one
consistent voice), matching how the real source material itself is organized."""

from pathlib import Path

import yaml

from agent_framework.core.personality.base import ConversionTechnique, Personality, PersonalityTrait


def load_personality(path: Path) -> Personality:
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    return Personality(
        role=data["role"],
        traits=tuple(PersonalityTrait(name=t["name"], description=t["description"]) for t in data["traits"]),
        techniques=tuple(ConversionTechnique(name=t["name"], when_to_use=t["when_to_use"], example=t["example"]) for t in data.get("techniques", [])),
    )
