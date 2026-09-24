def test_runs_prints_one_line_for_each_run_most_recent_first_and_exits_0(case, result):
    assert result == (0, case["expect"]["stdout"], [])
