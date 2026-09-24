"""Registers a scaffolded plugin with this repository's own tooling: the marketplace `/plugin
marketplace add` reads, and the uv workspace, which a plugin's own folder must stay out of, since a
plugin has no `pyproject.toml` to join it with. Run by `post.sh`, given the repository root, the
plugin's name and its folder from that root. Both writes are idempotent."""

import json
import re
import sys
from pathlib import Path


def _marketplace(root: Path, name: str) -> None:
    file = root / ".claude-plugin" / "marketplace.json"
    file.parent.mkdir(parents=True, exist_ok=True)
    data = (
        json.loads(file.read_text(encoding="utf-8"))
        if file.is_file()
        else {"name": "process-os", "owner": {"name": "process-os"}, "plugins": []}
    )
    if not any(plugin.get("name") == name for plugin in data["plugins"]):
        data["plugins"].append({"name": name, "source": f"./products/{name}"})
    file.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def _excluded(root: Path, target: str) -> None:
    """Adds `target` to `[tool.uv.workspace]`'s own `exclude`, creating that key if the table has
    none yet. Verified against uv's own docs: every directory a `members` glob matches, and
    `exclude` does not, must hold a `pyproject.toml` — a plugin, scaffolded under the same
    `products/*` glob every Python product is, would otherwise break every `uv` command in the
    repository the moment it exists (found on the way, this step's own page)."""
    file = root / "pyproject.toml"
    text = file.read_text(encoding="utf-8")
    match = re.search(r"^\[tool\.uv\.workspace\]\n(.*?)(?=\n\[|\Z)", text, re.S | re.M)
    if match is None:
        sys.exit("no [tool.uv.workspace] table in pyproject.toml")
    body = match.group(1)
    entry = f'"{target}"'
    exclude = re.search(r"^exclude\s*=\s*\[(.*?)\]\s*$", body, re.S | re.M)
    if exclude:
        items = [item.strip() for item in exclude.group(1).split(",") if item.strip()]
        if entry in items:
            return
        items.append(entry)
        body = body[: exclude.start()] + f"exclude = [{', '.join(items)}]" + body[exclude.end() :]
    else:
        body = body.rstrip("\n") + f"\nexclude = [{entry}]\n"
    text = text[: match.start(1)] + body + text[match.end(1) :]
    file.write_text(text, encoding="utf-8")


if __name__ == "__main__":
    root, name, target = Path(sys.argv[1]), sys.argv[2], sys.argv[3]
    _marketplace(root, name)
    _excluded(root, target)
