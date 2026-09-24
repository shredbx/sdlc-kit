def test_create_name_lets_one_folder_hold_more_than_one_instance_of_the_same_schema(case, tree, results):
    expects = [call["expect"] for call in case["runs"]]
    assert [(code, out, len(err)) for code, out, err in results] == [(one["code"], one["stdout"], 0) for one in expects]
    for file, text in case["files_after"].items():
        assert (tree / file).read_text(encoding="utf-8") == text
