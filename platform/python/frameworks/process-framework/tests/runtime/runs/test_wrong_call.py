import builtins

import pytest
from process_framework import Runs
from process_kit.filesystem import Folder


def test_a_wrong_target_or_run_id_raises_the_error_in_the_case(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        getattr(Runs(Folder(tree / "runs")), case["call"])(*case["args"])
