def test_create_prints_one_line_for_each_error_in_the_case_and_exits_1_and_writes_nothing(case, tree, result):
    code, out, err = result
    assert (code, out) == (1, [])
    assert len(err) == len(case["expect"]["starts"])
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, case["expect"]["starts"]))
    for file in case.get("not_written", []):
        assert not (tree / file).exists()
