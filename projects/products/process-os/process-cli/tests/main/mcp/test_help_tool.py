from process_cli.mcp import _help_tool


def test_help_tool_names_process_os_help_and_the_four_groups(case):
    tool = _help_tool()
    assert tool["name"] == case["expect"]["name"]
    assert tool["inputSchema"]["properties"]["group"]["enum"] == case["expect"]["groups"]
