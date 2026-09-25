from process_kit.types import YAML_TYPES, Types


def test_types_holds_the_ids_the_references_reach(case, reached):
    assert isinstance(reached, Types)
    assert sorted(name for name in reached.known if name not in YAML_TYPES) == case["expect"]["ids"]
