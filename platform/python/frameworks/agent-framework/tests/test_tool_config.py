"""Proves tool config loads from real YAML - the same usage_guidance a Python constant used to
hold, just editable as data now."""

from pathlib import Path

import yaml
from agent_framework.core.tool_config import ToolConfig, load_tool_config


def test_load_tool_config_reads_usage_guidance_and_enabled(tmp_path: Path) -> None:
    path = tmp_path / "search_properties.yaml"
    path.write_text(yaml.dump({"usage_guidance": "Ask about budget first.", "enabled": False}), encoding="utf-8")

    config = load_tool_config(path)

    assert config == ToolConfig(usage_guidance="Ask about budget first.", enabled=False)


def test_load_tool_config_defaults_enabled_to_true(tmp_path: Path) -> None:
    path = tmp_path / "handoff.yaml"
    path.write_text(yaml.dump({"usage_guidance": "Hand off once there's real interest."}), encoding="utf-8")

    assert load_tool_config(path).enabled is True
