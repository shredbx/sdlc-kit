from process_framework import Runs
from process_kit.filesystem import Folder


def test_prune_keeps_the_most_recent_done_runs_and_leaves_the_rest(case, tree):
    Runs(Folder(tree / "runs")).prune(case["keep"])
    remaining = sorted(path.parent.name for path in (tree / "runs").glob("*/run.yaml"))
    assert remaining == sorted(case["expect"])
