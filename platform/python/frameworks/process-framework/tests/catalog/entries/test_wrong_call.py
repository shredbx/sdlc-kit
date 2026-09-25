import builtins

import pytest
from process_framework import Catalog


def test_entries_raises_for_a_kind_that_is_wrong(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        Catalog.open(tree / "defs").entries(case["kind"])
