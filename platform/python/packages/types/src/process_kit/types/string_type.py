import re
from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, StrictInt, StrictStr, field_validator

from .error import Error, Location, wrong_type
from .type import TYPE_CONFIG

if TYPE_CHECKING:
    from .lookup import Types


class StringType(BaseModel):
    """Text. Every property is optional and narrows it."""

    model_config = TYPE_CONFIG

    min_length: StrictInt | None = None
    max_length: StrictInt | None = None
    pattern: StrictStr | None = None
    enum: list[StrictStr] | None = None

    @field_validator("pattern")
    @classmethod
    def _must_be_a_regular_expression(cls, value: str | None) -> str | None:
        if value is not None:
            try:
                re.compile(value)
            except re.error as problem:
                raise ValueError(f"not a valid regular expression: {problem}") from None
        return value

    def validate(self, value: Any, path: Location, types: "Types") -> list[Error]:  # type: ignore[override]
        if not isinstance(value, str):
            return [wrong_type(path, "a string", value)]
        errors: list[Error] = []
        if self.min_length is not None and len(value) < self.min_length:
            errors.append(Error(path, "min_length", f"must be at least {self.min_length} characters, got {len(value)}"))
        if self.max_length is not None and len(value) > self.max_length:
            errors.append(Error(path, "max_length", f"must be at most {self.max_length} characters, got {len(value)}"))
        if self.pattern is not None and re.fullmatch(self.pattern, value) is None:
            errors.append(Error(path, "pattern", f"must match {self.pattern}, got {value!r}"))
        if self.enum is not None and value not in self.enum:
            errors.append(Error(path, "enum", f"must be one of {self.enum}, got {value!r}"))
        return errors

    def references(self) -> list[tuple[Location, str]]:
        return []
