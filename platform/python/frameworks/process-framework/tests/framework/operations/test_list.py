from process_framework import initialize


def test_list_gives_the_entries_of_the_kind(case, tree, host):
    entries = initialize(tree / "processos.yaml", host).list(case["kind"])
    assert [(entry.id, entry.file) for entry in entries] == [(one["id"], tree / one["file"]) for one in case["expect"]]
