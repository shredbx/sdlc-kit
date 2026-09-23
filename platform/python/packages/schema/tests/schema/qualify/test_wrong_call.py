import builtins

import process_kit.schema as schema
import pytest


def test_a_wrong_call_raises_the_error_in_the_case(case):
    with pytest.raises(getattr(builtins, case["raises"])):
        getattr(schema, case["call"])(case["name"], case["namespace"])
