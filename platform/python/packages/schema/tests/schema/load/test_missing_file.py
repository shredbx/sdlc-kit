import pytest
from process_kit.schema import Schema


def test_load_raises_file_not_found_for_a_missing_file(tmp_path, case):
    with pytest.raises(FileNotFoundError):
        Schema.load(tmp_path / "does-not-exist.yaml")
