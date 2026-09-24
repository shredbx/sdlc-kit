"""The config file: read it, check it, and give every path in it resolved."""

import os
from dataclasses import dataclass
from functools import cache
from importlib.resources import files
from pathlib import Path

from process_kit.schema import Schema, parse
from process_kit.types import Error, Types

_CONFIG = "process_kit.config.config"


@dataclass(frozen=True)
class Runtime:
    """The folder runs work in, and the three folders inside it. Every path is absolute. `history` is
    the most recent done runs kept; 10 when the config does not say."""

    root: Path
    records: Path
    output: Path
    runs: Path
    history: int = 10


@dataclass(frozen=True)
class Library:
    """One entry of `libraries`: a scope kept in another folder, beside `definitions`. `path` is absolute. Whether the
    folder is there, and whether its `scope.yaml` really names this `name` and this `version`, is not checked here —
    that needs the folder read, which is the catalog's business."""

    name: str
    path: Path
    access: str
    version: str | None


@dataclass(frozen=True)
class Config:
    """A config file, with every path in it resolved: absolute, and with no `..` left in it. `version` is the
    shape of the file itself, when it says one — `None` for a config written before that field existed."""

    file: Path
    definitions: Path
    runtime: Runtime
    version: str | None = None
    libraries: tuple[Library, ...] = ()

    @classmethod
    def load(cls, file: str | Path) -> "Config | list[Error]":
        """The config in `file`, or every error in it — never both. Every error's path starts with the file.
        A file that is not there, or is a folder, is a wrong call and raises."""
        file = Path(os.path.abspath(file))
        data, errors = parse(file.read_text(encoding="utf-8"))
        here = (str(file),)
        if errors:
            return [Error(here + error.path, error.code, error.message) for error in errors]
        if errors := _known().validate(data, _CONFIG, here):
            return errors

        base, runtime = file.parent, data["runtime"]
        definitions, root = _absolute(base, data["definitions"]), _absolute(base, runtime["root"])
        problems: list[Error] = []
        if not definitions.is_dir():
            problems.append(Error(here + ("definitions",), "not_found", f"there is no folder {definitions}"))
        if not root.is_dir():
            problems.append(Error(here + ("runtime", "root"), "not_found", f"there is no folder {root}"))
        inside = {}
        for name in ("records", "output", "runs"):
            path = _absolute(root, runtime.get(name, name))
            if path != root and root not in path.parents:
                problems.append(Error(here + ("runtime", name), "outside_root", f"{path} is not inside {root}"))
            inside[name] = path
        libraries = tuple(
            Library(entry["name"], _absolute(base, entry["path"]), entry.get("access", "readonly"), entry.get("version")) for entry in data.get("libraries", [])
        )
        version = data.get("version")
        history = runtime.get("history", 10)
        return problems if problems else cls(file, definitions, Runtime(root, **inside, history=history), version, libraries)


def register(types: Types) -> list[Error]:
    """Add the schemas of the config file to `types`, as `process_kit.config.config`, `.runtime` and `.path`."""
    found = Schema.load_all(files(__package__) / "schemas", __package__)
    if isinstance(found, list):
        return found
    for name, schema in found.items():
        types.add(name, schema)
    return []


@cache
def _known() -> Types:
    types = Types()
    if errors := register(types):
        raise RuntimeError(f"the schemas shipped with process_kit.config are wrong: {errors}")
    return types


def _absolute(base: Path, written: str) -> Path:
    """`written` resolved against `base`. Symlinks are not followed, so the paths match what was written."""
    return Path(os.path.normpath(base / written))
