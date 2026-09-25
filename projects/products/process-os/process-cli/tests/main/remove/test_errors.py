def test_remove_prints_one_line_for_the_error_and_removes_nothing(case, tree, result):
    code, out, err = result
    assert (code, out) == (1, [])
    assert len(err) == len(case["expect"]["starts"])
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, case["expect"]["starts"]))
    for file in case.get("unchanged", []):
        assert (tree / file).is_file()
