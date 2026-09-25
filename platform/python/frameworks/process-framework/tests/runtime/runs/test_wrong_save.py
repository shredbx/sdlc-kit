import builtins

import pytest
from process_framework import Run, Runs
from process_kit.filesystem import Folder


def test_save_raises_for_an_id_that_leaves_the_runs_folder(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        Runs(Folder(tree / "runs")).save(Run(**case["run"]))
