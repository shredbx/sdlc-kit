def test_run_gives_exactly_the_errors_in_the_case_and_saves_nothing(case, tree, framework, located):
    errors = framework.run(case.get("action", "sdlc.greet"), {**case.get("inputs", {"msg": "hi"}), **case.get("values", {})}, case.get("typed"))
    assert isinstance(errors, list)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
    assert not (tree / "runs").exists()
