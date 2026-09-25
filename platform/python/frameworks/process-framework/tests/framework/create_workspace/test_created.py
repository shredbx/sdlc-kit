from process_framework import create_workspace


def test_create_workspace_writes_the_three_files_beside_what_is_there_and_names_them(case, tree):
    assert create_workspace(tree / case.get("folder", "."), case.get("scope", "main")) == (case["expect"]["paths"], [])
    assert {name: (tree / name).read_text() for name in case["expect"]["files"]} == case["expect"]["files"]
