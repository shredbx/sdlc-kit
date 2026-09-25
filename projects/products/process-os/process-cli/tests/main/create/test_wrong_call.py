def test_a_wrong_call_exits_2_and_prints_only_the_usage(case, result):
    code, out, err = result
    assert (code, out) == (2, [])
    assert err
