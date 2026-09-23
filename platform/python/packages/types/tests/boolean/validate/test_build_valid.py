from process_kit.types import BooleanType


def test_a_boolean_type_is_built_and_holds_the_properties_it_was_given(case):
    properties = case.get("properties", {})
    assert BooleanType(**properties).model_dump() == properties
