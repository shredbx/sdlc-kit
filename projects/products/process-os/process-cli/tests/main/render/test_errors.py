def test_render_that_cannot_go_on_prints_one_line_for_each_error_and_writes_nothing(case, tree, result):
    code, out, err = result
    assert (code, out) == (1, [])
    assert len(err) == len(case["expect"]["starts"])
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, case["expect"]["starts"]))
    assert not (tree / "output").exists()
