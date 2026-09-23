from dataclasses import dataclass
from importlib.resources.abc import Traversable
from pathlib import Path
from typing import Any

from process_kit.types import Error, Location, Type, Types

from .definition import build, parse
from .names import check_namespace


@dataclass(frozen=True)
class Schema:
    """A named type someone wrote. It is based on one configured YAML type, held as `base`.

    `name` is the full name — the package it came from, a dot, and the name written in the definition —
    because that is what a field names and what `Types` holds it under. It is a `Type` by having
    `validate` and `references`, and inherits nothing. `description` is the definition's own optional
    sentence; `None` for one that never wrote it."""

    name: str
    base: Type
    description: str | None = None

    @classmethod
    def load(cls, source: str | Path, namespace: str = "") -> "Schema | list[Error]":
        """The schema in the file at `source`, or every error in it — never both, never half a schema.
        `namespace` is the package the definition belongs to; empty means none."""
        check_namespace(namespace)
        return cls.loads(Path(source).read_text(encoding="utf-8"), namespace)

    @classmethod
    def loads(cls, text: str, namespace: str = "") -> "Schema | list[Error]":
        """The schema in the YAML `text`, or every error in it."""
        check_namespace(namespace)
        data, errors = parse(text)
        return errors if errors else cls.from_data(data, namespace)

    @classmethod
    def from_data(cls, data: Any, namespace: str = "") -> "Schema | list[Error]":
        """The schema in `data`, a definition already parsed. `data` is not changed."""
        check_namespace(namespace)
        result = build(data, namespace)
        if isinstance(result, list):
            return result
        name, base, description = result
        return cls(name, base, description)

    @classmethod
    def load_all(cls, location: Path | Traversable, namespace: str = "") -> "dict[str, Schema] | list[Error]":
        """Every `*.yaml` file directly in `location`, in file-name order, keyed by full name — or every error
        in all of them, each path starting with the file name. Two files with one name are an error."""
        check_namespace(namespace)
        found: dict[str, tuple[str, Schema]] = {}
        errors: list[Error] = []
        for entry in sorted(location.iterdir(), key=lambda entry: entry.name):
            if entry.name.startswith(".") or not entry.name.endswith(".yaml") or not entry.is_file():
                continue
            result = cls.loads(entry.read_text(encoding="utf-8"), namespace)
            if isinstance(result, list):
                errors.extend(Error((entry.name, *error.path), error.code, error.message) for error in result)
            elif result.name in found:
                errors.append(Error((entry.name, "name"), "invalid", f"{result.name!r} is already defined in {found[result.name][0]}"))
            else:
                found[result.name] = (entry.name, result)
        return errors if errors else {name: schema for name, (_, schema) in found.items()}

    def validate(self, value: Any, path: Location, types: Types) -> list[Error]:
        return self.base.validate(value, path, types)

    def references(self) -> list[tuple[Location, str]]:
        return self.base.references()
