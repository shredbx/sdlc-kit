from process_cli.mcp import _list


def test_list_gives_ids_or_record_paths_filtered_and_described_as_the_arguments_say(case, tree, framework):
    result = _list(framework, case["arguments"])
    assert result.is_error is False
    assert result.content[0].text.splitlines() == case["expect"]["lines"]
