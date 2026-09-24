def test_check_prints_the_message_in_the_case_to_stderr_and_exits_1(case, result):
    assert result == (1, [], case["expect"]["stderr"])
