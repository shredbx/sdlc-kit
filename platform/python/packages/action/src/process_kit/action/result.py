"""What one run of an action gave."""

from collections.abc import Mapping
from dataclasses import dataclass, field
from enum import StrEnum
from types import MappingProxyType
from typing import Any

from process_kit.types import Error


class Outcome(StrEnum):
    """How a run ended. `done` and `skipped` have outputs; `stopped` and `failed` have none."""

    done = "done"
    skipped = "skipped"
    stopped = "stopped"
    failed = "failed"


@dataclass(frozen=True)
class Result:
    """One run of an action. `reason` says why it did not go on, and `errors` says what was wrong with a value.
    `stdout` and `stderr` are what its scripts printed, across every hook that ran; `None`, for both, when none did
    (the inputs were wrong, so nothing was run)."""

    outcome: Outcome
    outputs: Mapping[str, Any] = field(default_factory=lambda: MappingProxyType({}))
    reason: str | None = None
    errors: tuple[Error, ...] = ()
    stdout: str | None = None
    stderr: str | None = None
