def test_show_action_prints_one_header_and_its_own_text_per_file_in_order(case, result):
    code, out, err = result
    assert (code, out, err) == (0, case["expect"]["stdout"], [])
