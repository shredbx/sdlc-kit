import pytest
from process_framework import Catalog


def test_types_raises_for_a_file_that_is_in_no_scope_of_the_catalog(case, tree):
    with pytest.raises(ValueError):
        Catalog.open(tree / "defs").types([], tree / case["file"])
