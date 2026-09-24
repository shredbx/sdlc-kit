from process_framework import Runs
from process_kit.filesystem import Folder


def test_load_gives_none_for_a_run_that_is_not_there_or_an_id_that_leaves_the_folder(case, tree):
    assert Runs(Folder(tree / "runs")).load(case["run_id"]) is None
