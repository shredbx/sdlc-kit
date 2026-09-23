from process_kit.types import StringType


def test_a_string_type_is_built_and_holds_the_properties_it_was_given(case):
    properties = case["properties"]
    string_type = StringType(**properties)
    for name, value in properties.items():
        assert getattr(string_type, name) == value
