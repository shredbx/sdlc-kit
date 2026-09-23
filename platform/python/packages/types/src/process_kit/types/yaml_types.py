"""The types YAML itself has, by the name a definition file gives them."""

from collections.abc import Mapping
from types import MappingProxyType

from .boolean_type import BooleanType
from .float_type import FloatType
from .integer_type import IntegerType
from .mapping_type import MappingType
from .sequence_type import SequenceType
from .string_type import StringType
from .type import Type

YAML_TYPES: Mapping[str, type[Type]] = MappingProxyType(
    {
        "string": StringType,
        "integer": IntegerType,
        "float": FloatType,
        "boolean": BooleanType,
        "sequence": SequenceType,
        "mapping": MappingType,
    }
)
