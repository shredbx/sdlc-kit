"""Gathers the 15 process-os.command records (processos-workspace/records/command/<product>/*.yaml)
into one process-os.command-reference document, printed to standard output — this action's own
equivalent of create-package/assets/lib.sh's package_spec(): a template's own context comes from a
mapping's top-level keys (process_kit.template's own rule), so the 15 records need one field,
`commands`, to sit under before process-os.skill-reference can render from them.

Reads no third-party package — the same discipline build-command-docs/assets/build_docs.py already
uses, since a shell action's python3 is not uv's venv. Understands only process-os.command's own
flat, six-field shape, the same narrow parser (duplicated, not imported — each action's assets/ is
self-contained, the shape every other action here already follows); process-cli check and validate
still verify every record really is that shape."""

import re
import sys
from pathlib import Path

ORDER = [
    "init", "check", "list", "validate", "show", "write", "remove", "create", "edit", "run", "resume",
    "runs", "render", "conform", "mcp",
]
FIELDS = ["name", "group", "usage", "summary", "does", "prints"]
FIELD = re.compile(r"^([a-z]+):\s?(.*)$")


def parse_record(path: Path) -> dict[str, str]:
    """One `process-os.command` record's six fields, folded to one line each."""
    lines = path.read_text(encoding="utf-8").splitlines()
    fields: dict[str, str] = {}
    i = 0
    while i < len(lines):
        line = lines[i]
        if not line:
            i += 1
            continue
        match = FIELD.match(line)
        if not match:
            sys.exit(f"{path}: cannot read line {i + 1}: {line!r}")
        key, rest = match.group(1), match.group(2)
        if rest in (">-", "|-"):
            i += 1
            block: list[str] = []
            while i < len(lines) and (lines[i].startswith("  ") or lines[i] == ""):
                block.append(lines[i][2:])
                i += 1
            fields[key] = " ".join(part for part in block if part).strip()
            continue
        value = rest.strip()
        if len(value) >= 2 and value[0] == value[-1] == '"':
            value = value[1:-1]
        fields[key] = value
        i += 1
    return fields


def quote(value: str) -> str:
    """A YAML double-quoted scalar. Every value here is already one line (`parse_record` folds
    every block scalar), so only `\\` and `"` need escaping."""
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def main() -> None:
    product = sys.argv[1]
    folder = Path("processos-workspace/records/command") / product
    records = {path.stem: parse_record(path) for path in folder.glob("*.yaml")}
    missing = [name for name in ORDER if name not in records]
    if missing:
        sys.exit(f"build-skill-reference: no record for {missing} in {folder}")
    print("commands:")
    for name in ORDER:
        record = records[name]
        print(f"  - name: {quote(record['name'])}")
        for field in FIELDS[1:]:
            print(f"    {field}: {quote(record[field])}")


if __name__ == "__main__":
    main()
