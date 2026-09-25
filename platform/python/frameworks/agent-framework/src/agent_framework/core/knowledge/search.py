"""Keyword-overlap matching — a stand-in for confidence-banded embedding retrieval, reusable
across any consumer with KnowledgeEntry-shaped content until that's actually built."""

import re
from collections.abc import Sequence

from agent_framework.core.knowledge.base import KnowledgeEntry

_WORD = re.compile(r"[a-z0-9']+")

_STOPWORDS = {
    "a", "an", "the", "is", "are", "was", "were", "be", "been", "am",
    "i", "you", "he", "she", "it", "we", "they", "this", "that", "these", "those",
    "do", "does", "did", "can", "could", "will", "would", "should",
    "what", "how", "when", "where", "why", "who",
    "in", "on", "at", "to", "for", "of", "with", "and", "or", "not",
    "my", "your", "there",
}


def _words(text: str) -> set[str]:
    return set(_WORD.findall(text.lower()))


def keyword_search(entries: Sequence[KnowledgeEntry], query: str) -> list[KnowledgeEntry]:
    query_words = _words(query) - _STOPWORDS
    if not query_words:
        return []

    def matches(entry: KnowledgeEntry) -> bool:
        haystack = _words(" ".join([entry.canonical_question, *entry.variations, *entry.tags]))
        return bool(query_words & haystack)

    return [entry for entry in entries if matches(entry)]
