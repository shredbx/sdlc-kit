from process_framework import find_config


def test_find_config_gives_none_when_there_is_no_config_here_or_above(case, tree):
    assert find_config(tree / case["start"]) is None
