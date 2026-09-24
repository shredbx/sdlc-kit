import pytest
from process_kit.process import Run


def test_resume_of_a_run_whose_target_is_not_a_process_raises(case, built):
    with pytest.raises(ValueError):
        built.runner.resume(Run("r1", "a1", {}, ""))
