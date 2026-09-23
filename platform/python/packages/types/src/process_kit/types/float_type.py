from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, StrictFloat

from .error import Error, Location, wrong_type
from .type import TYPE_CONFIG

if TYPE_CHECKING:
    from .lookup import Types


class FloatType(BaseModel):
    """A number. A whole number counts. Every property is optional and narrows it."""

    model_config = TYPE_CONFIG

    minimum: StrictFloat | None = None
    maximum: StrictFloat | None = None

    def validate(self, value: Any, path: Location, types: "Types") -> list[Error]:  # type: ignore[override]
        if not isinstance(value, (int, float)) or isinstance(value, bool):
            return [wrong_type(path, "a number", value)]
        errors: list[Error] = []
        if self.minimum is not None and value < self.minimum:
            errors.append(Error(path, "minimum", f"must be at least {self.minimum}, got {value}"))
        if self.maximum is not None and value > self.maximum:
            errors.append(Error(path, "maximum", f"must be at most {self.maximum}, got {value}"))
        return errors

    def references(self) -> list[tuple[Location, str]]:
        return []
