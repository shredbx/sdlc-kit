import pytest
from process_framework import Catalog


def test_node_raises_for_an_id_that_is_not_text(case, tree):
    with pytest.raises(TypeError):
        Catalog.open(tree / "defs").node(case["ident"])
