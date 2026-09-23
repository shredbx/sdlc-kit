import inspect
from collections.abc import Mapping
from types import MappingProxyType
from typing import Any

from .error import Error, Location, unknown_type
from .type import Type
from .yaml_types import YAML_TYPES


class Types:
    """Every type known by name. Starts with the YAML types. Holds any `Type` and never needs to know what it is."""

    def __init__(self) -> None:
        self._known: dict[str, Type] = {name: kind() for name, kind in YAML_TYPES.items()}

    @property
    def known(self) -> Mapping[str, Type]:
        return MappingProxyType(self._known)

    def add(self, name: str, type: Type) -> None:
        """Make `type` usable as `name`. Raises if the name is taken, so nothing can replace `string`."""
        if inspect.isclass(type) or not isinstance(type, Type):
            raise TypeError(f"{name!r}: expected a Type, an object with validate() and references(), got {type!r}")
        if name in self._known:
            raise ValueError(f"a type named {name!r} already exists")
        self._known[name] = type

    def get(self, name: str) -> Type | None:
        return self._known.get(name)

    def validate(self, data: Any, name: str, path: Location = ()) -> list[Error]:
        """The front door: check `data`, which sits at `path`, against the type called `name`."""
        found = self.get(name)
        if found is None:
            return [unknown_type(path, name)]
        return found.validate(data, path, self)

    def check(self) -> list[Error]:
        """Every type name a held type refers to that is not held, located in the definition of the type that holds it."""
        return [
            unknown_type((name, *where), reference)
            for name, kind in self._known.items()
            for where, reference in kind.references()
            if reference not in self._known
        ]
