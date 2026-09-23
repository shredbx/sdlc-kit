import pytest
from process_kit.types import StringType
from pydantic import ValidationError


def test_building_a_string_type_raises_and_names_each_wrong_property(case):
    with pytest.raises(ValidationError) as raised:
        StringType(**case["properties"])
    assert sorted(problem["loc"][0] for problem in raised.value.errors()) == sorted(case["expect"]["raises"])
