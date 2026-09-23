"""What every check returns: one problem, and where it is."""

from dataclasses import dataclass
from typing import Any

Location = tuple[str | int, ...]


@dataclass(frozen=True)
class Error:
    """One problem. `path` says where, `code` says what kind, `message` says it in a sentence."""

    path: Location
    code: str
    message: str


def wrong_type(path: Location, expected: str, value: Any) -> Error:
    return Error(path, "wrong_type", f"expected {expected}, got {value!r}")


def unknown_type(path: Location, name: str) -> Error:
    return Error(path, "unknown_type", f"there is no type named {name!r}")
