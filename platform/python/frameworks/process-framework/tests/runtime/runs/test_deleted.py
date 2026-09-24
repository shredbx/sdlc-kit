from process_framework import Runs
from process_kit.filesystem import Folder


def test_delete_removes_the_whole_run_folder_and_leaves_the_rest(case, tree):
    Runs(Folder(tree / "runs")).delete(case["run_id"])
    assert {str(path.relative_to(tree)) for path in tree.rglob("*") if path.is_file()} == set(case["expect"]["files"])
