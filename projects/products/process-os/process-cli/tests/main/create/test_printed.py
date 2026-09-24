def test_create_prints_the_line_in_the_case_and_exits_0_and_writes_the_file(case, tree, result):
    code, out, err = result
    assert (code, out, err) == (0, case["expect"]["stdout"], [])
    written = tree / case["expect"]["file"]
    assert written.read_text(encoding="utf-8") == case["expect"]["text"]
