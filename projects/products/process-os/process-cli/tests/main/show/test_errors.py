def test_show_prints_one_line_for_the_error_in_the_case_and_exits_1(case, result):
    code, out, err = result
    assert (code, out) == (1, [])
    assert len(err) == 1
    assert err[0].startswith(case["expect"]["starts"][0]) and len(err[0]) > len(case["expect"]["starts"][0])
