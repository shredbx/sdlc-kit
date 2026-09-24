from process_kit.filesystem import Folder


def test_exists_says_whether_a_file_or_a_folder_is_there(case, tree):
    assert Folder(tree / "work").exists(case["relative"]) is case["expect"]
