import pytest
from process_kit.types import IntegerType
from pydantic import ValidationError


def test_building_an_integer_type_raises_and_names_each_wrong_property(case):
    with pytest.raises(ValidationError) as raised:
        IntegerType(**case["properties"])
    assert sorted(problem["loc"][0] for problem in raised.value.errors()) == sorted(case["expect"]["raises"])
