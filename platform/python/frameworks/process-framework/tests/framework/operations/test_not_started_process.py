def test_run_of_a_process_gives_exactly_the_errors_in_the_case_and_saves_nothing(case, tree, framework, located):
    errors = framework.run(case["target"], case["inputs"])
    assert isinstance(errors, list)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
    assert framework.host.events == []
    assert not (tree / "runs").exists()
