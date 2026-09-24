def test_a_wrong_call_prints_the_usage_and_exits_2(case, result):
    code, out, err = result
    assert (code, out) == (2, [])
    assert len(err) == len(case["expect"]["starts"])
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, case["expect"]["starts"]))
