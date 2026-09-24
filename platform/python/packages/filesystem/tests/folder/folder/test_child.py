from process_kit.filesystem import Folder


def test_a_child_folder_has_the_root_and_refuses_what_leaves_it(case, tree):
    child = Folder(tree / "work").folder(case["relative"])
    assert child.root == tree / case["expect"]["root"]
    assert {path: child.inside(path) for path in case["expect"]["inside"]} == case["expect"]["inside"]
