from process_framework import find_config


def test_find_config_gives_the_file_the_case_expects(case, tree):
    assert find_config(tree / case["start"]) == tree / case["expect"]
