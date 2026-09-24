def test_render_prints_one_line_for_each_file_and_writes_what_the_case_says(case, tree, result):
    assert result == (0, case["expect"]["stdout"], [])
    written = {path: (tree / path).read_text() if (tree / path).is_file() else None for path in case["expect"]["files"]}
    assert written == case["expect"]["files"]
