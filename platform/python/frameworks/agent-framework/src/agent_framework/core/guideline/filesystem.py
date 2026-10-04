"""Loads one Guideline per YAML file under a root folder - add a topic file, restart, no code
change. Sorted by filename so render order is stable and predictable from the folder listing."""

from pathlib import Path

import yaml

from agent_framework.core.guideline.base import Guideline, GuidelineSituation


def load_guidelines(root: Path) -> tuple[Guideline, ...]:
    return tuple(_load(path) for path in sorted(root.glob("*.yaml")))


def _load(path: Path) -> Guideline:
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    return Guideline(
        topic=data["topic"],
        customer_asks=tuple(data["customer_asks"]),
        situations=tuple(GuidelineSituation(when=s["when"], reply_like=s["reply_like"]) for s in data["situations"]),
        answer_from=data.get("answer_from"),
        rules=tuple(data.get("rules", [])),
        technique=data.get("technique"),
    )
