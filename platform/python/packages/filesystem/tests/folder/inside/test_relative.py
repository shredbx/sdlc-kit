from process_kit.filesystem import Folder


def test_inside_says_whether_the_path_stays_in_the_root(case, tree):
    assert Folder(tree / "work").inside(case["relative"]) is case["expect"]
