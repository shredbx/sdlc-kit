def test_render_gives_exactly_the_errors_in_the_case_and_no_files(case, template, types, located):
    errors = template.render(case["data"], types)
    assert isinstance(errors, list)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
