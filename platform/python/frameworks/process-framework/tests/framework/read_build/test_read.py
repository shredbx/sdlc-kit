from process_framework import read_build


def test_read_build_gives_the_case_expects(case, tree):
    assert read_build(tree / case.get("file", "process-cli.build")) == case["expect"]
