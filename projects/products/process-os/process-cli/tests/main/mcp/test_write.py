import pytest
from mcp import MCPError, types

from process_cli.mcp import _write


def test_write_creates_or_replaces_or_gives_invalid_params(case, tree, framework):
    arguments = {"kind": case.get("kind"), "id": case.get("id"), "content": case.get("content")}
    if case.get("errors"):
        with pytest.raises(MCPError) as raised:
            _write(framework, arguments)
        assert raised.value.code == types.INVALID_PARAMS
        return
    result = _write(framework, arguments)
    assert result.is_error is False
    assert result.content[0].text == case["expect"]["text"]
