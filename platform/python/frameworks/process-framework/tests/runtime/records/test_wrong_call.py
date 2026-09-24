import pytest
from process_framework import Records
from process_kit.filesystem import Folder


def test_read_raises_for_a_name_or_a_folder_that_is_not_text(case, tree):
    with pytest.raises(TypeError):
        Records(Folder(tree / "records")).read(case["name"], case["folder"])
