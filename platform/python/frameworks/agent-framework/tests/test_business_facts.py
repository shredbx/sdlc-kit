"""Proves business facts load from real YAML and ${key} substitution works, including refusing to
silently leave an unresolved placeholder in text a customer or the model would actually see - and
that plain {word} text (e.g. a link template like "/p/{id}") is left alone, not mistaken for one."""

from pathlib import Path

import pytest
import yaml
from agent_framework.core.business_facts import BusinessFacts, load_business_facts, resolve_facts


def test_load_business_facts_reads_real_yaml(tmp_path: Path) -> None:
    path = tmp_path / "business_facts.yaml"
    path.write_text(yaml.dump({"whatsapp_number": "+66983480288", "website_listings_url": "https://example.com"}), encoding="utf-8")

    facts = load_business_facts(path)

    assert facts.values == {"whatsapp_number": "+66983480288", "website_listings_url": "https://example.com"}


def test_load_business_facts_empty_file(tmp_path: Path) -> None:
    path = tmp_path / "business_facts.yaml"
    path.write_text("", encoding="utf-8")

    assert load_business_facts(path).values == {}


def test_resolve_facts_substitutes_every_placeholder() -> None:
    facts = BusinessFacts(values={"whatsapp_number": "+66983480288"})

    result = resolve_facts("Message us on ${whatsapp_number} anytime.", facts)

    assert result == "Message us on +66983480288 anytime."


def test_resolve_facts_raises_on_unknown_key() -> None:
    facts = BusinessFacts(values={"whatsapp_number": "+66983480288"})

    with pytest.raises(KeyError, match="website_url"):
        resolve_facts("Visit ${website_url}.", facts)


def test_resolve_facts_leaves_plain_braces_untouched() -> None:
    facts = BusinessFacts(values={"whatsapp_number": "+66983480288"})

    result = resolve_facts('a link of "/p/{id}", not a business fact', facts)

    assert result == 'a link of "/p/{id}", not a business fact'
