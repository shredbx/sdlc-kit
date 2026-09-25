"""Regression test for a real bug: naive substring matching made "is" match inside "this"/"visit"
etc., so every query matched every entry. keyword_search must do whole-word matching."""

from agent_framework.core.knowledge.base import KnowledgeEntry, KnowledgeVariant
from agent_framework.core.knowledge.search import keyword_search

_ENTRIES = (
    KnowledgeEntry(
        id="pets",
        category="room-rental",
        canonical_question="Are pets allowed?",
        variations=("Is the house pet-friendly?",),
        tags=("pets", "animals"),
        variants=(KnowledgeVariant(when="default", answer_template="..."),),
    ),
    KnowledgeEntry(
        id="availability",
        category="room-rental",
        canonical_question="Is this still available?",
        variations=("Hello, available?",),
        tags=("availability",),
        variants=(KnowledgeVariant(when="default", answer_template="..."),),
    ),
)


def test_short_common_word_does_not_match_as_a_substring() -> None:
    # "is" must not match "this"/"visit" as a substring — this was the actual bug.
    assert keyword_search(_ENTRIES, "what is the meaning of life") == []


def test_real_keyword_matches_the_right_entry_only() -> None:
    results = keyword_search(_ENTRIES, "are pets allowed")
    assert [entry.id for entry in results] == ["pets"]


def test_no_words_after_stopword_filtering_returns_nothing() -> None:
    assert keyword_search(_ENTRIES, "is the a") == []
