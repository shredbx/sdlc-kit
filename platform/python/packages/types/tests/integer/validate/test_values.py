from process_kit.types import IntegerType, Types


def test_a_value_gets_exactly_the_errors_in_the_case_and_nothing_else(case):
    errors = IntegerType(**case["properties"]).validate(case["value"], tuple(case.get("path", [])), Types())
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
