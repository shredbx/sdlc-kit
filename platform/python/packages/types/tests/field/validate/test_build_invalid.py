import pytest
from process_kit.types import Field
from pydantic import ValidationError


def test_building_a_field_raises_with_exactly_the_problems_in_the_case(case):
    with pytest.raises(ValidationError) as raised:
        Field(**case["properties"])
    assert [(tuple(problem["loc"]), problem["type"]) for problem in raised.value.errors()] == [(tuple(one["loc"]), one["type"]) for one in case["expect"]]
