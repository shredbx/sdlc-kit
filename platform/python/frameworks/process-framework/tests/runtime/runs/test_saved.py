import yaml
from process_framework import Run, Runs
from process_kit.filesystem import Folder


def test_save_writes_the_run_as_yaml_in_its_order(case, tree):
    Runs(Folder(tree / "runs")).save(Run(**case["run"]))
    saved = yaml.safe_load((tree / "runs" / case["run"]["id"] / "run.yaml").read_text())
    assert (saved, list(saved)) == (case["expect"], list(case["expect"]))
