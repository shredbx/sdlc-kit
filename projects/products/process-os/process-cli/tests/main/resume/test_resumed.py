def test_resume_goes_on_with_the_first_call_run_and_prints_only_what_runs_now(case, tree, run_twice):
    code, out, err = run_twice
    expect = case["expect"]
    assert (code, out, len(err)) == (expect["code"], expect["stdout"], len(expect["starts"]))
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, expect["starts"]))
