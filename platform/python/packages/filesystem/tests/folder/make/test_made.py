from process_kit.filesystem import Folder


def test_make_makes_the_root_and_leaves_what_is_in_it(case, tree):
    Folder(tree / case["root"]).make()
    assert (tree / case["root"]).is_dir()
    assert {name: (tree / name).read_text() for name in case.get("files", {})} == case.get("files", {})
