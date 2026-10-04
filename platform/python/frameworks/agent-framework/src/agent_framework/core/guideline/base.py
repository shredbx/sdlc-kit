"""A Guideline is per-customer-topic prompt content - deliberately separate from
core.knowledge.KnowledgeEntry (a retrieval-tool-shaped concept the model chooses to look up).
Guidelines are always-eager: most customer messages match one, so per PydanticAI's own "what to
load eagerly" guidance, they belong in the static system prompt, not behind a tool call the model
might skip. See docs/plans/2026-09-28-chat-agent-configuration-spec.md, section 2, for the design
this implements."""

from dataclasses import dataclass


@dataclass(frozen=True)
class GuidelineSituation:
    """One conditional reply within a topic - e.g. "available" vs "not available" for the same
    underlying question. `when` is a plain-language condition for the model to judge, not code."""

    when: str
    reply_like: str


@dataclass(frozen=True)
class Guideline:
    topic: str
    customer_asks: tuple[str, ...]
    situations: tuple[GuidelineSituation, ...]
    answer_from: str | None = None
    rules: tuple[str, ...] = ()
    technique: str | None = None
