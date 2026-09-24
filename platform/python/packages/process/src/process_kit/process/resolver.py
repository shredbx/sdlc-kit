"""Where a node comes from."""

from typing import Protocol, runtime_checkable

from process_kit.action import Action
from process_kit.types import Error

from .process import Process


@runtime_checkable
class Resolver(Protocol):
    """Anything that can say what an id names. The framework's catalog is one: a class is a `Resolver` by having `node`."""

    def node(self, id: str) -> Action | Process | list[Error]:
        """The action or the process with that id, or the errors that say why there is none."""
        ...
