from process_kit.schema import Schema


def test_a_loaded_mapping_carries_its_own_optional_description(case):
    schema = Schema.loads(case["text"], case["namespace"])
    assert schema.description == case["expect"]
