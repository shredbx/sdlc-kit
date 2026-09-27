"""Turns loaded Guidelines into the prose block the model reads - one more string passed to
Agent(instructions=[...]), same pattern as core.personality.render. Always renders every guideline
(no retrieval/filtering) - see base.py's own docstring for why that's the right call here."""

from agent_framework.core.guideline.base import Guideline


def render_guidelines_prompt(guidelines: tuple[Guideline, ...]) -> str:
    if not guidelines:
        return ""

    lines = ["Topic guidelines - how to handle common customer questions:"]
    for g in guidelines:
        lines.append("")
        lines.append(f"## {g.topic}")
        asks = "; ".join(f'"{a}"' for a in g.customer_asks)
        lines.append(f"Customer might ask: {asks}")
        if g.answer_from:
            lines.append(f"Get the real answer from the {g.answer_from} tool - never guess it.")
        for s in g.situations:
            lines.append(f"- If {s.when}: {s.reply_like}")
        for r in g.rules:
            lines.append(f"- Rule: {r}")
        if g.technique:
            lines.append(f"- Consider the {g.technique} technique here, if it genuinely fits.")

    return "\n".join(lines)
