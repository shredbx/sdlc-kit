from process_kit.types import FloatType


def test_a_float_type_is_built_and_holds_the_properties_it_was_given(case):
    properties = case["properties"]
    float_type = FloatType(**properties)
    for name, value in properties.items():
        assert getattr(float_type, name) == value
