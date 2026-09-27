"""Loads a tool's usage_guidance from YAML instead of a hardcoded Python constant -
ToolEntry.usage_guidance is unchanged, only its source changes, so a console (M3) can edit it as
data. See docs/plans/2026-09-28-chat-agent-configuration-spec.md, section 4, for the design."""

from dataclasses import dataclass
from pathlib import Path

import yaml


@dataclass(frozen=True)
class ToolConfig:
    usage_guidance: str
    enabled: bool = True


def load_tool_config(path: Path) -> ToolConfig:
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    return ToolConfig(usage_guidance=data["usage_guidance"], enabled=data.get("enabled", True))
