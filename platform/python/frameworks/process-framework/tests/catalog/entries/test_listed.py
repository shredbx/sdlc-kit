from process_framework import Catalog


def test_entries_lists_the_definitions_of_the_kind_in_order(case, tree):
    entries = Catalog.open(tree / "defs").entries(case["kind"])
    assert [(entry.id, entry.kind, entry.file) for entry in entries] == [(one["id"], case["kind"], tree / one["file"]) for one in case["expect"]]
