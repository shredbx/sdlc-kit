def test_run_of_a_process_prints_the_lines_of_its_nodes_then_its_summary(case, result):
    code, out, err = result
    expect = case["expect"]
    assert (code, out, len(err)) == (expect["code"], expect["stdout"], len(expect["starts"]))
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, expect["starts"]))
