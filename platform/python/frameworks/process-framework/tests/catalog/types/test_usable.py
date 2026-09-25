def test_the_types_reached_check_data_by_their_full_names(case, reached):
    errors = reached.validate(case["value"], case["type"])
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
