from process_framework import input_schema


def test_input_schema_against_the_real_demo(case, catalog):
    node = catalog.node(case["node"])
    assert not isinstance(node, list), node
    types = catalog.types_of(node)
    assert not isinstance(types, list), types
    assert input_schema(node.input, types) == case["expect"]
