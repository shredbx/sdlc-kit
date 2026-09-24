import builtins

import pytest
from process_framework import Catalog


def test_locate_raises_for_a_kind_or_an_id_that_is_wrong(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        Catalog.open(tree / "defs").locate(case["kind"], case["ident"])
