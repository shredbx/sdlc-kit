from process_framework import Catalog


def test_types_of_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree, located):
    catalog = Catalog.open(tree / "defs")
    errors = catalog.types_of(catalog.node(case["ident"]))
    assert isinstance(errors, list)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
