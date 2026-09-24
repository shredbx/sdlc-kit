def test_run_prints_what_the_case_expects_and_exits_with_its_code(case, result):
    code, out, err = result
    expect = case["expect"]
    assert (code, out, len(err)) == (expect["code"], expect["stdout"], len(expect["starts"]))
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, expect["starts"]))
