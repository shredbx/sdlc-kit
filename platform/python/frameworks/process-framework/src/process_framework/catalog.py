"""The definitions folder: its scopes, where an id leads, and which ids a kind has. The layout lives here and only here."""

import hashlib
import os
import re
from collections import deque
from collections.abc import Iterable, Iterator, Mapping
from dataclasses import dataclass
from functools import cache
from importlib.resources import files
from pathlib import Path
from types import MappingProxyType

from process_kit.action import Action
from process_kit.config import Library
from process_kit.process import Call, Process, Step, Switch
from process_kit.process import check as check_graph
from process_kit.schema import Schema, parse
from process_kit.template import Template
from process_kit.types import YAML_TYPES, Error, Location, MappingType, Types

KINDS = ("type", "schema", "template", "action", "process")
FILE_KINDS = ("type", "schema", "process")  # a `<name>.yaml`; the others are a folder with a `<kind>.yaml`
_NAME = re.compile(r"[a-z][a-z0-9-]*")
_SCOPE = "process_framework.scope"
_SCOPE_FILE = "scope.yaml"


@dataclass(frozen=True)
class Scope:
    """One folder of the definitions folder, read from its `scope.yaml`, or a library's folder. `access` is
    `readwrite` for a scope of the default folder, and what the library said, `readonly` or `readwrite`, for one
    read from a library."""

    name: str
    version: str
    description: str | None
    uses: tuple[str, ...]
    folder: Path
    access: str


@dataclass(frozen=True)
class Entry:
    """One definition a kind has: its id, and the file that holds it."""

    id: str
    kind: str
    file: Path


@dataclass(frozen=True)
class Catalog:
    """The definitions folder, opened. It holds the scopes and knows the layout, and reads no definition."""

    root: Path
    scopes: Mapping[str, Scope]

    @classmethod
    def open(cls, root: str | Path, libraries: Iterable[Library] = (), config_file: str | Path | None = None) -> "Catalog | list[Error]":
        """The catalog of the definitions folder `root`, plus one scope for each of `libraries`, or every error, never
        both. A `root` that is not there, or is a file, is a wrong call and raises. Every folder in `root` not starting
        with a dot is a scope, read `readwrite`. Each library is read from its own `path`, `readonly` unless it says
        `readwrite`, and checked: its `path` must be a folder, its `name` must be free (not another library's, not a
        scope of `root`) and must be what its `scope.yaml` says, and its `version`, if given, must be what `scope.yaml`
        says too. Every error a library gives is located at `config_file`, except one in its own `scope.yaml`, which is
        located there instead. `config_file` only locates errors: it is not read."""
        root = Path(os.path.abspath(root))
        if not root.exists():
            raise FileNotFoundError(f"{root} does not exist")
        if not root.is_dir():
            raise NotADirectoryError(f"{root} is not a folder")
        scopes: dict[str, Scope] = {}
        errors: list[Error] = []
        for folder in sorted(root.iterdir()):
            if folder.is_dir() and not folder.name.startswith("."):
                found = _read(folder, folder.name, "readwrite")
                if isinstance(found, Scope):
                    scopes[found.name] = found
                else:
                    errors.extend(found)
        here = (str(config_file),)
        for index, library in enumerate(libraries):
            where = (*here, "libraries", index)
            if library.name in scopes:
                errors.append(Error((*where, "name"), "duplicate", f"{library.name!r} is already a scope"))
                continue
            if not library.path.is_dir():
                errors.append(Error((*where, "path"), "not_found", f"there is no folder {library.path}"))
                continue
            found = _read(library.path, library.name, library.access)
            if isinstance(found, list):
                errors.extend(found)
                continue
            if library.version is not None and library.version != found.version:
                errors.append(Error((*where, "version"), "version_mismatch", f"{library.path} is version {found.version!r}, not {library.version!r}"))
                continue
            scopes[found.name] = found
        errors = errors or _check_uses(scopes)
        return errors if errors else cls(root, MappingProxyType(scopes))

    def locate(self, kind: str, id: str) -> Path | None:
        """Where the definition `id` of `kind` is, or None. A file for a type, schema or process, a folder for an action
        or a template. Nothing is read. A wrong `kind`, or a `kind` or `id` that is not text, is a wrong call and raises."""
        _check_kind(kind)
        if not isinstance(id, str):
            raise TypeError(f"an id must be text, got {id!r}")
        parts = id.split(".")
        if len(parts) < 2 or not all(_NAME.fullmatch(part) for part in parts):
            return None
        scope, *namespaces, name = parts
        if scope not in self.scopes:
            return None
        folder = self.scopes[scope].folder.joinpath(*namespaces, kind)
        if kind in FILE_KINDS:
            path = folder / f"{name}.yaml"
            return path if path.is_file() else None
        return folder / name if (folder / name).is_dir() else None

    def place(self, kind: str, id: str) -> Path | list[Error]:
        """Where `id` of `kind` would be written, whether or not anything is there yet — `locate`'s own path
        computation, without the existence check, refusing what `locate` has no way to say: an id that is not
        scope, then namespaces, then a name, a scope that does not exist, and a scope that is not `readwrite`
        (a library's own `access`, enforced for the first time by anything other than convention). A wrong
        `kind` is a wrong call and raises, same as `locate`."""
        _check_kind(kind)
        if not isinstance(id, str):
            raise TypeError(f"an id must be text, got {id!r}")
        parts = id.split(".")
        if len(parts) < 2 or not all(_NAME.fullmatch(part) for part in parts):
            return [Error((), "invalid", f"{id!r} is not scope, then namespaces, then a name, each lower case words joined by hyphens")]
        scope_name, *namespaces, name = parts
        scope = self.scopes.get(scope_name)
        if scope is None:
            return [Error((), "unknown_scope", f"there is no scope named {scope_name!r}")]
        if scope.access != "readwrite":
            return [Error((), "readonly", f"scope {scope_name!r} is readonly")]
        folder = scope.folder.joinpath(*namespaces, kind)
        return folder / f"{name}.yaml" if kind in FILE_KINDS else folder / name

    def entries(self, kind: str) -> list[Entry]:
        """Every definition of `kind` in every scope, sorted by id. No file is read, so a broken definition is listed,
        and what is not a name in a kind folder is not."""
        _check_kind(kind)
        found = [entry for scope in self.scopes.values() for entry in _walk(scope.folder, (scope.name,), kind)]
        return sorted(found, key=lambda entry: entry.id)

    def node(self, id: str) -> Action | Process | list[Error]:
        """The action or the process with that id, or the errors: `unknown_id` when there is neither, and `duplicate` when there
        are both, since one id names one definition. What their types are is `types`'s business. An `id` that is not text is a
        wrong call and raises."""
        folder, file = self.locate("action", id), self.locate("process", id)
        if folder is None:
            if file is None:
                return [Error((), "unknown_id", f"there is no action or process with the id {id!r}")]
            return self._process(id, file)
        if file is not None:
            return [Error((), "duplicate", f"{id!r} is an action and a process; one id names one definition")]
        return self._action(id, folder)

    def _action(self, id: str, folder: Path) -> Action | list[Error]:
        file = folder / "action.yaml"
        if not file.is_file():
            return [Error((str(file),), "missing", "an action has an action.yaml")]
        return Action.load(folder, id.rpartition(".")[0])

    def _process(self, id: str, file: Path) -> Process | list[Error]:
        return Process.load(file, id.rpartition(".")[0])

    def types_of(self, node: Action | Process) -> Types | list[Error]:
        """The types and schemas that `node`, and every node its steps call, at any depth, reach: their ports, and what those name.
        A step that names a node that does not exist is passed over, since the graph check reports it. Errors are located in the
        file of the node they are about."""
        builder = _Builder(self)
        for item in self._reached(node):
            file = item.home if isinstance(item, Process) else item.home / "action.yaml"
            builder.refer(file, item.id.split(".")[0], item.references())
        builder.run()
        return builder.errors if builder.errors else builder.known

    def digest(self, node: Action | Process) -> str:
        """A hash of what a run of `node` depends on: the file of every process reached, every file of every action reached, and the file of
        every type and schema they reach. The same files give the same digest, in any catalog, whether a scope sits in the default folder
        or in a library: its key is its own scope's name and its path relative to that scope's folder, never `self.root`. It raises
        `ValueError` if `types_of` has errors."""
        types = self.types_of(node)
        if isinstance(types, list):
            raise ValueError(f"the definitions of {node.id!r} have errors: {types}")
        files: list[Path] = []
        for item in self._reached(node):
            if isinstance(item, Process):
                files.append(item.home)
            else:
                files.extend(sorted(path for path in item.home.rglob("*") if path.is_file() and "__pycache__" not in path.parts))
        for name in sorted(name for name in types.known if name not in YAML_TYPES):
            located = self.locate("type", name) or self.locate("schema", name)
            assert located is not None  # `name` is one of `types.known`, so a type or a schema file is there
            files.append(located)
        found = hashlib.sha256()
        for file in files:
            found.update(self._key(file).encode("utf-8") + b"\0" + file.read_bytes() + b"\0")
        return found.hexdigest()

    def _key(self, file: Path) -> str:
        """`file`'s digest key: the name of the scope it is in, and its path relative to that scope's own folder — the same
        wherever the scope's folder sits, in the default folder or a library."""
        scope = self._containing(file)
        return f"{scope.name}/{file.relative_to(scope.folder)}"

    def _containing(self, file: Path) -> Scope:
        """The scope whose folder holds `file`. Raises `ValueError` when no scope of this catalog does."""
        for scope in self.scopes.values():
            if file.is_relative_to(scope.folder):
                return scope
        raise ValueError(f"{file} is not inside a scope of this catalog")

    def _reached(self, node: Action | Process) -> list[Action | Process]:
        """`node` and every node its steps call, at any depth, each once, in the order they are met. A node that does not exist is passed over."""
        if not isinstance(node, (Action, Process)):
            raise TypeError(f"a node is an Action or a Process, got {node!r}")
        found: dict[str, Action | Process] = {}
        pending = [node]
        while pending:
            item = pending.pop(0)
            if item.id in found:
                continue
            found[item.id] = item
            if isinstance(item, Process):
                pending.extend(child for call in _calls(item.steps) if not isinstance(child := self.node(call.id), list))
        return list(found.values())

    def template(self, id: str) -> Template | list[Error]:
        """The template with that id, or the errors. What its data's type is, is `types`'s business. An `id` that is not text
        is a wrong call and raises."""
        folder = self.locate("template", id)
        if folder is None:
            return [Error((), "unknown_id", f"there is no template with the id {id!r}")]
        file = folder / "template.yaml"
        if not file.is_file():
            return [Error((str(file),), "missing", "a template has a template.yaml")]
        return Template.load(folder, id.rpartition(".")[0])

    def types(self, references: Iterable[tuple[Location, str]], file: str | Path) -> Types | list[Error]:
        """The types and schemas that `references` reach, loaded and checked, or the errors. `references` are
        `(where, name)` pairs written in `file`, which is inside a scope of this catalog, or the call raises. Each name
        is in full. A definition loads once, and the types it names follow. Errors are located in the file that holds
        the reference, or in the definition found."""
        file = Path(os.path.abspath(file))
        builder = _Builder(self)
        builder.refer(file, self._scope_of(file), references)
        builder.run()
        return builder.errors if builder.errors else builder.known

    def check(self) -> list[Error]:
        """Every problem in every scope, each located in its file: the folder rules, every type and schema, every action, every template,
        and every name they use, in that order (`steps/m3-6-catalog-check.md`, and templates after actions: `m4-3`). Processes are not read yet."""
        errors = [error for scope in self.scopes.values() for error in _layout(scope.folder, True)]
        builder = _Builder(self)
        for name in sorted({entry.id for kind in ("type", "schema") for entry in self.entries(kind)}):
            builder.define(name)
        definitions, loading = len(builder.errors), []
        for entry in self.entries("action"):
            action = self._action(entry.id, entry.file.parent)
            if isinstance(action, list):
                loading.extend(action)
            else:
                builder.refer(action.home / "action.yaml", entry.id.split(".")[0], action.references())
        for entry in self.entries("template"):
            template = self.template(entry.id)
            if isinstance(template, list):
                loading.extend(template)
            else:
                builder.refer(template.home / "template.yaml", entry.id.split(".")[0], template.references())
        processes = []
        for entry in self.entries("process"):
            process = self._process(entry.id, entry.file)
            if isinstance(process, list):
                loading.extend(process)
            else:
                processes.append(process)
                builder.refer(process.home, entry.id.split(".")[0], process.references())
        builder.run()
        graph = []
        checked: set[str] = set()
        for process in processes:
            graph.extend(check_graph(process, self, builder.known, checked))
        by_id = {entry.id: entry for entry in self.entries("process")}
        duplicates = [
            Error((str(by_id[entry.id].file), "name"), "duplicate", f"{entry.id!r} is also an action; one id names one definition")
            for entry in self.entries("action")
            if entry.id in by_id
        ]
        return errors + builder.errors[:definitions] + loading + builder.errors[definitions:] + graph + duplicates

    def _scope_of(self, file: Path) -> str:
        return self._containing(file).name


class _Builder:
    """Loads definitions and follows the references they hold, each once, and collects every error."""

    def __init__(self, catalog: Catalog) -> None:
        self.catalog = catalog
        self.known = Types()
        self.errors: list[Error] = []
        self._loaded: set[str] = set()
        self._queue: deque[tuple[Path, str, list[tuple[Location, str]]]] = deque()

    def refer(self, file: Path, scope: str, references: Iterable[tuple[Location, str]]) -> None:
        """Queue the references written in `file`, a file in `scope`."""
        self._queue.append((file, scope, list(references)))

    def define(self, name: str) -> None:
        """Load the definition of `name`, which is there, unless it is loaded, and queue the references it holds."""
        if name in self._loaded:
            return
        self._loaded.add(name)
        found = [(kind, path) for kind in ("type", "schema") if (path := self.catalog.locate(kind, name))]
        references = _load(name, found, self.known, self.errors)
        if references is not None:
            self._queue.append((found[0][1], name.split(".")[0], references))

    def run(self) -> None:
        """Take the queue in order: each name must lead somewhere, and to a scope the referring file may name."""
        catalog = self.catalog
        while self._queue:
            here, scope, pending = self._queue.popleft()
            for where, name in pending:
                place = (str(here), *where)
                parts = name.split(".")
                if name in YAML_TYPES:
                    continue
                if len(parts) < 2 or not all(_NAME.fullmatch(part) for part in parts) or parts[0] not in catalog.scopes:
                    self.errors.append(Error(place, "unknown_id", f"there is no type or schema {name!r}"))
                elif parts[0] != scope and parts[0] not in catalog.scopes[scope].uses:
                    self.errors.append(Error(place, "not_used", f"scope {scope!r} does not use scope {parts[0]!r}: add it to uses in its scope.yaml"))
                elif not (catalog.locate("type", name) or catalog.locate("schema", name)):
                    self.errors.append(Error(place, "unknown_id", f"there is no type or schema {name!r}"))
                else:
                    self.define(name)


def _layout(folder: Path, top: bool) -> Iterator[Error]:
    """The folder rules for a scope (`top`) or a namespace: kind folders and namespaces, and nothing else."""
    for item in sorted(folder.iterdir()):
        if item.name.startswith("."):
            continue
        if item.is_dir() and item.name in KINDS:
            yield from _definitions(item)
        elif item.is_dir() and _NAME.fullmatch(item.name):
            yield from _layout(item, False)
        elif item.is_dir():
            yield Error((str(item),), "unexpected", "a namespace is a name: lower case words joined by hyphens")
        elif not (top and item.name == _SCOPE_FILE):
            yield Error((str(item),), "unexpected", "only kind folders and namespaces go in a scope or a namespace")


def _definitions(folder: Path) -> Iterator[Error]:
    """A kind folder holds definitions and nothing else: files `<name>.yaml`, or folders named by a name."""
    files = folder.name in FILE_KINDS
    for item in sorted(folder.iterdir()):
        if item.name.startswith("."):
            continue
        fits = item.is_file() and item.suffix == ".yaml" and _NAME.fullmatch(item.stem) if files else item.is_dir() and _NAME.fullmatch(item.name)
        if not fits:
            yield Error((str(item),), "unexpected", f"{folder.name}/ holds only " + ("<name>.yaml files" if files else "folders named by a name"))


def _load(name: str, found: list[tuple[str, Path]], known: Types, errors: list[Error]) -> list[tuple[Location, str]] | None:
    """Load the definition `name` found in `found`, and add it to `known`. Its references, or None when it is wrong."""
    if len(found) > 1:
        errors.append(Error((str(found[1][1]), "name"), "duplicate", f"{name!r} is also a {found[0][0]}; one id names one definition"))
        return None
    kind, path = found[0]
    here = str(path)
    schema = Schema.load(path, name.rpartition(".")[0])
    if isinstance(schema, list):
        errors.extend(Error((here, *error.path), error.code, error.message) for error in schema)
        return None
    problems = []
    if schema.name != name:
        problems.append(Error((here, "name"), "invalid", f"name must be {name.rpartition('.')[2]!r}, the file's name"))
    if kind == "type" and isinstance(schema.base, MappingType):
        problems.append(Error((here, "type"), "invalid", "a mapping belongs in schema/: type/ holds rules for one value"))
    if kind == "schema" and not isinstance(schema.base, MappingType):
        problems.append(Error((here, "type"), "invalid", "a schema is a mapping: a rule for one value belongs in type/"))
    if problems:
        errors.extend(problems)
        return None
    known.add(name, schema)
    return schema.references()


def _calls(steps: tuple[Step, ...]) -> Iterator[Call]:
    """Every call in `steps`, in order, inside the cases of a switch too."""
    for step in steps:
        if isinstance(step, Call):
            yield step
        elif isinstance(step, Switch):
            for branch in (*step.cases.values(), *([step.default] if step.default else [])):
                yield from _calls(branch)


def _check_kind(kind: str) -> None:
    if not isinstance(kind, str):
        raise TypeError(f"a kind must be text, got {kind!r}")
    if kind not in KINDS:
        raise ValueError(f"{kind!r} is not a kind: {', '.join(KINDS)}")


def _read(folder: Path, name: str, access: str) -> Scope | list[Error]:
    """The scope in `folder`, or the errors in its `scope.yaml`. `name` is what `scope.yaml`'s own `name` must equal:
    the folder's name for a scope of the default folder, or a library's declared name."""
    file = folder / _SCOPE_FILE
    here = (str(file),)
    if not file.is_file():
        return [Error(here, "missing", f"a scope has a {_SCOPE_FILE}")]
    data, errors = parse(file.read_text(encoding="utf-8"))
    if errors:
        return [Error(here + error.path, error.code, error.message) for error in errors]
    if errors := _known().validate(data, _SCOPE, here):
        return errors
    if data["name"] != name:
        return [Error(here + ("name",), "invalid", f"name must be {name!r}")]
    return Scope(data["name"], data["version"], data.get("description"), tuple(data.get("uses", [])), folder, access)


def _check_uses(scopes: Mapping[str, Scope]) -> list[Error]:
    """Every scope a scope uses is one there is, and no scope uses itself, straight or through others."""
    errors = [
        Error((str(scope.folder / _SCOPE_FILE), "uses", index), "unknown_scope", f"there is no scope named {used!r}")
        for scope in scopes.values()
        for index, used in enumerate(scope.uses)
        if used not in scopes
    ]
    for scope in scopes.values():
        if route := _cycle(scopes, scope.name):
            errors.append(Error((str(scope.folder / _SCOPE_FILE), "uses"), "cycle", " uses ".join(route)))
    return errors


def _cycle(scopes: Mapping[str, Scope], start: str) -> list[str] | None:
    """The scopes from `start`, through what each uses, back to `start`; or None. A name no scope has is passed over."""
    stack, seen = [(start, [start])], set()
    while stack:
        name, route = stack.pop()
        for used in scopes[name].uses:
            if used == start:
                return [*route, start]
            if used in scopes and used not in seen:
                seen.add(used)
                stack.append((used, [*route, used]))
    return None


def _walk(folder: Path, prefix: tuple[str, ...], kind: str) -> Iterator[Entry]:
    """The definitions of `kind` in `folder`, and in each namespace folder below it."""
    definitions = folder / kind
    if definitions.is_dir():
        for item in sorted(definitions.iterdir()):
            if kind in FILE_KINDS:
                name, is_definition, file = item.stem, item.is_file() and item.suffix == ".yaml", item
            else:
                name, is_definition, file = item.name, item.is_dir(), item / f"{kind}.yaml"
            if is_definition and _NAME.fullmatch(name):
                yield Entry(".".join((*prefix, name)), kind, file)
    for item in sorted(folder.iterdir()):
        if item.is_dir() and item.name not in KINDS and _NAME.fullmatch(item.name):
            yield from _walk(item, (*prefix, item.name), kind)


def register(types: Types) -> list[Error]:
    """Add the schema of `scope.yaml` to `types`, as `process_framework.scope` and the types it names."""
    found = Schema.load_all(files(__package__) / "schemas", __package__)
    if isinstance(found, list):
        return found
    for name, schema in found.items():
        types.add(name, schema)
    return []


def check_scope_name(name: str) -> list[Error]:
    """The errors of `name` as the name of a scope, by the schema of `scope.yaml`, each with the path `()`, or `[]`."""
    return _known().validate(name, f"{_SCOPE}-name")


@cache
def _known() -> Types:
    types = Types()
    if errors := register(types):
        raise RuntimeError(f"the schemas shipped with process_framework are wrong: {errors}")
    return types
