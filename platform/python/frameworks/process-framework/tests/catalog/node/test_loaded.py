from process_framework import Catalog
from process_kit.action import Action


def test_node_gives_the_action_with_the_id_and_its_types_in_full(case, tree):
    action = Catalog.open(tree / "defs").node(case["ident"])
    assert isinstance(action, Action)
    assert (action.id, dict(action.input), dict(action.output)) == (case["ident"], case["expect"]["input"], case["expect"]["output"])
