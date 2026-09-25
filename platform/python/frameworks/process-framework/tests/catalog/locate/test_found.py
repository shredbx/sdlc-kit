from process_framework import Catalog


def test_locate_gives_the_path_of_the_definition(case, tree):
    assert Catalog.open(tree / "defs").locate(case["kind"], case["ident"]) == tree / case["expect"]
