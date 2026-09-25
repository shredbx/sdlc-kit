"""An action or a process as one MCP tool: its own description, and its input as JSON Schema."""

from collections.abc import Mapping
from typing import Any

from process_kit.action import Action
from process_kit.process import Process
from process_kit.schema import Schema
from process_kit.types import (
    BooleanType,
    FloatType,
    IntegerType,
    MappingType,
    SequenceType,
    StringType,
    Type,
    Types,
)


def tool_of(node: Action | Process, types: Types) -> dict[str, Any]:
    """One node — an action or a process — as one MCP tool descriptor: `name` is its id, `inputSchema` is
    `input_schema(node.input, types)`, and `description` is `node.description` — left out entirely, not
    `None`, when the node has none."""
    schema: dict[str, Any] = {"name": node.id, "inputSchema": input_schema(node.input, types)}
    if node.description is not None:
        schema["description"] = node.description
    return schema


def input_schema(ports: Mapping[str, str], types: Types) -> dict[str, Any]:
    """One tool's `inputSchema`: `ports` (name -> full type name) become `properties`, every name is
    `required` (an input is always required — `Records.gather` fills nothing in for one that is
    missing), and `$defs` holds every named type reached, each built once, `$ref`'d everywhere it is
    used."""
    defs: dict[str, dict[str, Any]] = {}
    properties = {name: _reference(name_of, types, defs) for name, name_of in ports.items()}
    schema: dict[str, Any] = {
        "type": "object",
        "properties": properties,
        "required": list(ports),
        "additionalProperties": False,
    }
    if defs:
        schema["$defs"] = defs
    return schema


def _reference(name: str, types: Types, defs: dict[str, dict[str, Any]]) -> dict[str, Any]:
    """The type `name` as one property's own schema: a `$ref` into `defs` for a named type (a
    `Schema`), built once — a placeholder goes in first, so a type that names itself, directly or
    through others, does not recurse forever — or the bare YAML type's own mapping, inline, for one
    of the six that is never named."""
    found = types.get(name)
    if isinstance(found, Schema):
        if name not in defs:
            defs[name] = {}
            body = _map(found.base, types, defs)
            defs[name] = {"description": found.description, **body} if found.description is not None else body
        return {"$ref": f"#/$defs/{name}"}
    return _map(found, types, defs)


def _map(kind: Type | None, types: Types, defs: dict[str, dict[str, Any]]) -> dict[str, Any]:
    """One YAML type's own JSON Schema (`mcp-research.md` §3's table), `$ref`-ing any named type it
    reaches — a field's own type, a sequence's `items`, a mapping's `keys` or `values` — through
    `_reference`."""
    if isinstance(kind, StringType):
        body: dict[str, Any] = {"type": "string"}
        if kind.min_length is not None:
            body["minLength"] = kind.min_length
        if kind.max_length is not None:
            body["maxLength"] = kind.max_length
        if kind.pattern is not None:
            body["pattern"] = kind.pattern
        if kind.enum is not None:
            body["enum"] = list(kind.enum)
        return body
    if isinstance(kind, IntegerType):
        return _bounded("integer", kind.minimum, kind.maximum)
    if isinstance(kind, FloatType):
        return _bounded("number", kind.minimum, kind.maximum)
    if isinstance(kind, BooleanType):
        return {"type": "boolean"}
    if isinstance(kind, SequenceType):
        body = {"type": "array"}
        if kind.items is not None:
            body["items"] = _reference(kind.items, types, defs)
        return body
    if isinstance(kind, MappingType):
        return _mapping(kind, types, defs)
    raise TypeError(f"not a YAML type: {kind!r}")


def _bounded(json_type: str, minimum: Any, maximum: Any) -> dict[str, Any]:
    body: dict[str, Any] = {"type": json_type}
    if minimum is not None:
        body["minimum"] = minimum
    if maximum is not None:
        body["maximum"] = maximum
    return body


def _mapping(kind: MappingType, types: Types, defs: dict[str, dict[str, Any]]) -> dict[str, Any]:
    if kind.fields is not None:
        properties, required = {}, []
        for field in kind.fields:
            entry = _reference(field.type, types, defs)
            properties[field.name] = {"description": field.description, **entry} if field.description is not None else entry
            if field.required:
                required.append(field.name)
        body: dict[str, Any] = {"type": "object", "properties": properties, "additionalProperties": False}
        if required:
            body["required"] = required
        return body
    if kind.keys is not None or kind.values is not None:
        body = {"type": "object"}
        if kind.keys is not None:
            body["propertyNames"] = _reference(kind.keys, types, defs)
        if kind.values is not None:
            body["additionalProperties"] = _reference(kind.values, types, defs)
        return body
    return {"type": "object"}
