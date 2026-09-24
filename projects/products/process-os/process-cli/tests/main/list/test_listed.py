def test_list_prints_one_line_for_each_definition_in_the_case_and_exits_0(case, result):
    assert result == (0, case["expect"]["stdout"], [])
