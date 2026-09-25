def test_validate_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree, framework, located):
    (tree / "data.yaml").write_text(case["data"], encoding="utf-8")
    errors = framework.validate(case["type"], tree / "data.yaml")
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
