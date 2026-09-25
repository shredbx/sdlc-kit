def test_write_action_replaces_the_whole_folder_not_a_patch_and_runs_afterward(case, tree, results):
    expects = [call["expect"] for call in case["runs"]]
    assert [(code, out, len(err)) for code, out, err in results] == [(one["code"], one["stdout"], 0) for one in expects]
    folder = tree / case["file"]
    for relative, text in case["files_after"].items():
        assert (folder / relative).read_text(encoding="utf-8") == text
    for relative in case.get("gone", []):
        assert not (folder / relative).exists()
    seen = {path.relative_to(folder).as_posix() for path in folder.rglob("*") if path.is_file()}
    assert seen == set(case["files_after"])
