"""Regenerates process-cli's own readme.yaml "commands:" list and "## The commands" section body
from the process-os.command records (processos-workspace/records/command/<product>/*.yaml) — task
08's "describe once" bridge. Nothing else in readme.yaml changes: this only replaces the text
between two anchors, so a comment or a hand-tuned block scalar elsewhere in the file survives byte
for byte. Reads no third-party package — plain stdlib, the same discipline
create-plugin/assets/register.py already uses, since a shell action's python3 is not uv's venv.

Only understands the flat, six-field shape process-os.command's own records are written in (a plain
or quoted one-line scalar, or a ">-"/"|-" block folded to one line) — not a general YAML reader. That
shape is task 08's own to keep; process-cli check and validate still verify every record is really
what this expects."""

import re
import sys
from pathlib import Path

ORDER = [
    "init", "check", "list", "validate", "show", "write", "remove", "create", "edit", "run", "resume",
    "runs", "render", "conform", "mcp",
]

# Global-flag examples: real, useful, and today's readme.yaml already carries them, but they are not
# commands of their own — process-os.command has no record for a flag. Carried over verbatim.
FLAG_EXAMPLES = [
    ("--config FILE check", "a config that is not processos.yaml"),
    ("--json run ID ...", "any command: the answer as one JSON document"),
    ("--version", "what it is; needs no config"),
]

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


def load_records(product: str) -> dict[str, dict[str, str]]:
    folder = Path("processos-workspace/records/command") / product
    records = {path.stem: parse_record(path) for path in folder.glob("*.yaml")}
    missing = [name for name in ORDER if name not in records]
    if missing:
        sys.exit(f"build-command-docs: no record for {missing} in {folder}")
    return records


def quick_reference(records: dict[str, dict[str, str]]) -> str:
    entries = [(f"process-cli {records[name]['usage']}", records[name]["summary"]) for name in ORDER]
    entries += [(f"process-cli {usage}", summary) for usage, summary in FLAG_EXAMPLES]
    width = max(len(line) for line, _ in entries) + 2
    return "\n".join(f'  - "{line.ljust(width)}# {summary}"' for line, summary in entries)


def commands_table(records: dict[str, dict[str, str]]) -> str:
    rows = ["| command | does | prints |", "|---|---|---|"]
    for name in ORDER:
        r = records[name]
        rows.append(f"| `{r['usage']}` | {r['does']} | {r['prints']} |")
    return "\n".join(rows)


def splice(text: str, records: dict[str, dict[str, str]]) -> str:
    quick = quick_reference(records)
    text, count = re.subn(r"(?m)^commands:\n(?:  - .*\n)+", lambda m: "commands:\n" + quick + "\n", text, count=1)
    if count != 1:
        sys.exit("build-command-docs: could not find the commands: list to replace")

    table = commands_table(records)
    indented = "\n".join((f"      {line}" if line else "") for line in table.split("\n"))
    pattern = re.compile(r'(  - heading: "## The commands"\n    body: \|-\n).*?(?=  - heading:|\Z)', re.S)
    text, count = pattern.subn(lambda m: m.group(1) + indented + "\n", text, count=1)
    if count != 1:
        sys.exit('build-command-docs: could not find the "## The commands" section to replace')
    return text


def main() -> None:
    product = sys.argv[1]
    records = load_records(product)
    readme = Path("processos-workspace/records/product") / product / "readme.yaml"
    readme.write_text(splice(readme.read_text(encoding="utf-8"), records), encoding="utf-8")


if __name__ == "__main__":
    main()
