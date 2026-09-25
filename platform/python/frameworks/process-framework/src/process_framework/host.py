"""What only the application knows, handed to `initialize`."""

from pathlib import Path
from typing import Protocol


class Host(Protocol):
    """The application's side of the seam. An application, such as the command line, implements it."""

    command: Path | None
    """The program an action calls back as `process-cli`, put on its PATH. `None` when there is none."""

    def report(self, event: object) -> None:
        """Show progress. The framework calls it once for each thing that happens in a run."""
        ...
