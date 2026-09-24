def test_check_prints_the_line_in_the_case_and_exits_0(case, result):
    assert result == (0, case["expect"]["stdout"], [])
