def test_conform_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree, framework, located):
    errors = framework.conform(case.get("template", "sdlc.pkg"), tree / "data.yaml", case.get("folder", "made"))
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
