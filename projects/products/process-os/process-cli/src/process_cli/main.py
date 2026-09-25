"""The command line: the arguments, where the config is, and the exit code. It imports the framework and nothing below it."""

import argparse
import os
import sys
from pathlib import Path

from process_framework import BUILD_FILE, CONFIG_FILE, FILE_KINDS, KINDS, find_config, initialize, read_build

from . import commands
from .printer import Printer, shown

ENVIRONMENT = "PROCESS_CLI_CONFIG"
PROGRAM_ENVIRONMENT = "PROCESS_CLI_PROGRAM"
COMMANDS = {
    "check": commands.check, "list": commands.list_, "validate": commands.validate, "run": commands.run,
    "create": commands.create, "resume": commands.resume, "render": commands.render, "conform": commands.conform,
    "runs": commands.runs_, "mcp": commands.mcp, "show": commands.show, "write": commands.write, "remove": commands.remove,
    "edit": commands.edit,
}


def main(argv: list[str] | None = None) -> int:
    """Run the command in `argv` (the process's own arguments by default) and give the exit code: 0 when everything
    is fine, 1 when errors were found, and 2 for a wrong call."""
    parser = _parser()
    try:
        args = parser.parse_args(argv)
        if args.version:
            print(_version())
            return 0
        if args.command is None:
            parser.error("a command is required")
        if args.command == "run":
            names = [name for name, _ in args.records]
            if len(names) != len(set(names)):
                parser.error("an input is given twice: --records names it once")
        if args.command == "mcp" and args.json:
            parser.error("--json is not accepted with mcp: it owns standard output for the protocol itself, not a JSON answer")
    except SystemExit as stop:
        return stop.code if isinstance(stop.code, int) else 2

    printer = Printer(json=args.json)
    printer.command = _identity()
    with printer.session():
        return _run(args, printer)


def _program() -> Path:
    """The file this program was started from, resolved: its real path, following any symlink to it."""
    return Path(sys.argv[0]).resolve()


def _identity() -> Path | None:
    """What this program is, for an action to call back as `process-cli`: `$PROCESS_CLI_PROGRAM`, resolved, when a
    launcher sets it (for when the program was not started as a file named `process-cli` — a renamed copy, or
    `python -m`); else the program itself, when it is named `process-cli`, as it is once built (M5) or installed
    (`uv tool install`); else `None`, so nothing that is not really this program is ever offered as a stand-in for
    it — the running program's own name alone, such as a test runner that happens to import and call this module,
    is not enough."""
    override = os.environ.get(PROGRAM_ENVIRONMENT)
    if override:
        return Path(override).resolve()
    program = _program()
    return program if program.name == "process-cli" else None


def _version() -> str:
    """`process-cli`, and where its version came from: the stamp `process-cli.build` beside the program, if there is
    one and it names a `version`, else the version this package was installed as."""
    build = read_build(_program().parent / BUILD_FILE)
    if build and "version" in build:
        parts = [f"built {build['date']}"] if build.get("date") else []
        parts += [f"commit {str(build['commit'])[:7]}"] if build.get("commit") else []
        extra = f" ({', '.join(parts)})" if parts else ""
        return f"process-cli {build['version']}{extra}"
    from importlib.metadata import PackageNotFoundError, version

    try:
        return f"process-cli {version('process-cli')} (source checkout)"
    except PackageNotFoundError:
        return "process-cli (version unknown)"


def _run(args: argparse.Namespace, printer: Printer) -> int:
    """The command in `args`, once the arguments are right. `init` needs no config; the others find theirs and start the framework."""
    if args.command == "init":
        return commands.init(printer, args)
    file = _config_file(args.config, printer)
    if file is None:
        return 1
    framework = initialize(file, printer)
    if isinstance(framework, list):
        printer.errors(framework)
        return 1
    return COMMANDS[args.command](framework, printer, args)


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="process-cli", description="Work on a workspace of YAML definitions.")
    parser.add_argument("--json", action="store_true", help="say the answer as one JSON document on standard output, and nothing else there")
    parser.add_argument("--config", metavar="FILE", help=f"the config file; else ${ENVIRONMENT}; else processos.yaml here or above")
    parser.add_argument("--version", action="store_true", help="print the version and exit; needs no config")
    commands_ = parser.add_subparsers(dest="command", metavar="command")
    commands_.add_parser("check", help="check every definition")
    one = commands_.add_parser("list", help="list the ids of the definitions, or, for kind records, the paths of the records")
    one.add_argument("kind", nargs="?", choices=(*KINDS, "records"), help="one kind, or records; else every kind")
    one.add_argument("prefix", nargs="?", help="for kind records only: a folder under the runtime's records; else every record")
    one.add_argument("--long", action="store_true", help="also show each definition's own one-line description")
    one.add_argument("--scope", metavar="NAME", help="only definitions in this one scope")
    one.add_argument("--namespace", metavar="NAME", help="only definitions with this namespace between the scope and the name")
    one.add_argument("--limit", type=int, metavar="N", help="at most this many")
    one = commands_.add_parser("validate", help="check a data file against a type or a schema")
    one.add_argument("type", help="the id of the type or schema")
    one.add_argument("file", help="the data file")
    one = commands_.add_parser("render", help="make the files of a template from a data file")
    one.add_argument("template", help="the id of the template")
    one.add_argument("data", help="the data file")
    one.add_argument("--into", metavar="FOLDER", help="write the files under this folder of the runtime's output; else only list them")
    one = commands_.add_parser("conform", help="check that the files in a folder still hold what a template requires")
    one.add_argument("template", help="the id of the template")
    one.add_argument("data", help="the data file")
    one.add_argument("folder", help="a folder of the runtime's output")
    one = commands_.add_parser("run", help="run an action or a process")
    one.add_argument("id", help="the id of the action or the process")
    one.add_argument(
        "--records", action="append", default=[], type=_records, metavar="NAME=TEXT",
        help="an input: an existing file at that exact path, else an id read as <id>/NAME.yaml under the runtime's records, else, "
        "for a string, an integer, a float or a boolean, the text itself",
    )
    one = commands_.add_parser("create", help="build a new instance of a schema from field values, and write it")
    one.add_argument("schema", help="the id of the schema")
    one.add_argument("--set", action="append", default=[], dest="fields", type=_records, metavar="NAME=VALUE", help="a field of the new instance, cast from text as its declared type")
    one.add_argument("--into", required=True, metavar="FOLDER", help="the folder under the runtime's records the instance is written into")
    one.add_argument("--name", metavar="NAME", help="the file's own name; else the schema's own last name — give it when a folder holds more than one instance of the same schema")
    one = commands_.add_parser("edit", help="merge field values into an existing record, checked against its schema whole")
    one.add_argument("schema", help="the id of the schema")
    one.add_argument("path", help="the record's own path under the runtime's records")
    one.add_argument("--set", action="append", default=[], dest="fields", type=_records, metavar="NAME=VALUE", help="a field to merge in, cast from text as its declared type")
    one = commands_.add_parser("init", help="write a config and an empty scope, to start a workspace")
    one.add_argument("folder", nargs="?", default=".", help="the folder to start it in; else the current folder")
    one.add_argument("--scope", default="main", metavar="NAME", help="the name of the scope; else main")
    one = commands_.add_parser("resume", help="go on with a run that stopped or failed")
    one.add_argument("run", help="the id of the run: a folder of the runtime's runs")
    one = commands_.add_parser("runs", help="list past runs, or show one")
    one.add_argument("run", nargs="?", help="the id of one run; else every run is listed")
    commands_.add_parser("mcp", help="serve this workspace's tools and processes over MCP, on standard input and output, until the client disconnects")
    one = commands_.add_parser("show", help="the raw text of a definition (by id) or a record (by path under the runtime's records)")
    one.add_argument("kind", choices=(*FILE_KINDS, "action", "record"), help="type, schema, process, action or record")
    one.add_argument("id", help="its id, or, for a record, its path")
    one = commands_.add_parser("write", help="create or replace a definition, checked before it is kept — records use create/edit instead")
    one.add_argument("kind", choices=(*FILE_KINDS, "action"), help="type, schema, process or action")
    one.add_argument("id", help="its id")
    one.add_argument("file", metavar="FILE", help="the new text; for an action, a local FOLDER of its files instead (action.yaml, its scripts, assets/)")
    one = commands_.add_parser("remove", help="delete a definition (by id) or a record (by path under the runtime's records)")
    one.add_argument("kind", choices=(*FILE_KINDS, "action", "record"), help="type, schema, process, action or record")
    one.add_argument("id", help="its id, or, for a record, its path")
    return parser


def _records(text: str) -> tuple[str, str]:
    name, equals, value = text.partition("=")
    if not (name and equals):
        raise argparse.ArgumentTypeError(f"{text!r} is not NAME=TEXT")
    return name, value


def _config_file(given: str | None, printer: Printer) -> Path | None:
    """`--config`, else the environment, else the first config found looking upward, else one beside the program
    (`_program()`'s folder — a config above the current folder still wins over it). `None` after saying why."""
    chosen = given or os.environ.get(ENVIRONMENT)
    if chosen:
        if not Path(chosen).is_file():
            printer.message(f"{shown(chosen)}: no such file")
            return None
        return Path(chosen)
    found = find_config(Path.cwd())
    if found is not None:
        return found
    beside = _program().parent / CONFIG_FILE
    if beside.is_file():
        return beside
    printer.message("no processos.yaml here or above, or beside the program; process-cli init makes one")
    return None
