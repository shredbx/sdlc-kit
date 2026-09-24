import pytest
from process_kit.filesystem import Folder


def test_make_raises_for_a_root_that_is_a_file(case, tree):
    with pytest.raises(FileExistsError):
        Folder(tree / case["root"]).make()
