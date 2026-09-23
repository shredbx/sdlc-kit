from process_kit.schema import Schema


def test_a_loaded_mapping_names_its_key_and_value_types_in_full(case):
    schema = Schema.loads(case["text"], case["namespace"])
    assert schema.references() == [(tuple(one["path"]), one["name"]) for one in case["expect"]]
