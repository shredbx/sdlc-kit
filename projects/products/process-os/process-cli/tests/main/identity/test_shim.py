def test_an_action_calling_process_cli_back_reaches_it_exactly_when_the_case_says(case, tree, result):
    code, out, err = result
    assert code == case["expect"]["code"]
    if "called" in case["expect"]:
        assert (tree / "output" / "called.txt").read_text() == case["expect"]["called"]
