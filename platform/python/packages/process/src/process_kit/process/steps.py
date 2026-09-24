"""The steps of a process: a node, a choice on a value, an end on purpose. And the type of a step in a definition."""

import re
from collections.abc import Mapping
from dataclasses import dataclass
from typing import TYPE_CHECKING, Any

from process_kit.types import Error, Location, wrong_type

if TYPE_CHECKING:
    from process_kit.types import Types

_NAMES = re.compile(r"[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)*")
_SWITCH_KEYS = ("switch", "cases", "default")


@dataclass(frozen=True)
class Call:
    """A node: an action or a process, by its id in full."""

    id: str


@dataclass(frozen=True)
class Switch:
    """A choice on a value in the context: `on` is a name and then fields, joined by dots."""

    on: str
    cases: Mapping[str, tuple["Step", ...]]
    default: tuple["Step", ...] | None


@dataclass(frozen=True)
class Stop:
    """The end of the run, on purpose."""

    reason: str


Step = Call | Switch | Stop


class StepType:
    """The type of a step in a definition (G2): an id, a `switch` or a `stop`. It is a `Type` by having `validate` and
    `references`, and refers to no type, since a step names nodes."""

    def validate(self, value: Any, path: Location, types: "Types") -> list[Error]:
        if isinstance(value, str):
            return _names(value, path)
        if not isinstance(value, dict):
            return [wrong_type(path, "a step: an id, a switch or a stop", value)]
        kinds = [key for key in ("switch", "stop") if key in value]
        if len(kinds) != 1:
            return [Error(path, "invalid", "a step is an id, a switch or a stop, and only one of them")]
        return self._stop(value, path) if kinds == ["stop"] else self._switch(value, path, types)

    def references(self) -> list[tuple[Location, str]]:
        return []

    def _stop(self, value: dict[str, Any], path: Location) -> list[Error]:
        errors = _unexpected(value, path, ("stop",))
        reason = value["stop"]
        if not isinstance(reason, str):
            errors.append(wrong_type((*path, "stop"), "text, the reason", reason))
        elif not reason:
            errors.append(Error((*path, "stop"), "min_length", "the reason must not be empty"))
        return errors

    def _switch(self, value: dict[str, Any], path: Location, types: "Types") -> list[Error]:
        errors = _unexpected(value, path, _SWITCH_KEYS)
        if not isinstance(value["switch"], str):
            errors.append(wrong_type((*path, "switch"), "text, a name and then fields joined by dots", value["switch"]))
        else:
            errors += _names(value["switch"], (*path, "switch"))
        if "cases" not in value:
            errors.append(Error((*path, "cases"), "missing", '"cases" is required'))
        elif not isinstance(value["cases"], dict):
            errors.append(wrong_type((*path, "cases"), "a mapping of value to steps", value["cases"]))
        else:
            for key, steps in value["cases"].items():
                here = (*path, "cases", _segment(key))
                if not isinstance(key, str):
                    errors.append(wrong_type(here, "text, a value of the type", key))
                errors += self._steps(steps, here, types)
        if "default" in value:
            errors += self._steps(value["default"], (*path, "default"), types)
        return errors

    def _steps(self, value: Any, path: Location, types: "Types") -> list[Error]:
        if not isinstance(value, list):
            return [wrong_type(path, "a list of steps", value)]
        return [error for index, step in enumerate(value) for error in self.validate(step, (*path, index), types)]


def _names(value: str, path: Location) -> list[Error]:
    return [] if _NAMES.fullmatch(value) else [Error(path, "pattern", f"must match {_NAMES.pattern}, got {value!r}")]


def _unexpected(value: dict[str, Any], path: Location, allowed: tuple[str, ...]) -> list[Error]:
    return [Error((*path, _segment(key)), "unexpected", f'"{key}" is not part of this step') for key in value if key not in allowed]


def _segment(key: Any) -> str | int:
    """A key as a step in a path: text and whole numbers stay as they are, and anything else, a boolean too, becomes text."""
    return key if isinstance(key, (str, int)) and not isinstance(key, bool) else str(key)
