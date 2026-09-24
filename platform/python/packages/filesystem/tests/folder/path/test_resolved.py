from process_kit.filesystem import Folder


def test_path_is_the_absolute_path_with_the_dots_folded(case, tree):
    assert Folder(tree / "work").path(case["relative"]) == tree / case["expect"]
