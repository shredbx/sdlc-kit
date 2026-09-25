from process_framework import Catalog


def test_locate_gives_none_for_what_leads_nowhere(case, tree):
    assert Catalog.open(tree / "defs").locate(case["kind"], case["ident"]) is None
