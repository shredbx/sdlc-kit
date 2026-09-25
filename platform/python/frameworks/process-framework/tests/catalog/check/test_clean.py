from process_framework import Catalog


def test_check_finds_nothing_in_definitions_that_are_right(case, tree):
    assert Catalog.open(tree / "defs").check() == []
