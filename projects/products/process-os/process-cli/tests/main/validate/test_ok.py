def test_validate_prints_the_line_in_the_case_and_exits_0(case, tree, result):
    assert result == (0, case["expect"]["stdout"], [])
