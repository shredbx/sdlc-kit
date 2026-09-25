def test_init_that_cannot_write_prints_one_line_for_each_error_and_exits_1_writing_nothing(case, tree, result):
    code, out, err = result
    assert (code, out) == (1, [])
    assert len(err) == len(case["expect"]["starts"])
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, case["expect"]["starts"]))
    assert sorted(path.name for path in tree.iterdir()) == sorted(case.get("files", {}))
