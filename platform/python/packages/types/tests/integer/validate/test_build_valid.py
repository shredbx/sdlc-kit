from process_kit.types import IntegerType


def test_an_integer_type_is_built_and_holds_the_properties_it_was_given(case):
    properties = case["properties"]
    integer_type = IntegerType(**properties)
    for name, value in properties.items():
        assert getattr(integer_type, name) == value
