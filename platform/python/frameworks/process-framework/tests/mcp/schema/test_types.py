from process_framework import input_schema


def test_input_schema_maps_each_yaml_type_and_builds_defs_once(case, types):
    schema = input_schema(case["ports"], types)
    expect = case["expect"]
    if "properties" in expect:
        assert schema["properties"] == expect["properties"]
    if "required" in expect:
        assert schema["required"] == expect["required"]
    assert schema.get("$defs", {}) == expect.get("defs", {})
