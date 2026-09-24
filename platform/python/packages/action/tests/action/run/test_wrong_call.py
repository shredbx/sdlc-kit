import builtins

import pytest
from process_kit.action import Context, ShellExecutor
from process_kit.types import Types


def test_run_raises_for_inputs_or_a_folder_that_are_wrong(case, tree, built):
    with pytest.raises(getattr(builtins, case["raises"])):
        built.run(case["inputs"], Context(tree / case["folder"]), Types(), ShellExecutor())
