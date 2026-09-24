import builtins

import pytest
from process_kit.config import Config


def test_load_raises_for_a_file_that_cannot_be_read(case, tmp_path):
    with pytest.raises(getattr(builtins, case["raises"])):
        Config.load(tmp_path / case["file"])
