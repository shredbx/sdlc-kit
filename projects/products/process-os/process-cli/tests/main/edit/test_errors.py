def test_edit_prints_one_line_for_the_error_and_leaves_the_file_exactly_as_it_was(case, tree, result):
    code, out, err = result
    assert (code, out) == (1, [])
    assert len(err) == len(case["expect"]["starts"])
    assert all(line.startswith(start) and len(line) > len(start) for line, start in zip(err, case["expect"]["starts"]))
    for file, text in case.get("unchanged", {}).items():
        assert (tree / file).read_text(encoding="utf-8") == text
