def test_resume_of_a_run_that_cannot_go_on_gives_exactly_the_errors_in_the_case(case, tree, framework, located):
    errors = framework.resume(case["run_id"])
    assert isinstance(errors, list)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
