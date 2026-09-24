def test_show_run_of_an_unknown_or_wrong_id_gives_the_error(case, tree, framework, located):
    errors = framework.show_run(case["run_id"])
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
