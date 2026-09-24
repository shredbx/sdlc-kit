import pytest
from mcp import MCPError, types

from process_cli.mcp import _help


def test_help_gives_the_groups_own_text_or_the_same_invalid_params_a_call_that_never_starts_gives(case):
    if case.get("errors"):
        with pytest.raises(MCPError) as raised:
            _help(case["group"])
        assert raised.value.code == types.INVALID_PARAMS
        return
    result = _help(case["group"])
    assert result.is_error is False
    assert result.content[0].text.startswith(case["expect"]["heading"])
