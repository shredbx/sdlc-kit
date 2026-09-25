from process_framework import Runs
from process_kit.filesystem import Folder


def test_list_gives_every_runs_id_most_recent_first(case, tree):
    assert Runs(Folder(tree / "runs")).list() == case["expect"]
