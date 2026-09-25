"""Knowledge content storage (FAQ entries and similar): an adapter interface, plus filesystem/
postgres implementations — same shape as core/store. Filesystem is the local-dev/default adapter;
postgres is a later swap behind the same interface."""

from abc import ABC, abstractmethod
from dataclasses import dataclass


@dataclass(frozen=True)
class KnowledgeVariant:
    """One conditional answer within an entry — e.g. "available" vs "not available" for the same
    underlying question. `when` is a plain-language condition for the model to judge, not code."""

    when: str
    answer_template: str
    tone_note: str | None = None


@dataclass(frozen=True)
class KnowledgeEntry:
    id: str
    category: str
    canonical_question: str
    variations: tuple[str, ...]
    tags: tuple[str, ...]
    variants: tuple[KnowledgeVariant, ...]


class KnowledgeStore(ABC):
    @abstractmethod
    def all_entries(self) -> tuple[KnowledgeEntry, ...]: ...
