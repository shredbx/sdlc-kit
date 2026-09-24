import builtins

import pytest
from process_framework import create_workspace


def test_create_workspace_raises_the_error_in_the_case(case, tree, monkeypatch):
    monkeypatch.chdir(tree)
    with pytest.raises(getattr(builtins, case["raises"])):
        create_workspace(case["folder"], case.get("scope", "main"))
