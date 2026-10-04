"""Loads a Personality from one YAML file - unlike knowledge/filesystem_store.py's one-file-per-
entry, personality is a single cohesive document (role, traits, techniques all describe one
consistent voice), matching how the real source material itself is organized."""

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


def load_personality(path: Path) -> Personality:
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    phrases_data = data.get("phrases", {})
    rules_data = data.get("rules", {})
    return Personality(
        role=data["role"],
        scope=data["scope"],
        traits=tuple(PersonalityTrait(name=t["name"], description=t["description"]) for t in data["traits"]),
        techniques=tuple(
            ConversionTechnique(name=t["name"], when_to_use=t["when_to_use"], examples=tuple(t["examples"]))
            for t in data.get("techniques", [])
        ),
        phrases=Phrases(use=tuple(phrases_data.get("use", [])), avoid=tuple(phrases_data.get("avoid", []))),
        rules=Rules(do=tuple(rules_data.get("do", [])), dont=tuple(rules_data.get("dont", []))),
        hesitation_replies=tuple(
            HesitationReply(customer_says=h["customer_says"], reply=h["reply"]) for h in data.get("hesitation_replies", [])
        ),
        max_techniques_per_message=data.get("max_techniques_per_message", 2),
    )
