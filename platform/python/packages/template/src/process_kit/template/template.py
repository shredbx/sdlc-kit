"""A template: read it from its folder, and check that everything in it is valid."""

import os
from collections.abc import Iterator, Mapping
from dataclasses import dataclass
from functools import cache
from importlib.resources import files
from pathlib import Path
from types import MappingProxyType
from typing import Any

import jinja2
from process_kit.schema import Schema, parse, qualify, qualify_id
from process_kit.types import Error, Location, Types

from .engine import environment, syntax_error
from .source import Source

_TEMPLATE = "process_kit.template.template"
_FILE = "template.yaml"
_FILES = "files"
_JINJA = ".jinja"

Files = dict[str, str | bytes]


@dataclass(frozen=True)
class Template:
    """A folder with `template.yaml` and `files/`. `description` is the definition's own optional one-line
    summary. `pattern` maps a path to the lines that file must hold."""

    id: str
    description: str | None
    input: str
    pattern: Mapping[str, tuple[str, ...]]
    home: Path

    @classmethod
    def load(cls, folder: str | Path, namespace: str = "") -> "Template | list[Error]":
        """The template in `folder`, or every error in it, never both. Every error's path starts with the file it is about.
        A folder that is not there, has no `template.yaml`, or is a file, is a wrong call and raises."""
        folder = Path(os.path.abspath(folder))
        id = qualify_id(folder.name, namespace)
        file = folder / _FILE
        data, errors = parse(file.read_text(encoding="utf-8"))
        here = (str(file),)
        if errors:
            return [Error(here + error.path, error.code, error.message) for error in errors]
        if errors := _known().validate(data, _TEMPLATE, here):
            return errors

        pattern = data.get("pattern", {})
        if data["name"] != folder.name:
            errors.append(Error(here + ("name",), "invalid", f"name must be {folder.name!r}, the folder's name"))
        if not (folder / _FILES).is_dir():
            errors.append(Error((str(folder / _FILES),), "missing", f"a template has a {_FILES}/ folder"))
        errors += _file_errors(folder / _FILES)
        errors += _pattern_errors(pattern, here)
        if errors:
            return errors
        return cls(
            id,
            data.get("description"),
            qualify(data["input"], namespace),
            MappingProxyType({path: tuple(lines) for path, lines in pattern.items()}),
            folder,
        )

    def render(self, data: Any, types: Types) -> Files | list[Error]:
        """Every file this template makes, `{path: text or bytes}` in source-path order, or every error, never both, and
        nothing written. The data is checked against `input` first, and its errors, with paths in the data, are returned alone.
        Then each file's name and, for a `.jinja` file, its text are rendered with the data. A name the data lacks, a path that
        is empty, absolute or leaves the folder, and two files that make one path, are errors located in the template's file."""
        if errors := types.validate(data, self.input):
            return errors
        context = _context(data)
        made: Files = {}
        errors = []
        for file, relative in _files(self.home / _FILES):
            here = (str(file),)
            try:
                path = _render(relative, context).removesuffix(_JINJA)
            except jinja2.TemplateError as problem:
                errors.append(_problem(here, problem, "the name"))
                continue
            if bad := _bad_path(path):
                errors.append(Error(here, "invalid", f"the name renders to {path!r}, which {bad}"))
            elif path in made:
                errors.append(Error(here, "duplicate", f"the name renders to {path!r}, which another file makes"))
            elif (content := _content(file, context, here, errors)) is not None:
                made[path] = content
        return errors if errors else made

    def conform(self, data: Any, source: Source, types: Types) -> list[Error]:
        """What is wrong with the files `source` reads, or `[]`. The data is checked first, and its errors are the result. Then
        for each file of the `pattern`, its rendered path must be in `source` (`missing`), and each rendered line must be found
        in a line of it (`missing_line`), so a file may hold more, and a line may hold more. A finding's path is the rendered path."""
        if errors := types.validate(data, self.input):
            return errors
        context, errors = _context(data), []
        here = (str(self.home / _FILE), "pattern")
        for key, lines in self.pattern.items():
            try:
                path = _render(key, context)
            except jinja2.TemplateError as problem:
                errors.append(_problem(here + (key,), problem, "the path"))
                continue
            if bad := _bad_path(path):
                errors.append(Error(here + (key,), "invalid", f"the path renders to {path!r}, which {bad}"))
                continue
            text = source.read(path)
            if text is None:
                errors.append(Error((path,), "missing", "there is no such file"))
                continue
            held = text.splitlines()
            for index, line in enumerate(lines):
                try:
                    wanted = _render(line, context)
                except jinja2.TemplateError as problem:
                    errors.append(_problem(here + (key, index), problem, "the line"))
                    continue
                if not any(wanted in one for one in held):
                    errors.append(Error((path,), "missing_line", f"no line holds {wanted!r}"))
        return errors

    def references(self) -> list[tuple[Location, str]]:
        """The type name of the data, in full, with where it is written."""
        return [(("input",), self.input)]


def register(types: Types) -> list[Error]:
    """Add the schemas of `template.yaml` to `types`, as `process_kit.template.template` and the types it names."""
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
        raise RuntimeError(f"the schemas shipped with process_kit.template are wrong: {errors}")
    return types


def _file_errors(folder: Path) -> list[Error]:
    """Every name under `folder`, and every `.jinja` file, must be a valid template. `__pycache__` and `.DS_Store` are skipped."""
    if not folder.is_dir():
        return []
    errors = []
    for file, relative in _files(folder):
        if problem := syntax_error(relative):
            errors.append(Error((str(file),), "invalid", f"the name is not a valid template: {problem}"))
        if file.suffix == _JINJA:
            try:
                problem = syntax_error(file.read_bytes().decode("utf-8"))
            except UnicodeDecodeError:
                errors.append(Error((str(file),), "invalid", "not UTF-8 text"))
                continue
            if problem:
                errors.append(Error((str(file),), "invalid", f"not a valid template: {problem}"))
    return errors


def _pattern_errors(pattern: Mapping[str, list[str]], here: tuple[str, ...]) -> list[Error]:
    errors = []
    for path, lines in pattern.items():
        if problem := syntax_error(path):
            errors.append(Error(here + ("pattern", path), "invalid", f"not a valid template: {problem}"))
        errors += [
            Error(here + ("pattern", path, index), "invalid", f"not a valid template: {problem}")
            for index, line in enumerate(lines)
            if (problem := syntax_error(line))
        ]
    return errors


def _context(data: Any) -> Mapping[str, Any]:
    """The names a template may use: the top-level keys of the data, or none when the data is not a mapping."""
    return data if isinstance(data, Mapping) else {}


def _render(source: str, context: Mapping[str, Any]) -> str:
    return environment().from_string(source).render(context)


def _problem(here: Location, problem: jinja2.TemplateError, what: str) -> Error:
    if isinstance(problem, jinja2.UndefinedError):
        return Error(here, "undefined", f"{what} uses a name the data does not have: {problem.message}")
    return Error(here, "invalid", f"{what} cannot be rendered: {problem.message}")


def _bad_path(path: str) -> str | None:
    """Why `path` is not a place inside the folder a template makes, or None."""
    if not path:
        return "is empty"
    if path.startswith("/"):
        return "is absolute"
    if ".." in path.split("/"):
        return "leaves the folder"
    return None


def _files(folder: Path) -> Iterator[tuple[Path, str]]:
    """Each file under `folder` and its path from it, in path order. `__pycache__` and `.DS_Store` are skipped."""
    for file in sorted(path for path in folder.rglob("*") if path.is_file()):
        relative = file.relative_to(folder)
        if "__pycache__" not in relative.parts and relative.name != ".DS_Store":
            yield file, relative.as_posix()


def _content(file: Path, context: Mapping[str, Any], here: Location, errors: list[Error]) -> str | bytes | None:
    """A `.jinja` file rendered, or any other file as it is. None, and an error, when it cannot be."""
    if file.suffix != _JINJA:
        return file.read_bytes()
    try:
        return _render(file.read_bytes().decode("utf-8"), context)
    except UnicodeDecodeError:
        errors.append(Error(here, "invalid", "not UTF-8 text"))
    except jinja2.TemplateError as problem:
        errors.append(_problem(here, problem, "the file"))
    return None
