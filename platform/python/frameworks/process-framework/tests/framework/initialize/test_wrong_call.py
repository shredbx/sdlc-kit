import pytest
from process_framework import initialize


def test_initialize_raises_for_a_config_file_that_does_not_exist(case, tree):
    with pytest.raises(FileNotFoundError):
        initialize(tree / case["file"], None)
