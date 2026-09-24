import pytest
from mcp import MCPError, types
from process_framework import Error, Run

from process_cli.mcp import _result


def test_result_turns_call_into_what_an_mcp_client_reads(case):
    if "errors" in case:
        made = [Error(tuple(one["path"]), one["code"], one["code"]) for one in case["errors"]]
        with pytest.raises(MCPError) as raised:
            _result(made)
        assert raised.value.code == types.INVALID_PARAMS
        return
    made = Run(
        id="r1", target=case["target"], scopes={}, digest="d", status=case["status"],
        done=[], skipped=[], context={}, nested={}, output={}, reason=case.get("reason"),
    )
    result = _result(made)
    assert result.is_error is False
    assert result.content[0].text == case["expect"]["text"]
    assert result.structured_content == made.to_data()
