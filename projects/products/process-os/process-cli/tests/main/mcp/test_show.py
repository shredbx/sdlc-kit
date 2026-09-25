import pytest
from mcp import MCPError, types

from process_cli.mcp import _show


def test_show_reads_a_definition_or_a_record_or_gives_invalid_params(case, tree, framework):
    if case.get("errors"):
        with pytest.raises(MCPError) as raised:
            _show(framework, {"kind": case.get("kind"), "id": case.get("id")})
        assert raised.value.code == types.INVALID_PARAMS
        return
    result = _show(framework, {"kind": case["kind"], "id": case["id"]})
    assert result.is_error is False
    if "text" in case["expect"]:
        assert result.content[0].text == case["expect"]["text"]
        assert result.structured_content == {"text": case["expect"]["text"]}
    else:
        assert result.structured_content == {"files": case["expect"]["files"]}
