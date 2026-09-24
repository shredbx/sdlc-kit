from process_framework import Catalog


def test_node_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree, located):
    errors = Catalog.open(tree / "defs").node(case["ident"])
    assert isinstance(errors, list)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
