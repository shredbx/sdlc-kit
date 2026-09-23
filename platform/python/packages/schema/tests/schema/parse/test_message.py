from process_kit.schema import parse


def test_a_yaml_error_is_one_line_that_ends_with_the_line_it_is_on(case):
    _, errors = parse(case["text"])
    (error,) = errors
    assert "\n" not in error.message
    assert error.message.endswith(f"(line {case['line']})")
