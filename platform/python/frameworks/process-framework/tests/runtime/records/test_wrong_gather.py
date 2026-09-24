import pytest
from process_framework import Records
from process_kit.filesystem import Folder


def test_gather_raises_for_a_data_or_a_typed_that_is_not_a_mapping(case, tree, types):
    with pytest.raises(TypeError):
        Records(Folder(tree / "records")).gather({}, {}, case.get("records", {}), case.get("typed", {}), types)
