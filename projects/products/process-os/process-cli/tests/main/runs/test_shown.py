def test_runs_run_prints_its_status_its_nodes_and_its_logs(case, result):
    code, out, err = result
    expect = case["expect"]
    assert (code, out) == (expect["code"], expect["stdout"])
    if "starts" in expect:
        assert len(err) == len(expect["starts"])
        assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, expect["starts"]))
