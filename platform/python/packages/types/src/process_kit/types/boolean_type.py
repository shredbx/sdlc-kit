from typing import TYPE_CHECKING, Any

from pydantic import BaseModel

from .error import Error, Location, wrong_type
from .type import TYPE_CONFIG

if TYPE_CHECKING:
    from .lookup import Types


class BooleanType(BaseModel):
    """True or false. It has no properties."""

    model_config = TYPE_CONFIG

    def validate(self, value: Any, path: Location, types: "Types") -> list[Error]:  # type: ignore[override]
        if not isinstance(value, bool):
            return [wrong_type(path, "true or false", value)]
        return []

    def references(self) -> list[tuple[Location, str]]:
        return []
