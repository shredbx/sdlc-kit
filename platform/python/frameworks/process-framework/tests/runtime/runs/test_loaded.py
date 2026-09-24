from process_framework import Run, Runs
from process_kit.filesystem import Folder


def test_load_gives_the_run_that_was_saved(case, tree):
    runs = Runs(Folder(tree / "runs"))
    runs.save(Run(**case["run"]))
    assert runs.load(case["run"]["id"]) == Run(**case["run"])
