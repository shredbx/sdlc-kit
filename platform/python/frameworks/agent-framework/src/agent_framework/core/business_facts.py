"""Single-source-of-truth business values (a phone number, a website URL) referenced by name from
Persona/Guideline/tool-config text instead of being retyped in every place that mentions them -
the direct fix for a real bug found in the bestays consumer, where a phone number was hand-typed
into a dozen+ places and drifted (docs/plans/2026-09-28-chat-agent-system-review-and-delivery-plan.md,
finding D2/S1)."""

import re
from dataclasses import dataclass
from pathlib import Path

import yaml

# ${key}, not {key} - plain {word} already appears in ordinary tool-usage prose (e.g. a link
# template like "/p/{id}", describing output construction to the model, not a business fact) and
# would otherwise collide. ${...} is unambiguous.
_PLACEHOLDER = re.compile(r"\$\{(\w+)\}")


@dataclass(frozen=True)
class BusinessFacts:
    values: dict[str, str]


def load_business_facts(path: Path) -> BusinessFacts:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    return BusinessFacts(values={str(k): str(v) for k, v in data.items()})


def resolve_facts(text: str, facts: BusinessFacts) -> str:
    """Replaces every ${key} in `text` with facts.values[key]. Raises KeyError naming the missing
    key rather than silently leaving a literal "${whatsapp_number}" in a real prompt or reply."""

    def _sub(match: re.Match[str]) -> str:
        key = match.group(1)
        if key not in facts.values:
            raise KeyError(f"business fact {key!r} referenced but not defined")
        return facts.values[key]

    return _PLACEHOLDER.sub(_sub, text)
