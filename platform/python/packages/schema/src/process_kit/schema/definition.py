"""Reads a definition. The only code in the kit that knows the definition format."""

from collections.abc import Hashable
from typing import Any, cast

import yaml
from process_kit.types import YAML_TYPES, Error, MappingType, SequenceType, Type
from pydantic import ValidationError
from pydantic_core import ErrorDetails
from yaml.nodes import MappingNode

from .names import qualify, qualify_id

_WRONG_TYPE = {"string_type", "int_type", "float_type", "bool_type", "list_type", "dict_type", "model_type"}


class _Loader(yaml.SafeLoader):
    """A SafeLoader that refuses a mapping key written twice, so a line is never silently overridden."""

    def construct_mapping(self, node: MappingNode, deep: bool = False) -> dict[Any, Any]:
        self.flatten_mapping(node)
        seen = set()
        for key_node, _ in node.value:
            key = self.construct_object(key_node, deep=deep)
            if isinstance(key, Hashable):
                if key in seen:
                    raise yaml.constructor.ConstructorError(
                        "while constructing a mapping", node.start_mark, f"found duplicate key {key!r}", key_node.start_mark
                    )
                seen.add(key)
        return super().construct_mapping(node, deep=deep)


def parse(text: str) -> tuple[Any, list[Error]]:
    """The YAML in `text` as `(data, errors)`, the reader every loader uses, and public for any package that reads
    a YAML file. The one error there can be is code `invalid`, for a syntax error or a mapping key written twice,
    and then `data` is None. A `text` that is not text is a wrong call and raises."""
    if not isinstance(text, str):
        raise TypeError(f"the text of a definition must be text, got {text!r}")
    try:
        return yaml.load(text, Loader=_Loader), []
    except yaml.YAMLError as problem:
        return None, [Error((), "invalid", f"not valid YAML: {_one_line(problem)}")]


def _one_line(problem: yaml.YAMLError) -> str:
    """What is wrong, and the line it is on, in one line: PyYAML's own text runs over several."""
    what = getattr(problem, "problem", None) or " ".join(str(problem).split())
    mark = getattr(problem, "problem_mark", None)
    return f"{what} (line {mark.line + 1})" if mark else what


def build(data: Any, namespace: str) -> tuple[str, Type, str | None] | list[Error]:
    """The full name, built type and own description of the definition in `data`, or every error in it. `data`
    is not changed."""
    if not isinstance(data, dict):
        return [Error((), "wrong_type", f"a definition must be a mapping, got {data!r}")]

    errors: list[Error] = []
    name = _name(data, errors)
    kind = _text(data, "type", errors)
    description = _optional_text(data, "description", errors)
    if description is not None:
        errors.extend(_description_errors(description))
    if kind is not None and kind not in YAML_TYPES:
        errors.append(Error(("type",), "unknown_type", f"there is no YAML type named {kind!r}"))
        kind = None
    built = None
    if kind is not None:
        properties = {key: value for key, value in data.items() if key not in ("name", "type", "description")}
        built = _build(YAML_TYPES[kind], properties, namespace, errors)
    if errors:
        return errors
    assert name is not None
    assert built is not None
    return qualify_id(name, namespace), built, description


def _name(data: dict[Any, Any], errors: list[Error]) -> str | None:
    """The definition's own name: text that is not empty, has no dot and is not the name of a YAML type."""
    name = _text(data, "name", errors)
    if name is None:
        return None
    if not name or "." in name:
        errors.append(Error(("name",), "invalid", f"a name is not empty and has no dot, got {name!r}; the package is given when it is loaded"))
    elif name in YAML_TYPES:
        errors.append(Error(("name",), "invalid", f"{name!r} is the name of a YAML type"))
    else:
        return name
    return None


def _text(data: dict[Any, Any], key: str, errors: list[Error]) -> str | None:
    if key not in data:
        errors.append(Error((key,), "missing", f'"{key}" is required'))
    elif not isinstance(value := data[key], str):
        errors.append(Error((key,), "wrong_type", f"expected text, got {value!r}"))
    else:
        return value
    return None


def _description_errors(description: str) -> list[Error]:
    """A description is one line, at most 255 characters — the same rule every other kind's own `description`
    type enforces; type and schema descriptions are parsed here, in bare Python, with no schema of their own
    to enforce it for them."""
    errors = []
    if len(description) > 255:
        errors.append(Error(("description",), "max_length", f"must be at most 255 characters, got {len(description)}"))
    if "\n" in description:
        errors.append(Error(("description",), "invalid", "a description is one line: no newline"))
    return errors


def _optional_text(data: dict[Any, Any], key: str, errors: list[Error]) -> str | None:
    """The text at `key`: `None` when it is left out, an error when it is given and is not text."""
    if key not in data:
        return None
    value = data[key]
    if not isinstance(value, str):
        errors.append(Error((key,), "wrong_type", f"expected text, got {value!r}"))
        return None
    return value


def _build(kind: type[Type], properties: dict[Any, Any], namespace: str, errors: list[Error]) -> Type | None:
    """Build the YAML type from the definition's properties. What the type raises becomes errors in the definition."""
    names = None
    if kind is MappingType:
        for key in ("values", "keys"):
            if isinstance(properties.get(key), str):
                properties[key] = qualify(properties[key], namespace)
        if "fields" in properties:
            names = _fields(properties, namespace, errors)
    elif kind is SequenceType and isinstance(properties.get("items"), str):
        properties["items"] = qualify(properties["items"], namespace)
    try:
        return cast(Type, kind.model_validate(properties))  # type: ignore[attr-defined]
    except ValidationError as failure:
        errors.extend(_error(problem, names) for problem in failure.errors())
        return None


def _fields(properties: dict[Any, Any], namespace: str, errors: list[Error]) -> list[Any] | None:
    """Turn the definition's `fields` (name -> entry) into the list a mapping holds, and return the names in order."""
    entries = properties["fields"]
    if not isinstance(entries, dict):
        errors.append(Error(("fields",), "wrong_type", f"expected a mapping of name to field, got {entries!r}"))
        del properties["fields"]
        return None
    names, listed = [], []
    for name, entry in entries.items():
        if not isinstance(entry, dict):
            errors.append(Error(("fields", _step(name)), "wrong_type", f"expected a field with a type, got {entry!r}"))
            continue
        if "name" in entry:
            errors.append(Error(("fields", _step(name), "name"), "unexpected", '"name" is not allowed here, the key is the name'))
            entry = {key: value for key, value in entry.items() if key != "name"}
        names.append(name)
        field = {**entry, "name": name}
        if isinstance(field.get("type"), str):
            field["type"] = qualify(field["type"], namespace)
        listed.append(field)
    properties["fields"] = listed
    return names


def _error(problem: ErrorDetails, names: list[Any] | None) -> Error:
    """One pydantic problem as one Error, located in the definition the way the author wrote it."""
    path = tuple(problem["loc"])
    if names is not None and len(path) >= 2 and path[0] == "fields" and isinstance(path[1], int):
        path = ("fields", _step(names[path[1]]), *path[2:])
    kind = problem["type"]
    if kind == "missing":
        return Error(path, "missing", f'"{path[-1]}" is required')
    if kind == "extra_forbidden":
        return Error(path, "unexpected", f'"{path[-1]}" is not allowed here')
    if kind in _WRONG_TYPE:
        return Error(path, "wrong_type", f"{problem['msg']}, got {problem['input']!r}")
    if kind == "value_error":
        return Error(path, "invalid", str(problem["ctx"]["error"]))
    return Error(path, "invalid", problem["msg"])


def _step(key: Any) -> str | int:
    return key if isinstance(key, (str, int)) else str(key)
