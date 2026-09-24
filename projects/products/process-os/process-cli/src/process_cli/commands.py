"""One function for each command. Each is one operation of the framework, says what it found through the `Printer`, and gives the exit code."""

import argparse
import json
import os
import re
from pathlib import Path

from process_framework import KINDS, Error, ProcessFramework, Run, create_workspace

from .printer import Printer, shown


def init(printer: Printer, args: argparse.Namespace) -> int:
    """The three files a workspace starts with, in a folder. It needs no config, since it makes one."""
    paths, errors = create_workspace(args.folder, args.scope)
    printer.result([shown(Path(args.folder) / path) for path in paths], {"paths": paths}, errors)
    return 1 if errors else 0


def check(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Every definition of every scope. The config has loaded, since `initialize` got this far."""
    errors = framework.check()
    scopes = [
        {"name": name, "access": scope.access, "library": not scope.folder.is_relative_to(framework.catalog.root)}
        for name, scope in framework.catalog.scopes.items()
    ]
    printer.result([] if errors else [f"{shown(framework.config.file)}: ok"], {"config": str(framework.config.file), "scopes": scopes}, errors)
    return 1 if errors else 0


def list_(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """The ids of the definitions of one kind, or of every kind — or, for kind `records`, the paths of the records.
    `--long` also shows each definition's own one-line description; `--scope`/`--namespace`/`--limit` filter before
    anything past the bare id is loaded."""
    if args.kind == "records":
        paths = framework.list_records(args.prefix, args.limit)
        printer.result(paths, {"records": paths})
        return 0
    entries = [(kind, entry) for kind in ([args.kind] if args.kind else KINDS) for entry in framework.list(kind, args.scope, args.namespace, args.limit)]
    if args.long:
        described = [(kind, entry, framework.description(kind, entry.id)) for kind, entry in entries]
        lines = [f"{entry.id if args.kind else f'{kind} {entry.id}'}  {description or ''}".rstrip() for kind, entry, description in described]
        data = {"entries": [{"kind": kind, "id": entry.id, "file": str(entry.file), "description": description} for kind, entry, description in described]}
    else:
        lines = [entry.id if args.kind else f"{kind} {entry.id}" for kind, entry in entries]
        data = {"entries": [{"kind": kind, "id": entry.id, "file": str(entry.file)} for kind, entry in entries]}
    printer.result(lines, data)
    return 0


def validate(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """A data file against a type or a schema."""
    if not Path(args.file).is_file():
        printer.message(f"{shown(args.file)}: no such file")
        return 1
    errors = framework.validate(args.type, args.file)
    printer.result([] if errors else [f"{shown(args.file)}: ok"], {"file": os.path.abspath(args.file)}, errors)
    return 1 if errors else 0


def create(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """A new instance of a schema, from field values given by --set, written under a folder of the runtime's records.
    The exit code says whether it was written."""
    file, errors = framework.create(args.schema, dict(args.fields), args.into, args.name)
    printer.result([f"{shown(file)}: written"] if file else [], {"file": file}, errors)
    return 1 if errors else 0


def edit(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Merge field values given by --set into an existing record, checked against its schema whole before
    anything is written."""
    errors = framework.edit(args.schema, args.path, dict(args.fields))
    printer.result([] if errors else [f"{args.path}: edited"], {"path": args.path}, errors)
    return 1 if errors else 0


def show(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """The raw text of a definition: one string for a type, a schema or a process; one `# path` header
    per file, in order, for an action."""
    found = framework.show(args.kind, args.id)
    if isinstance(found, list):
        printer.errors(found)
        return 1
    if isinstance(found, dict):
        lines = [line for path, text in found.items() for line in (f"# {path}", text)]
        printer.result(lines, {"files": found})
        return 0
    printer.result([found], {"text": found})
    return 0


def write(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Create or replace a definition, checked before it is kept: a type, a schema or a process from FILE's
    own text; an action from every file under FOLDER, in the same relative-path shape `show` prints."""
    if args.kind == "action":
        if not Path(args.file).is_dir():
            printer.message(f"{shown(args.file)}: no such folder")
            return 1
        content = _read_folder(Path(args.file))
    else:
        if not Path(args.file).is_file():
            printer.message(f"{shown(args.file)}: no such file")
            return 1
        content = Path(args.file).read_text(encoding="utf-8")
    errors = framework.write(args.kind, args.id, content)
    printer.result([] if errors else [f"{args.kind} {args.id}: written"], {"kind": args.kind, "id": args.id}, errors)
    return 1 if errors else 0


def _read_folder(folder: Path) -> dict[str, str]:
    """Every file under `folder`, by its path from it, as text — `__pycache__` and `.DS_Store` skipped, the
    same as `write`'s own action shape does on the way back out through `show`."""
    found = {}
    for file in sorted(path for path in folder.rglob("*") if path.is_file()):
        relative = file.relative_to(folder)
        if "__pycache__" not in relative.parts and relative.name != ".DS_Store":
            found[relative.as_posix()] = file.read_text(encoding="utf-8")
    return found


def remove(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Delete a single-file definition: a type, a schema or a process."""
    errors = framework.remove(args.kind, args.id)
    printer.result([] if errors else [f"{args.kind} {args.id}: removed"], {"kind": args.kind, "id": args.id}, errors)
    return 1 if errors else 0


def render(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """The files a template makes from a data file, and, with `--into`, written under a folder of the output."""
    if not Path(args.data).is_file():
        printer.message(f"{shown(args.data)}: no such file")
        return 1
    paths, errors = framework.render(args.template, args.data, args.into)
    printer.result(paths, {"paths": paths}, errors)
    return 1 if errors else 0


def conform(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Whether the files in a folder of the output still hold what a template requires."""
    if not Path(args.data).is_file():
        printer.message(f"{shown(args.data)}: no such file")
        return 1
    errors = framework.conform(args.template, args.data, args.folder)
    printer.result([] if errors else [f"{args.folder}: ok"], {"folder": args.folder}, errors)
    return 1 if errors else 0


def run(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """One action or one process, on inputs given by name, each resolved by its own declared type. The exit code says whether it went through."""
    return _finish(printer, framework.run(args.id, dict(args.records)))


def resume(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Go on with a run that stopped or failed, from its first node that is not done."""
    return _finish(printer, framework.resume(args.run))


def _finish(printer: Printer, made: Run | list[Error]) -> int:
    """What became of a run: its summary, its outputs and errors, and how to go on when it did not end `done`; or the errors that kept it from starting."""
    if isinstance(made, list):
        printer.errors(made)
        return 1
    idle = made.status == "done" and bool(made.done) and all(node in made.skipped for node in made.done)
    reason = f" — {made.reason}" if made.reason else ""
    lines = [f"{made.target}: {'skipped' if idle else made.status}{reason}"]
    lines += [f"  {name}: {value if isinstance(value, str) else json.dumps(value)}" for name, value in made.output.items()]
    printer.result(lines, {"run": made.to_data(), "events": printer.events}, made.errors, ok=made.status == "done")
    if made.status == "done":
        return 0
    printer.note(f"run {made.id} is saved; go on with: process-cli resume {made.id}")
    return 1


def runs_(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Every run kept, most recent first, when `args.run` is not given; or one run's status, its nodes, and where its logs are."""
    if args.run is None:
        found = framework.list_runs()
        lines = [f"{run.id}: {run.target} {run.status} ({_when(run.id)})" for run in found]
        data = {"runs": [{"id": run.id, "process": run.target, "status": run.status, "when": _when(run.id)} for run in found]}
        printer.result(lines, data)
        return 0
    shown_run = framework.show_run(args.run)
    if isinstance(shown_run, list):
        printer.errors(shown_run)
        return 1
    logs = framework.runs.folder.path(shown_run.id)
    reason = f" — {shown_run.reason}" if shown_run.reason else ""
    lines = [f"{shown_run.target}: {shown_run.status}{reason}"]
    lines += [f"  {node}: {'skipped' if node in shown_run.skipped else 'done'}" for node in shown_run.done]
    lines += [f"logs: {shown(logs)}"]
    printer.result(lines, {"run": shown_run.to_data(), "logs": str(logs)})
    return 0


def mcp(framework: ProcessFramework, printer: Printer, args: argparse.Namespace) -> int:
    """Serve this workspace's tools and processes over MCP, on standard input and output, until the client
    disconnects. Prints nothing of its own — standard output is the protocol's own channel for as long as
    this runs, not this command's to print a result on. `serve` is imported here, not at the top of this
    file, so every other command's startup does not pay for importing the SDK."""
    from .mcp import serve

    serve(framework)
    return 0


def _when(run_id: str) -> str:
    """A run's start, read from its id (`Runs.new_id`'s own shape, `yymmdd-hhmmss-...`), shown as `yyyy-mm-dd hh:mm:ss`.
    `""` when the id does not have that shape."""
    match = re.match(r"(\d{2})(\d{2})(\d{2})-(\d{2})(\d{2})(\d{2})-", run_id)
    if not match:
        return ""
    yy, mm, dd, hh, mi, ss = match.groups()
    return f"20{yy}-{mm}-{dd} {hh}:{mi}:{ss}"
