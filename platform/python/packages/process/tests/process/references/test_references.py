from process_kit.process import Process


def test_references_lists_the_types_of_the_ports_in_full_with_where_they_are_written(case, tree):
    process = Process.load(tree / "x.yaml", case["namespace"])
    assert process.references() == [(tuple(one["path"]), one["name"]) for one in case["expect"]]
