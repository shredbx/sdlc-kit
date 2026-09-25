from process_framework import Runs
from process_kit.filesystem import Folder


def test_log_writes_stdout_and_stderr_in_the_nodes_own_folder(case, tree):
    runs = Runs(Folder(tree / "runs"))
    runs.log(case["run_id"], case["node"], case["stdout"], case["stderr"])
    base = tree / "runs" / case["run_id"] / case["node"]
    stdout = (base / "stdout.log").read_text() if (base / "stdout.log").is_file() else ""
    stderr = (base / "stderr.log").read_text() if (base / "stderr.log").is_file() else ""
    assert {"stdout": stdout, "stderr": stderr} == case["expect"]
    if not case["stdout"]:
        assert not (base / "stdout.log").exists()
    if not case["stderr"]:
        assert not (base / "stderr.log").exists()
