import builtins

import pytest
from process_kit.process import Run


def test_resume_with_a_process_that_is_not_the_target_or_not_a_process_raises(case, built):
    process = {"main": built.main}.get(case["given"], case["given"])
    with pytest.raises(getattr(builtins, case["raises"])):
        built.runner.resume(Run("r1", case["target"], {}, ""), process)
