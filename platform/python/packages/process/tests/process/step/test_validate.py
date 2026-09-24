from process_kit.process import StepType
from process_kit.types import Types


def test_a_step_gets_exactly_the_errors_in_the_case_and_nothing_else(case):
    errors = StepType().validate(case["value"], tuple(case.get("path", [])), Types())
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
    assert StepType().references() == []
