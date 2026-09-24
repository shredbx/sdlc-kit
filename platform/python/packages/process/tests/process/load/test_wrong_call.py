import builtins

import pytest
from process_kit.process import Process


def test_load_raises_for_a_file_or_a_namespace_that_is_wrong(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        Process.load(tree / case["file"], case.get("namespace", ""))
