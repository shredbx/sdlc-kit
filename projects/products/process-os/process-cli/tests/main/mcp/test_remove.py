import pytest
from mcp import MCPError, types

from process_cli.mcp import _remove


def test_remove_deletes_or_gives_invalid_params(case, tree, framework):
    arguments = {"kind": case.get("kind"), "id": case.get("id")}
    if case.get("errors"):
        with pytest.raises(MCPError) as raised:
            _remove(framework, arguments)
        assert raised.value.code == types.INVALID_PARAMS
        return
    result = _remove(framework, arguments)
    assert result.is_error is False
    assert result.content[0].text == case["expect"]["text"]
