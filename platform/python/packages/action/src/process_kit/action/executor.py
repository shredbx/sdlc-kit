"""How the scripts of an action run."""

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, Protocol, runtime_checkable

from .context import Context
from .result import Result

if TYPE_CHECKING:
    from .action import Action


@runtime_checkable
class Executor(Protocol):
    """Anything that can run an action. A class is an executor by having `execute`, and inherits nothing."""

    def execute(self, action: "Action", inputs: Mapping[str, Any], context: Context) -> Result:
        """Run `action` on `inputs`, which are already checked. Its result holds the outputs as read, unchecked."""
        ...
