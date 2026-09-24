import builtins

import pytest


def test_a_wrong_call_raises_the_error_in_the_case(case, framework):
    with pytest.raises(getattr(builtins, case["raises"])):
        getattr(framework, case["call"])(*case["args"])
