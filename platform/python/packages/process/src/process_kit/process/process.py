"""A process: read it from its file, and give every step and every name in it in full."""

import os
from collections.abc import Mapping
from dataclasses import dataclass
from functools import cache
from importlib.resources import files
from pathlib import Path
from types import MappingProxyType
from typing import Any

from process_kit.action import Action
from process_kit.action import register as register_actions
from process_kit.schema import Schema, parse, qualify, qualify_id
from process_kit.types import Error, Location, Types

from .steps import Call, Step, StepType, Stop, Switch

_PROCESS = "process_kit.process.process"
_STEP = "process_kit.process.step"


@dataclass(frozen=True)
class Process:
    """One process. `input`, `requires` and `output` map a name to the full name of its type, in the order written."""

    id: str
    description: str | None
    input: Mapping[str, str]
    requires: Mapping[str, str]
    output: Mapping[str, str]
    steps: tuple[Step, ...]
    home: Path

    @classmethod
    def load(cls, file: str | Path, namespace: str = "") -> "Process | list[Error]":
        """The process in `file`, `<name>.yaml`, or every error in it, never both. Every error's path starts with the file. A
        file that is not there, or is a folder, raises, and so does one not named `<name>.yaml` or a wrong namespace."""
        file = Path(os.path.abspath(file))
        if file.suffix != ".yaml":
            raise ValueError(f"a process is a file named <name>.yaml, got {file.name!r}")
        id = qualify_id(file.stem, namespace)
        data, errors = parse(file.read_text(encoding="utf-8"))
        here = (str(file),)
        if errors:
            return [Error(here + error.path, error.code, error.message) for error in errors]
        if errors := _known().validate(data, _PROCESS, here):
            return errors
        if data["name"] != file.stem:
            return [Error(here + ("name",), "invalid", f"name must be {file.stem!r}, the file's name")]
        return cls(
            id=id,
            description=data.get("description"),
            input=_full(data.get("input"), namespace),
            requires=_full(data.get("requires"), namespace),
            output=_full(data.get("output"), namespace),
            steps=tuple(_step(step, namespace) for step in data["steps"]),
            home=file,
        )

    @classmethod
    def of_action(cls, action: Action) -> "Process":
        """The process of one step that runs `action`: its input and output are the action's, so an action is run as any process is.
        A value that is not an `Action` raises."""
        if not isinstance(action, Action):
            raise TypeError(f"an Action is needed, got {action!r}")
        return cls(
            id=action.id,
            description=action.description,
            input=action.input,
            requires=MappingProxyType({}),
            output=action.output,
            steps=(Call(action.id),),
            home=action.home,
        )

    def references(self) -> list[tuple[Location, str]]:
        """Each type name the input, the requires and the output name, in full, with where it is written."""
        return [
            ((part, name), kind)
            for part, ports in (("input", self.input), ("requires", self.requires), ("output", self.output))
            for name, kind in ports.items()
        ]


def register(types: Types) -> list[Error]:
    """Add the schemas of a process to `types`, as `process_kit.process.process` and `.steps`, and the type `.step`, written in code. The
    types `process_kit.action.name`, `.ports` and `.description` must be there: register `action` first."""
    found = Schema.load_all(files(__package__) / "schemas", __package__)
    if isinstance(found, list):
        return found
    for name, schema in found.items():
        types.add(name, schema)
    types.add(_STEP, StepType())
    return []


@cache
def _known() -> Types:
    types = Types()
    if errors := register_actions(types) or register(types):
        raise RuntimeError(f"the schemas shipped with process_kit.process are wrong: {errors}")
    return types


def _full(ports: dict[str, str] | None, namespace: str) -> Mapping[str, str]:
    return MappingProxyType({name: qualify(kind, namespace) for name, kind in (ports or {}).items()})


def _step(data: Any, namespace: str) -> Step:
    """A step that has passed `StepType`, as a value."""
    if isinstance(data, str):
        return Call(qualify_id(data, namespace))
    if "stop" in data:
        return Stop(data["stop"])
    cases = {key: tuple(_step(step, namespace) for step in steps) for key, steps in data["cases"].items()}
    default = data.get("default")
    return Switch(data["switch"], MappingProxyType(cases), None if default is None else tuple(_step(step, namespace) for step in default))
