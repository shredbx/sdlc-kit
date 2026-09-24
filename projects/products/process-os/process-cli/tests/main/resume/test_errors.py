def test_resume_that_cannot_go_on_prints_one_line_for_each_error_and_exits_1(case, tree, run_twice):
    code, out, err = run_twice
    expect = case["expect"]
    assert (code, out) == (expect["code"], expect["stdout"])
    assert len(err) == len(expect["starts"])
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, expect["starts"]))
