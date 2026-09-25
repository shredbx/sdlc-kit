def test_edit_merges_one_field_and_leaves_the_rest_untouched(case, tree, result):
    code, out, err = result
    assert (code, out, err) == (0, case["expect"]["stdout"], [])
    assert (tree / case["expect"]["file"]).read_text(encoding="utf-8") == case["expect"]["text"]
