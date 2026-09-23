from process_kit.types import wrong_type


def test_wrong_type_gives_an_error_at_the_path_with_the_code_and_a_message(case):
    error = wrong_type(tuple(case["path"]), case["expected"], case["value"])
    assert (error.path, error.code) == (tuple(case["path"]), "wrong_type")
    assert error.message
