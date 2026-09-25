def test_remove_deletes_the_definition_for_real(case, tree, results):
    expects = [call["expect"] for call in case["runs"]]
    assert [(code, out) for code, out, err in results] == [(one["code"], one["stdout"]) for one in expects]
    assert not (tree / case["file"]).exists()
