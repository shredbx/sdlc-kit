def test_tools_lists_every_action_and_process_sorted_by_name(case, tree, framework):
    assert framework.tools() == case["expect"]
