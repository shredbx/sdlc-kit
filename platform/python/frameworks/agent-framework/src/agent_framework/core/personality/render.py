"""Turns a loaded Personality into the prose block the model actually reads - one of the (usually
two) strings passed to Agent(instructions=[...]); PydanticAI concatenates a sequence of static
instructions itself, so this only needs to produce this one piece, not compose the final prompt."""

from agent_framework.core.personality.base import Personality


def render_personality_prompt(personality: Personality) -> str:
    lines = [
        personality.role,
        "",
        f"You only discuss {personality.scope} and what your tools/guidelines cover. If asked about "
        "anything else, say so briefly and steer the conversation back to how you can help.",
        "",
        "Tone of voice:",
    ]
    lines += [f"- {trait.name} — {trait.description}" for trait in personality.traits]

    if personality.phrases.use or personality.phrases.avoid:
        lines += ["", "Phrasing:"]
        lines += [f'- Use: "{p}"' for p in personality.phrases.use]
        lines += [f'- Avoid: "{p}"' for p in personality.phrases.avoid]

    if personality.rules.do or personality.rules.dont:
        lines += ["", "Rules:"]
        lines += [f"- Do: {r}" for r in personality.rules.do]
        lines += [f"- Don't: {r}" for r in personality.rules.dont]

    if personality.techniques:
        lines += [
            "",
            f"Helping a hesitant customer decide (use at most {personality.max_techniques_per_message} per "
            "message, only once there's real interest):",
        ]
        for t in personality.techniques:
            examples = "; ".join(f'"{e}"' for e in t.examples)
            lines.append(f"- {t.name} — {t.when_to_use} Examples: {examples}")

    if personality.hesitation_replies:
        lines += ["", "If the customer hesitates or goes quiet, examples of how to respond:"]
        lines += [f'- "{h.customer_says}" → "{h.reply}"' for h in personality.hesitation_replies]

    return "\n".join(lines)
