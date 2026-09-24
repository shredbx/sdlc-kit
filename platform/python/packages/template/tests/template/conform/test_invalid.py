def test_conform_gives_exactly_the_errors_in_the_case_and_nothing_else(case, template, types, source, located):
    errors = template.conform(case["data"], source, types)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
