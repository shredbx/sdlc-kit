"""The `framework` fixture this folder's own tests need, that no other `main/` test does — the four new
MCP tools (task 09, M6) call a real `ProcessFramework`, not just static package data the way `_help`/
`_help_tool` do."""

from types import SimpleNamespace

import pytest
from process_framework import initialize


@pytest.fixture
def framework(tree):
    """The framework for the config the case writes as `processos.yaml`, in `tree`."""
    host = SimpleNamespace(command=None, report=lambda event: None)
    made = initialize(tree / "processos.yaml", host)
    assert not isinstance(made, list), made
    return made
