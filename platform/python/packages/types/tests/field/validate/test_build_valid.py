from process_kit.types import Field


def test_a_field_carries_its_own_optional_description(case):
    field = Field(**case["properties"])
    assert field.description == case["expect"]
