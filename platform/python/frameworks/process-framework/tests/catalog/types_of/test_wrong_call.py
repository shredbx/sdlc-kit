import pytest
from process_framework import Catalog


def test_types_of_raises_for_a_node_that_is_not_one(case, tree):
    with pytest.raises(TypeError):
        Catalog.open(tree / "defs").types_of(case["node"])
