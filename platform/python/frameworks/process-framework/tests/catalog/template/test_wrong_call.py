import pytest
from process_framework import Catalog


def test_template_raises_for_an_id_that_is_not_text(case, tree):
    with pytest.raises(TypeError):
        Catalog.open(tree / "defs").template(case["ident"])
