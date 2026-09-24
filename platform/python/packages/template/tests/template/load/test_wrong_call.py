import builtins

import pytest
from process_kit.template import Template


def test_load_raises_for_a_folder_or_a_namespace_that_is_wrong(case, tree):
    with pytest.raises(getattr(builtins, case["raises"])):
        Template.load(tree / case["folder"], case.get("namespace", ""))
