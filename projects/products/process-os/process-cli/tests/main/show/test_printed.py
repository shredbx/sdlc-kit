def test_show_prints_the_definitions_own_raw_text_and_exits_0(case, result):
    code, out, err = result
    assert (code, out, err) == (0, case["expect"]["stdout"], [])
