from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, StrictBool, StrictStr

from .error import Error, Location, unknown_type
from .type import TYPE_CONFIG

if TYPE_CHECKING:
    from .lookup import Types


class Field(BaseModel):
    """One named entry of a mapping. It holds the NAME of its type, looked up when it validates."""

    model_config = TYPE_CONFIG

    name: StrictStr
    type: StrictStr
    required: StrictBool = False
    description: StrictStr | None = None

    def validate(self, data: dict[Any, Any], path: Location, types: "Types") -> list[Error]:  # type: ignore[override]
        """Check the value this field finds under its name in `data`, the mapping it sits in."""
        child = (*path, self.name)
        if self.name not in data:
            return [Error(child, "missing", f'"{self.name}" is required')] if self.required else []
        found = types.get(self.type)
        if found is None:
            return [unknown_type(child, self.type)]
        return found.validate(data[self.name], child, types)
