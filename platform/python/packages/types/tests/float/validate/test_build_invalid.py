import pytest
from process_kit.types import FloatType
from pydantic import ValidationError


def test_building_a_float_type_raises_and_names_each_wrong_property(case):
    with pytest.raises(ValidationError) as raised:
        FloatType(**case["properties"])
    assert sorted(problem["loc"][0] for problem in raised.value.errors()) == sorted(case["expect"]["raises"])
