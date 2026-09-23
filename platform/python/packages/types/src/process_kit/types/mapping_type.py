from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, StrictStr, model_validator

from .error import Error, Location, unknown_type, wrong_type
from .field import Field
from .type import TYPE_CONFIG

if TYPE_CHECKING:
    from .lookup import Types


class MappingType(BaseModel):
    """A mapping. Bare, any mapping. With `fields`, only those keys, each checked by its field. With `keys`
    or `values`, any keys, each key or value checked by the type of that name. `fields` never comes with them."""

    model_config = TYPE_CONFIG

    fields: list[Field] | None = None
    values: StrictStr | None = None
    keys: StrictStr | None = None

    @model_validator(mode="after")
    def _fields_or_free_keys(self) -> "MappingType":
        if self.fields is not None and (self.values is not None or self.keys is not None):
            raise ValueError("a mapping has fields, or keys and values, not both")
        return self

    def validate(self, value: Any, path: Location, types: "Types") -> list[Error]:  # type: ignore[override]
        if not isinstance(value, dict):
            return [wrong_type(path, "a mapping", value)]
        if self.fields is not None:
            return self._validate_fields(value, path, types)
        return self._validate_entries(value, path, types)

    def _validate_fields(self, value: dict[Any, Any], path: Location, types: "Types") -> list[Error]:
        assert self.fields is not None
        declared = {field.name for field in self.fields}
        errors = [Error((*path, _segment(key)), "unexpected", f'"{key}" is not a declared field') for key in value if key not in declared]
        for field in self.fields:
            errors.extend(field.validate(value, path, types))
        return errors

    def _validate_entries(self, value: dict[Any, Any], path: Location, types: "Types") -> list[Error]:
        """Both types are looked up before any key is read. Then each key is checked, and then its value, at the key's path."""
        key_type, value_type = (None if name is None else types.get(name) for name in (self.keys, self.values))
        errors = [unknown_type(path, name) for name, found in ((self.keys, key_type), (self.values, value_type)) if name is not None and found is None]
        if errors:
            return errors
        for key, entry in value.items():
            child = (*path, _segment(key))
            if key_type is not None:
                errors.extend(key_type.validate(key, child, types))
            if value_type is not None:
                errors.extend(value_type.validate(entry, child, types))
        return errors

    def references(self) -> list[tuple[Location, str]]:
        found: list[tuple[Location, str]] = [(("fields", field.name, "type"), field.type) for field in self.fields or []]
        found += [((part,), name) for part, name in (("keys", self.keys), ("values", self.values)) if name is not None]
        return found


def _segment(key: Any) -> str | int:
    """A key as a step in a path: text and whole numbers stay as they are, anything else becomes text. A boolean
    is text too, because `yes` and `on` read as `true` in YAML, and a path should not hold one."""
    return key if isinstance(key, (str, int)) and not isinstance(key, bool) else str(key)
