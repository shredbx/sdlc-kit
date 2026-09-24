import pytest
from process_framework import find_config


def test_find_config_raises_for_a_start_that_is_not_a_folder(case, tree):
    with pytest.raises(NotADirectoryError):
        find_config(tree / case["start"])
