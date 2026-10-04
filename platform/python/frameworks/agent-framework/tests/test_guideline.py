"""Proves the guideline loader reads real YAML correctly and the renderer produces the prose block
an Agent's instructions=[...] actually needs - not just that the dataclasses hold data."""

from pathlib import Path

import yaml
from agent_framework.core.guideline.base import Guideline, GuidelineSituation
from agent_framework.core.guideline.filesystem import load_guidelines
from agent_framework.core.guideline.render import render_guidelines_prompt

_AVAILABILITY_YAML = {
    "topic": "Is it still available?",
    "customer_asks": ["Is this still available?", "Is it available from [date]?"],
    "answer_from": "check_availability",
    "situations": [
        {"when": "available", "reply_like": "Yes, it's available! When would you like to check in?"},
        {"when": "not available", "reply_like": "Not available right now - could you share your dates and budget?"},
    ],
    "rules": ["Confirm right away, then ask about dates."],
    "technique": "Scarcity Signal",
}


def test_load_guidelines_reads_every_yaml_file_sorted(tmp_path: Path) -> None:
    (tmp_path / "availability.yaml").write_text(yaml.dump(_AVAILABILITY_YAML), encoding="utf-8")
    (tmp_path / "pricing.yaml").write_text(
        yaml.dump({"topic": "Price?", "customer_asks": ["What's the price?"], "situations": [{"when": "any", "reply_like": "Ask duration first."}]}),
        encoding="utf-8",
    )

    guidelines = load_guidelines(tmp_path)

    assert [g.topic for g in guidelines] == ["Is it still available?", "Price?"]
    assert guidelines[0] == Guideline(
        topic="Is it still available?",
        customer_asks=("Is this still available?", "Is it available from [date]?"),
        answer_from="check_availability",
        situations=(
            GuidelineSituation(when="available", reply_like="Yes, it's available! When would you like to check in?"),
            GuidelineSituation(when="not available", reply_like="Not available right now - could you share your dates and budget?"),
        ),
        rules=("Confirm right away, then ask about dates.",),
        technique="Scarcity Signal",
    )


def test_load_guidelines_defaults_optional_fields(tmp_path: Path) -> None:
    (tmp_path / "noise.yaml").write_text(
        yaml.dump({"topic": "Noise?", "customer_asks": ["Is it noisy?"], "situations": [{"when": "always", "reply_like": "We keep it quiet."}]}),
        encoding="utf-8",
    )

    guidelines = load_guidelines(tmp_path)

    assert guidelines[0].answer_from is None
    assert guidelines[0].rules == ()
    assert guidelines[0].technique is None


def test_render_includes_topic_tool_reference_situations_and_rules() -> None:
    guideline = Guideline(
        topic="Is it still available?",
        customer_asks=("Is this still available?",),
        answer_from="check_availability",
        situations=(GuidelineSituation(when="available", reply_like="Yes, it's available!"),),
        rules=("Confirm right away.",),
        technique="Scarcity Signal",
    )

    prompt = render_guidelines_prompt((guideline,))

    assert "Is it still available?" in prompt
    assert "check_availability tool" in prompt
    assert "Yes, it's available!" in prompt
    assert "Confirm right away." in prompt
    assert "Scarcity Signal" in prompt


def test_render_empty_guidelines_returns_empty_string() -> None:
    assert render_guidelines_prompt(()) == ""
