def test_write_creates_or_replaces_the_definition_checked_first_and_the_file_holds_it_after(case, tree, results):
    expects = [call["expect"] for call in case["runs"]]
    assert [(code, out, len(err)) for code, out, err in results] == [(one["code"], one["stdout"], 0) for one in expects]
    texts = [one["text"] for one in expects if "text" in one]
    assert (tree / case["file"]).read_text(encoding="utf-8") == texts[-1]
