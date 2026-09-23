from process_kit.schema import Schema


def test_a_wrong_mapping_definition_gets_exactly_the_errors_in_the_case(case):
    errors = Schema.loads(case["text"])
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
