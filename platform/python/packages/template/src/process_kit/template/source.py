"""Where conform reads the files it checks."""

from typing import Protocol, runtime_checkable


@runtime_checkable
class Source(Protocol):
    """Anything a file can be read from by path. A folder is one, without knowing it: a class is a `Source` by having `read`."""

    def read(self, path: str) -> str | None:
        """The text of the file at `path`, or None when there is none."""
        ...
