import builtins

import pytest
from process_kit.filesystem import Folder


def test_a_root_that_is_not_an_absolute_path_raises_the_error_in_the_case(case):
    with pytest.raises(getattr(builtins, case["raises"])):
        Folder(case["root"])
