from process_kit.schema import parse


def test_parse_gives_the_data_and_exactly_the_errors_in_the_case(case):
    data, errors = parse(case["text"])
    assert data == case["expect"]["data"]
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]["errors"]]
    assert all(error.message for error in errors)
