"""Where a run is kept."""

from typing import Protocol, runtime_checkable

from .run import Run


@runtime_checkable
class RunStore(Protocol):
    """Anything a run can be kept in. The framework's `Runs` is one: a class is a `RunStore` by having these four."""

    def new_id(self, target: str) -> str:
        """A name for a new run of `target`, that no other run has."""
        ...

    def save(self, run: Run) -> None:
        """Keep the run as it is now, replacing what was kept for its id."""
        ...

    def load(self, run_id: str) -> Run | None:
        """The run kept under that id, or None."""
        ...

    def log(self, run_id: str, node: str, stdout: str, stderr: str) -> None:
        """Keep what the node's scripts printed while it ran, replacing what was kept for it."""
        ...
