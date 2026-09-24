"""An action: read it from its folder, and give every type name in it in full."""

import os
from collections.abc import Mapping
from dataclasses import dataclass
from functools import cache
from importlib.resources import files
from pathlib import Path
from types import MappingProxyType
from typing import Any

from process_kit.schema import Schema, parse, qualify, qualify_id
from process_kit.types import Error, Location, Types

from .context import Context
from .executor import Executor
from .result import Outcome, Result

_ACTION = "process_kit.action.action"
_FILE = "action.yaml"
_SCRIPT = "action.sh"


@dataclass(frozen=True)
class Action:
    """One action. `input` and `output` map a name to the full name of its type, in the order written. `timeout` is
    seconds one hook (`check.sh`, `pre.sh`, `action.sh` or `post.sh`) may run before it is failed as timed out — 60
    when the action does not say, 0 meaning no limit."""

    id: str
    kind: str
    description: str | None
    input: Mapping[str, str]
    output: Mapping[str, str]
    timeout: int
    home: Path

    @classmethod
    def load(cls, folder: str | Path, namespace: str = "") -> "Action | list[Error]":
        """The action in `folder`, or every error in it, never both. Every error's path starts with the file it is
        about. A folder that is not there, has no `action.yaml`, or is a file, is a wrong call and raises."""
        folder = Path(os.path.abspath(folder))
        id = qualify_id(folder.name, namespace)
        file = folder / _FILE
        data, errors = parse(file.read_text(encoding="utf-8"))
        here = (str(file),)
        if errors:
            return [Error(here + error.path, error.code, error.message) for error in errors]
        if errors := _known().validate(data, _ACTION, here):
            return errors

        if data["name"] != folder.name:
            errors.append(Error(here + ("name",), "invalid", f"name must be {folder.name!r}, the folder's name"))
        if not (folder / _SCRIPT).is_file():
            errors.append(Error((str(folder / _SCRIPT),), "missing", f"a shell action needs {_SCRIPT}"))
        if errors:
            return errors
        return cls(
            id=id,
            kind=data["type"],
            description=data.get("description"),
            input=_full(data.get("input"), namespace),
            output=_full(data.get("output"), namespace),
            timeout=data.get("timeout", 60),
            home=folder,
        )

    def references(self) -> list[tuple[Location, str]]:
        """Each type name the action's inputs and then its outputs name, in full, with where it is written."""
        return [((part, name), kind) for part, ports in (("input", self.input), ("output", self.output)) for name, kind in ports.items()]

    def run(self, inputs: Mapping[str, Any], context: Context, types: Types, executor: Executor) -> Result:
        """Check the inputs, hand the action to `executor`, and check the outputs. A wrong input or output is a `failed`
        result with the errors, and a wrong input runs nothing. `inputs` that is not a mapping is a wrong call and raises."""
        if not isinstance(inputs, Mapping):
            raise TypeError(f"inputs must be a mapping of name to value, got {inputs!r}")
        errors = _problems(self.input, inputs, types)
        errors += [Error((name,), "unexpected", f'"{name}" is not an input of {self.id}') for name in inputs if name not in self.input]
        if errors:
            return Result(Outcome.failed, reason="the inputs are wrong", errors=tuple(errors))
        result = executor.execute(self, {name: inputs[name] for name in self.input}, context)
        if result.outcome not in (Outcome.done, Outcome.skipped):
            return result
        if errors := _problems(self.output, result.outputs, types):
            return Result(Outcome.failed, reason="the outputs are wrong", errors=tuple(errors), stdout=result.stdout, stderr=result.stderr)
        return Result(
            result.outcome,
            MappingProxyType({name: result.outputs[name] for name in self.output}),
            result.reason,
            stdout=result.stdout,
            stderr=result.stderr,
        )


def register(types: Types) -> list[Error]:
    """Add the schemas of `action.yaml` to `types`, as `process_kit.action.action`, `.description`, `.kind`, `.name`, `.ports`, `.timeout` and `.type-name`."""
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
        raise RuntimeError(f"the schemas shipped with process_kit.action are wrong: {errors}")
    return types


def _full(ports: dict[str, str] | None, namespace: str) -> Mapping[str, str]:
    return MappingProxyType({name: qualify(kind, namespace) for name, kind in (ports or {}).items()})


def _problems(ports: Mapping[str, str], values: Mapping[str, Any], types: Types) -> list[Error]:
    """Each declared name, in order: missing, or the errors of its value against its type, located at the name."""
    errors: list[Error] = []
    for name, kind in ports.items():
        if name not in values:
            errors.append(Error((name,), "missing", f'"{name}" is required'))
        else:
            errors.extend(types.validate(values[name], kind, (name,)))
    return errors
