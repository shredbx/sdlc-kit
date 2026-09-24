import builtins

import pytest
from process_kit.filesystem import Folder


def test_a_wrong_path_raises_the_error_in_the_case_and_writes_nothing(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        getattr(Folder(tree / "work"), case["call"])(*case["args"])
    assert not (tree / "x").exists()
