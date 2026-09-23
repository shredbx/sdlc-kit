"""The one question every type answers: what is wrong with this value?"""

from typing import TYPE_CHECKING, Any, Protocol, runtime_checkable

from pydantic import ConfigDict

from .error import Error, Location

if TYPE_CHECKING:
    from .lookup import Types

# Every type is read-only once built, and refuses a property it does not have.
TYPE_CONFIG = ConfigDict(frozen=True, extra="forbid")


@runtime_checkable
class Type(Protocol):
    """Anything a field can name as its type. A class is a type by having these two methods, and
    inherits nothing: a pydantic class cannot inherit a protocol, so the YAML types conform the same way."""

    def validate(self, value: Any, path: Location, types: "Types") -> list[Error]:
        """The errors in `value`, which sits at `path`. Empty when the value is fine."""
        ...

    def references(self) -> list[tuple[Location, str]]:
        """Each type name this type refers to, with where it is written. Empty when it refers to none."""
        ...
