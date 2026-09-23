from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, StrictStr

from .error import Error, Location, unknown_type, wrong_type
from .type import TYPE_CONFIG

if TYPE_CHECKING:
    from .lookup import Types


class SequenceType(BaseModel):
    """A list. Bare, any list. With `items`, every item must be of that type."""

    model_config = TYPE_CONFIG

    items: StrictStr | None = None

    def validate(self, value: Any, path: Location, types: "Types") -> list[Error]:  # type: ignore[override]
        if not isinstance(value, list):
            return [wrong_type(path, "a list", value)]
        if self.items is None:
            return []
        item_type = types.get(self.items)
        if item_type is None:
            return [unknown_type(path, self.items)]
        errors: list[Error] = []
        for index, item in enumerate(value):
            errors.extend(item_type.validate(item, (*path, index), types))
        return errors

    def references(self) -> list[tuple[Location, str]]:
        return [] if self.items is None else [(("items",), self.items)]
