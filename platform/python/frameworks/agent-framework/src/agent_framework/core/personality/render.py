"""Turns a loaded Personality into the prose block the model actually reads - one of the (usually
two) strings passed to Agent(instructions=[...]); PydanticAI concatenates a sequence of static
instructions itself, so this only needs to produce this one piece, not compose the final prompt."""

from agent_framework.core.personality.base import Personality


def render_personality_prompt(personality: Personality) -> str:
    lines = [personality.role, "", "Tone of voice:"]
    lines += [f"- {trait.name} — {trait.description}" for trait in personality.traits]

    if personality.techniques:
        lines += ["", "Helping a hesitant customer decide (use at most one or two per message, only once there's real interest):"]
        lines += [f'- {t.name} — {t.when_to_use} Example: "{t.example}"' for t in personality.techniques]

    return "\n".join(lines)
