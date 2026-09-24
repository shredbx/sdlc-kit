"""The state of one run, and what is told about it as it goes."""

import copy
from dataclasses import dataclass, field
from typing import Any

from process_kit.types import Error


@dataclass
class Run:
    """One run of a process. The runner changes it and saves it after every node. `context` is the top process's, and `nested`
    holds the context of each nested process that is under way, by the key of its node, so a resume can go on inside it."""

    id: str
    target: str
    scopes: dict[str, str]
    digest: str
    status: str = "running"
    done: list[str] = field(default_factory=list)
    skipped: list[str] = field(default_factory=list)
    context: dict[str, Any] = field(default_factory=dict)
    nested: dict[str, dict[str, Any]] = field(default_factory=dict)
    output: dict[str, Any] = field(default_factory=dict)
    reason: str | None = None
    errors: list[Error] = field(default_factory=list)

    def to_data(self) -> dict[str, Any]:
        """The run as plain data, a copy that shares nothing with it."""
        data: dict[str, Any] = {
            "id": self.id,
            "process": self.target,
            "scopes": dict(self.scopes),
            "digest": self.digest,
            "status": self.status,
            "done": list(self.done),
            "skipped": list(self.skipped),
            "context": copy.deepcopy(self.context),
            "nested": copy.deepcopy(self.nested),
            "output": copy.deepcopy(self.output),
        }
        if self.reason is not None:
            data["reason"] = self.reason
        if self.errors:
            data["errors"] = [{"path": list(error.path), "code": error.code, "message": error.message} for error in self.errors]
        return data

    @classmethod
    def from_data(cls, data: dict[str, Any]) -> "Run":
        """The run in `data`, as `to_data` gives it."""
        return cls(
            id=data["id"],
            target=data["process"],
            scopes=dict(data["scopes"]),
            digest=data["digest"],
            status=data["status"],
            done=list(data["done"]),
            skipped=list(data["skipped"]),
            context=copy.deepcopy(data["context"]),
            nested=copy.deepcopy(data["nested"]),
            output=copy.deepcopy(data["output"]),
            reason=data.get("reason"),
            errors=[Error(tuple(one["path"]), one["code"], one["message"]) for one in data.get("errors", [])],
        )


@dataclass(frozen=True)
class Event:
    """One thing that happened to a node: `started`, `done`, `skipped`, `stopped` or `failed`."""

    run: str
    node: str
    state: str
    reason: str | None = None
