from process_framework import Catalog
from process_kit.types import YAML_TYPES, Types


def test_types_of_holds_the_ids_the_node_and_what_it_calls_reach(case, tree):
    catalog = Catalog.open(tree / "defs")
    known = catalog.types_of(catalog.node(case["ident"]))
    assert isinstance(known, Types)
    assert sorted(name for name in known.known if name not in YAML_TYPES) == case["expect"]
