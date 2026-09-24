import builtins

import pytest
from process_framework import Catalog


def test_open_raises_for_a_root_that_is_not_a_folder(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        Catalog.open(tree / case["root"])
