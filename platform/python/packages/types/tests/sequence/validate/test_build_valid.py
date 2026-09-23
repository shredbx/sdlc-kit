from process_kit.types import SequenceType


def test_a_sequence_type_is_built_and_holds_the_properties_it_was_given(case):
    properties = case["properties"]
    sequence_type = SequenceType(**properties)
    for name, value in properties.items():
        assert getattr(sequence_type, name) == value
