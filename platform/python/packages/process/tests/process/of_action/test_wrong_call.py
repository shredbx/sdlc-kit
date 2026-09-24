import builtins

import pytest
from process_kit.process import Process


def test_of_action_raises_for_a_value_that_is_not_an_action(case):
    with pytest.raises(getattr(builtins, case["raises"])):
        Process.of_action(case["value"])
