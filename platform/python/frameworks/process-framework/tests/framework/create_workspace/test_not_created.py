from process_framework import create_workspace


def test_create_workspace_writes_nothing_and_names_the_files_that_are_there(case, tree, located):
    before = {str(path.relative_to(tree)): path.read_text() for path in tree.rglob("*") if path.is_file()}
    paths, errors = create_workspace(tree, case.get("scope", "main"))
    assert paths == []
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
    assert {str(path.relative_to(tree)): path.read_text() for path in tree.rglob("*") if path.is_file()} == before
