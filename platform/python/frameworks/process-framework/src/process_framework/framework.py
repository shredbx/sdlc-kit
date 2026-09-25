"""The kit put together for one config."""

from __future__ import annotations

import builtins  # ProcessFramework.list shadows the builtin inside the class body, so later annotations say builtins.list
import os
import re
import shlex
import shutil
import tempfile
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml
from process_kit.action import Action, Context, Executor
from process_kit.config import Config
from process_kit.filesystem import Folder
from process_kit.process import Process, Run, Runner
from process_kit.process import check as check_graph
from process_kit.schema import Schema, parse
from process_kit.template import Template
from process_kit.types import Error, MappingType, Types

from .catalog import FILE_KINDS, KINDS, Catalog, Entry
from .host import Host
from .mcp import tool_of
from .runtime import Records, Runs, cast_value

#: An action's own bounded shape (task 09, M3): action.yaml and its four named scripts. Anything else it
#: may hold lives under assets/, checked by prefix, not listed here — the same two-part rule Action.load
#: itself doesn't need to state, since it only ever reads these five by name and everything else by glob.
ACTION_FILES = ("action.yaml", "check.sh", "pre.sh", "action.sh", "post.sh")

#: A name: lower case words joined by hyphens — the same rule action/name.yaml and every other name in
#: this system already states; used here for create's own optional --name (task 09, M4).
_NAME = re.compile(r"[a-z][a-z0-9-]*")


@dataclass(frozen=True)
class ProcessFramework:
    """What `initialize` gives back: the config, the catalog of definitions, the runtime, the executor and the host."""

    config: Config
    catalog: Catalog
    output: Folder
    records: Records
    runs: Runs
    executor: Executor
    host: Host

    def check(self) -> list[Error]:
        """Every problem in every scope of the definitions, each located in its file. See `Catalog.check`."""
        return self.catalog.check()

    def validate(self, type_id: str, file: str | Path) -> list[Error]:
        """The errors of the data in `file` against the type or schema `type_id`, each located in the file. A file that is
        not there is a wrong call and raises."""
        found = self.catalog.locate("type", type_id) or self.catalog.locate("schema", type_id)
        if found is None:
            return [Error((), "unknown_id", f"there is no type or schema {type_id!r}")]
        types = self.catalog.types([((), type_id)], found)
        if isinstance(types, list):
            return types
        file = Path(os.path.abspath(file))
        data, errors = parse(file.read_text(encoding="utf-8"))
        errors = errors or types.validate(data, type_id)
        return [Error((str(file), *error.path), error.code, error.message) for error in errors]

    def create(self, schema_id: str, values: Mapping[str, str], into: str, name: str | None = None) -> tuple[str, list[Error]]:
        """A new instance of the schema `schema_id`, cast from `values` field by field — the same casting `run`'s own literal
        values use — and checked against the schema whole. Written to `<into>/<name or the schema's own last name>.yaml` under
        `runtime.records` — `name`, when given, is how a folder holds more than one instance of the same schema, checked as
        any other name is: lower case words joined by hyphens. Refuses to write over one that is already there. `(file, errors)`:
        the file written, as text, or the errors, never both. `values` that is not a mapping is a wrong call and raises."""
        if not isinstance(values, Mapping):
            raise TypeError(f"values must be a mapping of field name to text, got {values!r}")
        if name is not None and not _NAME.fullmatch(name):
            return "", [Error(("--name",), "pattern", "a name is lower case words joined by hyphens")]
        found = self.catalog.locate("schema", schema_id)
        if found is None:
            return "", [Error((), "unknown_id", f"there is no schema {schema_id!r}")]
        types = self.catalog.types([((), schema_id)], found)
        if isinstance(types, list):
            return "", types
        data, errors = self._cast_record(schema_id, values, types)
        if errors:
            return "", errors
        if errors := types.validate(data, schema_id):
            return "", errors
        relative = f"{into}/{name or schema_id.rpartition('.')[2]}.yaml"
        if not self.records.folder.inside(relative):
            return "", [Error(("into",), "outside_root", f"{into!r} is not inside the records folder")]
        if self.records.folder.exists(relative):
            target = self.records.folder.path(relative)
            return "", [Error((str(target),), "exists", "a file is there already, and create writes over nothing")]
        self.records.folder.write(relative, yaml.safe_dump(data, sort_keys=False))
        return str(self.records.folder.path(relative)), []

    def edit(self, schema_id: str, path: str, values: Mapping[str, str]) -> list[Error]:
        """Merge `values` — cast field by field, the same casting `create` gives a new record — into the record kept at
        `path` under `runtime.records`, checked against `schema_id` whole before anything is written, so a bad merge
        never touches the file at all — no separate rollback needed, unlike `write`, which has to write a file before
        its own loader can check it. `path` not there is `unknown_id`. `values` that is not a mapping is a wrong call
        and raises."""
        if not isinstance(values, Mapping):
            raise TypeError(f"values must be a mapping of field name to text, got {values!r}")
        found = self.catalog.locate("schema", schema_id)
        if found is None:
            return [Error((), "unknown_id", f"there is no schema {schema_id!r}")]
        types = self.catalog.types([((), schema_id)], found)
        if isinstance(types, list):
            return types
        if not self.records.folder.inside(path):
            return [Error((), "outside_root", f"{path!r} is not inside the records folder")]
        previous = self.records.folder.read(path)
        if previous is None:
            return [Error((), "unknown_id", f"there is no record at {path!r}")]
        existing, errors = parse(previous)
        if errors:
            return [Error((path, *error.path), error.code, error.message) for error in errors]
        if not isinstance(existing, dict):
            return [Error((path,), "wrong_type", f"a record is a mapping, got {existing!r}")]
        merge, errors = self._cast_record(schema_id, values, types)
        if errors:
            return errors
        data = {**existing, **merge}
        if errors := types.validate(data, schema_id):
            return errors
        self.records.folder.write(path, yaml.safe_dump(data, sort_keys=False))
        return []

    def _cast_record(self, schema_id: str, values: Mapping[str, str], types: Types) -> tuple[dict[str, Any], list[Error]]:
        """`values`, each cast to its own field's declared type, or the errors: a name `schema_id` does not have as a
        field, or a value that fails its own field's type. `schema_id` is not a mapping schema with fields is
        `not_a_record` — `create` and `edit` share this, so records built either way are checked exactly alike."""
        schema = types.get(schema_id)
        base = schema.base if isinstance(schema, Schema) else schema
        fields = {field.name: field.type for field in getattr(base, "fields", None) or []}
        if not fields:
            return {}, [Error((), "not_a_record", f"{schema_id!r} is not a mapping schema with fields")]
        errors = [Error((name,), "unexpected", f"{name!r} is not a field of {schema_id!r}") for name in values if name not in fields]
        data: dict[str, Any] = {}
        for name, text in values.items():
            if name not in fields:
                continue
            cast, problems = cast_value(text, fields[name], types, (name,))
            errors.extend(problems)
            if not problems:
                data[name] = cast
        return data, errors

    def show(self, kind: str, id: str) -> str | dict[str, str] | list[Error]:
        """The raw text of `id`: for `type`, `schema` or `process`, a definition's own catalog id; for `action`,
        a catalog id too, given back as `{relative path: text}`; for `record`, not a catalog id at all but a
        path under `runtime.records` (with `.yaml`) — a record has never had one. `unknown_id` when there is
        none. `kind` not one of those five is a wrong call and raises: `template` is not part of this."""
        if kind == "record":
            if not self.records.folder.inside(id):
                return [Error((), "outside_root", f"{id!r} is not inside the records folder")]
            found_text = self.records.folder.read(id)
            if found_text is None:
                return [Error((), "unknown_id", f"there is no record at {id!r}")]
            return found_text
        if kind not in (*FILE_KINDS, "action"):
            raise ValueError(f"show reads type, schema, process, action or record here, got {kind!r}")
        found = self.catalog.locate(kind, id)
        if found is None:
            return [Error((), "unknown_id", f"there is no {kind} {id!r}")]
        return _walk_action(found) if kind == "action" else found.read_text(encoding="utf-8")

    def write(self, kind: str, id: str, content: str | Mapping[str, str]) -> list[Error]:
        """Create, or replace, the definition `id` of `kind` with `content`: text for `type`, `schema` or
        `process`, `{relative path: text}` for `action`'s own bounded shape (`action.yaml`, its four named
        scripts, and anything under `assets/`; another name is `unexpected`, refused before anything is
        written). Checked before it is kept: its own shape (the same rule `check()` uses for this kind), then
        everything it reaches (`catalog.types`, the same transitive check `validate`/`create` already give a
        record). A failure restores what was there before, or removes what was written if it was new — nothing
        is left half-written. Refused before anything is written: an id that is not scope-then-namespaces-then-
        name, a scope that does not exist, and a scope that is not `readwrite`. What this does not check: whether
        something *else* already in the catalog now points at a broken reference because of this write —
        `check()` is still the tool for the whole catalog. `kind` not one of the four is a wrong call and raises."""
        if kind not in (*FILE_KINDS, "action"):
            raise ValueError(f"write makes type, schema, process or action here, got {kind!r}")
        placed = self.catalog.place(kind, id)
        if isinstance(placed, list):
            return placed
        if kind == "action":
            return self._write_action(id, placed, content)
        if not isinstance(content, str):
            raise TypeError(f"a {kind}'s content is text, got {content!r}")
        previous = placed.read_text(encoding="utf-8") if placed.is_file() else None
        placed.parent.mkdir(parents=True, exist_ok=True)
        placed.write_text(content, encoding="utf-8")
        errors = self._verify(kind, id, placed)
        if errors:
            if previous is None:
                placed.unlink()
            else:
                placed.write_text(previous, encoding="utf-8")
        return errors

    def remove(self, kind: str, id: str) -> list[Error]:
        """Delete `id`: a file for `type`, `schema` or `process`, a whole folder for `action`, a file under
        `runtime.records` for `record` (a path, not a catalog id — same as `show`). Refuses an unknown id, and,
        for the four catalog kinds, a readonly scope; beyond that, deletes unconditionally — nothing here checks
        whether another definition still refers to `id`, since no reverse-reference index exists yet (research.md
        decision 104's own named limit, task 09). `check()` afterward is how a caller who wants to know finds
        out. `kind` not one of the five is a wrong call and raises."""
        if kind == "record":
            if not self.records.folder.inside(id):
                return [Error((), "outside_root", f"{id!r} is not inside the records folder")]
            if not self.records.folder.path(id).is_file():
                return [Error((), "unknown_id", f"there is no record at {id!r}")]
            self.records.folder.remove(id)
            return []
        if kind not in (*FILE_KINDS, "action"):
            raise ValueError(f"remove takes type, schema, process, action or record here, got {kind!r}")
        placed = self.catalog.place(kind, id)
        if isinstance(placed, list):
            return placed
        exists = placed.is_dir() if kind == "action" else placed.is_file()
        if not exists:
            return [Error((), "unknown_id", f"there is no {kind} {id!r}")]
        if kind == "action":
            shutil.rmtree(placed)
        else:
            placed.unlink()
        return []

    def _write_action(self, id: str, folder: Path, files: str | Mapping[str, str]) -> list[Error]:
        """`write`'s own `action` case: replace the whole folder's tracked content with `files` — the caller
        supplies everything they want there, not a patch. Backs up what the folder held (small; kept in memory,
        not on disk), removes it, writes the new files, verifies, and on failure removes what was written and
        restores exactly what was backed up."""
        if not isinstance(files, Mapping):
            raise TypeError(f"an action's content is a mapping of relative path to text, got {files!r}")
        bad = sorted(name for name in files if name not in ACTION_FILES and not name.startswith("assets/"))
        if bad:
            return [Error((name,), "unexpected", f"{name!r} is not action.yaml, one of its scripts, or under assets/") for name in bad]
        previous = _walk_action(folder) if folder.is_dir() else None
        if folder.is_dir():
            shutil.rmtree(folder)
        for relative, text in files.items():
            target = folder / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(text, encoding="utf-8")
        errors = self._verify_action(id, folder)
        if errors:
            shutil.rmtree(folder, ignore_errors=True)
            if previous is not None:
                for relative, text in previous.items():
                    target = folder / relative
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_text(text, encoding="utf-8")
        return errors

    def _verify_action(self, id: str, folder: Path) -> list[Error]:
        """Whatever `write` just put at `folder` really is a valid action, named `id`, and everything it reaches
        resolves — `catalog.node` and `catalog.types_of`, the same load `run` gives one before executing it."""
        node = self.catalog.node(id)
        if isinstance(node, list):
            return node
        if not isinstance(node, Action):
            return [Error((str(folder),), "invalid", f"{id!r} is a process, not an action")]
        types = self.catalog.types_of(node)
        return types if isinstance(types, list) else []

    def _verify(self, kind: str, id: str, path: Path) -> list[Error]:
        """Whatever `write` just put at `path` really is a valid `kind`, named `id`, and everything it reaches
        resolves. A process also gets its own graph checked, the same way `run` checks one before starting."""
        if kind == "process":
            node = self.catalog.node(id)
            if isinstance(node, list):
                return node
            if not isinstance(node, Process):
                return [Error((str(path),), "invalid", f"{id!r} is an action, not a process")]
            types = self.catalog.types_of(node)
            if isinstance(types, list):
                return types
            return check_graph(node, self.catalog, types)
        schema = Schema.load(path, id.rpartition(".")[0])
        if isinstance(schema, list):
            return [Error((str(path), *error.path), error.code, error.message) for error in schema]
        errors = []
        if schema.name != id:
            errors.append(Error((str(path), "name"), "invalid", f"name must be {id.rpartition('.')[2]!r}, the file's own name"))
        if kind == "type" and isinstance(schema.base, MappingType):
            errors.append(Error((str(path), "type"), "invalid", "a mapping belongs in schema/: type/ holds rules for one value"))
        if kind == "schema" and not isinstance(schema.base, MappingType):
            errors.append(Error((str(path), "type"), "invalid", "a schema is a mapping: a rule for one value belongs in type/"))
        if errors:
            return errors
        types = self.catalog.types(schema.references(), path)
        return types if isinstance(types, list) else []

    def render(self, template_id: str, file: str | Path, into: str | None = None) -> tuple[list[str], list[Error]]:
        """`(paths, errors)`: the files the template makes from the data in `file`, by path, and nothing else, or the errors.
        With `into`, a folder inside `runtime.output`, they are written there, after everything that can go wrong has been
        checked. Without it nothing is written, and the paths say what would be. A `file` that is not there is a wrong call
        and raises."""
        template, types, data, errors = self._template_and_data(template_id, file)
        if errors:
            return [], errors
        files = template.render(data, types)
        if isinstance(files, list):
            return [], files
        if into is not None:
            if not self.output.inside(into):
                return [], [Error((into,), "outside_root", f"{into!r} is not inside the output folder")]
            target = self.output.folder(into)
            for path, content in files.items():
                target.write(path, content)
        return list(files), []

    def conform(self, template_id: str, file: str | Path, folder: str) -> list[Error]:
        """What is wrong with the files in `folder`, inside `runtime.output`, against the template and the data in `file`, each
        located at the file it is about. `[]` when they conform. A `file` that is not there is a wrong call and raises."""
        template, types, data, errors = self._template_and_data(template_id, file)
        if errors:
            return errors
        if not self.output.inside(folder):
            return [Error((folder,), "outside_root", f"{folder!r} is not inside the output folder")]
        source = self.output.folder(folder)
        return [
            error if os.path.isabs(error.path[0]) else Error((str(source.path(error.path[0])), *error.path[1:]), error.code, error.message)
            for error in template.conform(data, source, types)
        ]

    def _template_and_data(self, template_id: str, file: str | Path) -> tuple[Any, Any, Any, list[Error]]:
        """The template, its types and the data in `file` checked against its input, or the errors, located in the file."""
        template = self.catalog.template(template_id)
        if isinstance(template, list):
            return None, None, None, template
        types = self.catalog.types(template.references(), template.home / "template.yaml")
        if isinstance(types, list):
            return None, None, None, types
        file = Path(os.path.abspath(file))
        data, errors = parse(file.read_text(encoding="utf-8"))
        errors = errors or types.validate(data, template.input)
        return template, types, data, [Error((str(file), *error.path), error.code, error.message) for error in errors]

    def run(self, id: str, records: Mapping[str, str], typed: Mapping[str, Any] | None = None) -> Run | list[Error]:
        """Run the process or the action `id`. `records` maps each input's name to text, read by its declared type — a scalar as a literal
        value, anything else as a file at that path or, failing that, an id under `runtime.records` (`Records.gather`'s own rule). `typed`
        maps an input's name to an already-typed value instead, checked as it is, never cast — `None` by default, meaning none are given
        that way. The errors that keep it from starting are returned, and nothing is saved: an id that is not one, definitions with errors,
        a graph that does not hold, and inputs that are not given, not there or not of their type. Once it starts, its `Run` is returned,
        saved after every node, whatever came of it. An action is run as a process of one step. `records` or `typed` that is not a mapping
        is a wrong call and raises."""
        if not isinstance(records, Mapping):
            raise TypeError(f"records must be a mapping of input name to text, got {records!r}")
        node = self.catalog.node(id)
        if isinstance(node, list):
            return node
        types = self.catalog.types_of(node)
        if isinstance(types, list):
            return types
        if isinstance(node, Process) and (errors := check_graph(node, self.catalog, types)):
            return errors
        process = node if isinstance(node, Process) else Process.of_action(node)
        gathered, errors = self.records.gather(process.input, process.requires, records, {} if typed is None else typed, types)
        if errors:
            return errors
        scopes = {name: scope.version for name, scope in self.catalog.scopes.items()}
        digest = self.catalog.digest(node)
        return self._walk(node, types, lambda runner: runner.start(process, gathered, scopes, digest))

    def resume(self, run_id: str) -> Run | list[Error]:
        """Go on with the run `run_id` from its first node that is not done, and return it. It is refused, with the errors returned and
        nothing changed, when there is no such run, when its file is not a run, when what it ran is gone or wrong, and when a definition it
        depends on has changed since it started. A run that is done is returned as it is. A `run_id` that is not text raises."""
        if not isinstance(run_id, str):
            raise TypeError(f"a run id is text, got {run_id!r}")
        run = self._load_run(run_id)
        if isinstance(run, list):
            return run
        node = self.catalog.node(run.target)
        if isinstance(node, list):
            return node
        types = self.catalog.types_of(node)
        if isinstance(types, list):
            return types
        if self.catalog.digest(node) != run.digest:
            return [Error((), "changed", f"the definitions of {run.target!r} have changed since the run {run_id!r} started, so it cannot go on")]
        process = node if isinstance(node, Process) else Process.of_action(node)
        return self._walk(node, types, lambda runner: runner.resume(run, process))

    def list_runs(self) -> list[Run]:
        """Every run that is kept, most recent first (`Runs.list`'s order). One whose file is not a run is left out, quietly —
        `show_run` is where that is an error."""
        found = []
        for run_id in self.runs.list():
            try:
                run = self.runs.load(run_id)
            except ValueError:
                continue
            if run is not None:
                found.append(run)
        return found

    def show_run(self, run_id: str) -> Run | list[Error]:
        """The run kept under `run_id`, exactly as `resume` would load it, without going on with it — so the same two errors
        (`unknown_id`, `invalid`) for the same two problems. A `run_id` that is not text raises."""
        if not isinstance(run_id, str):
            raise TypeError(f"a run id is text, got {run_id!r}")
        return self._load_run(run_id)

    def _load_run(self, run_id: str) -> Run | list[Error]:
        """The run kept under `run_id`, or the errors that say why there is none: `invalid`, located at its file, when it is
        there but is not a run; `unknown_id` when there is none, or when `run_id` leaves the runs folder."""
        try:
            run = self.runs.load(run_id)
        except ValueError as reason:
            return [Error((str(self.runs.folder.path(f"{run_id}/run.yaml")),), "invalid", str(reason))]
        if run is None:
            return [Error((), "unknown_id", f"there is no run {run_id!r}")]
        return run

    def _walk(self, node: Action | Process, types: Types, go: Callable[[Runner], Run]) -> Run:
        """Make the output folder, build the runner for `node`, and let `go` start or resume with it. A process tells the host each node it
        walks, and an action tells it nothing. A run that ends `done` prunes the done runs down to `config.runtime.history`, the most
        recent kept — a `stopped` or `failed` run is never pruned, since `resume` and a person still need it."""
        self.output.make()
        with tempfile.TemporaryDirectory() as scratch:
            context = Context(self.output.root, self._environment(Path(scratch)))
            run = go(Runner(self.catalog, types, self.executor, self.runs, context, self.host.report if isinstance(node, Process) else None))
        if run.status == "done":
            self.runs.prune(self.config.runtime.history)
        return run

    def _environment(self, scratch: Path) -> dict[str, str]:
        """What an action is given besides its inputs: the config, and, when the host has a `command`, a `process-cli` on its
        `PATH` that runs it, so an action can call the program that is running it."""
        environment = {"PROCESS_CLI_CONFIG": str(self.config.file)}
        if self.host.command is not None:
            shim = scratch / "process-cli"
            shim.write_text(f'#!/usr/bin/env bash\nexec {shlex.quote(str(self.host.command))} "$@"\n', encoding="utf-8")
            shim.chmod(0o755)
            environment["PATH"] = f"{scratch}{os.pathsep}{os.environ.get('PATH', '')}"
        return environment

    def list(self, kind: str, scope: str | None = None, namespace: str | None = None, limit: int | None = None) -> list[Entry]:
        """Every definition of `kind`, by id — filtered *before* anything is loaded, the same cheap id-prefix compare
        `entries()` already makes free: `scope`, when given, only ids in that one scope; `namespace`, when given, only
        ids with that segment somewhere between the scope and the name; `limit`, when given, the first that many, in
        `entries()`'s own order. A `kind` that is not one of the five is a wrong call and raises."""
        found = self.catalog.entries(kind)
        if scope is not None:
            found = [entry for entry in found if entry.id.split(".")[0] == scope]
        if namespace is not None:
            found = [entry for entry in found if namespace in entry.id.split(".")[1:-1]]
        return found[:limit] if limit is not None else found

    def description(self, kind: str, id: str) -> str | None:
        """The one-line `description` of the definition `id` of `kind`, or `None`: when it has none, or when it
        cannot be loaded at all — `check()` is where a broken definition is reported, not this, so `list --long`
        can show every id it always did, description or not. A `kind` that is not one of the five is a wrong
        call and raises."""
        if kind not in KINDS:
            raise ValueError(f"description reads a kind here: {', '.join(KINDS)}, got {kind!r}")
        if kind in ("process", "action"):
            node = self.catalog.node(id)
            return node.description if isinstance(node, (Action, Process)) else None
        if kind == "template":
            template = self.catalog.template(id)
            return template.description if isinstance(template, Template) else None
        found = self.catalog.locate(kind, id)
        if found is None:
            return None
        schema = Schema.load(found, id.rpartition(".")[0])
        return schema.description if isinstance(schema, Schema) else None

    def list_records(self, prefix: str | None = None, limit: int | None = None) -> builtins.list[str]:
        """Every record's own path under `runtime.records`, sorted, each ending `.yaml`. `prefix`, when given,
        only those under that folder of it — one outside the records folder, or that is not a folder there,
        gives none, quietly, the same as `entries()` does for a kind with nothing in it. `limit`, when given,
        the first that many."""
        root = self.records.folder.root
        base = root
        if prefix is not None:
            if not self.records.folder.inside(prefix):
                return []
            base = self.records.folder.path(prefix)
        if not base.is_dir():
            return []
        found = sorted(path.relative_to(root).as_posix() for path in base.rglob("*.yaml") if path.is_file())
        return found[:limit] if limit is not None else found

    def tools(self) -> builtins.list[dict[str, Any]]:
        """Every action and process, in every scope, as one MCP tool descriptor each (`mcp.tool_of`), sorted by
        name. A definition with an error is left out, quietly — `check()` is where that is reported."""
        found = []
        for kind in ("action", "process"):
            for entry in self.catalog.entries(kind):
                node = self.catalog.node(entry.id)
                if isinstance(node, list):
                    continue
                types = self.catalog.types_of(node)
                if isinstance(types, list):
                    continue
                found.append(tool_of(node, types))
        return sorted(found, key=lambda tool: tool["name"])

    def call(self, name: str, arguments: Mapping[str, Any]) -> Run | builtins.list[Error]:
        """Serve one MCP `tools/call`: `run(name, {}, typed=arguments)` — M3's typed path, reused, not
        reinvented. A stopped or failed run is still returned, not raised; only a bad call (an unknown id,
        wrong input) gives the list of errors this can also give."""
        return self.run(name, {}, arguments)


def _walk_action(folder: Path) -> dict[str, str]:
    """Every file under an action's own `folder`, by its path from it, as text — `process_kit.template`'s own
    `_files` walk, minus the `.jinja` handling an action has no use for. `__pycache__` and `.DS_Store` are
    skipped, the same as there."""
    found = {}
    for file in sorted(path for path in folder.rglob("*") if path.is_file()):
        relative = file.relative_to(folder)
        if "__pycache__" not in relative.parts and relative.name != ".DS_Store":
            found[relative.as_posix()] = file.read_text(encoding="utf-8")
    return found
