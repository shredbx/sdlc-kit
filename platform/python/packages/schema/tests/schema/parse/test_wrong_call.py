import pytest
from process_kit.schema import parse


def test_parse_raises_a_type_error_for_anything_but_text(case):
    with pytest.raises(TypeError):
        parse(case["value"])
