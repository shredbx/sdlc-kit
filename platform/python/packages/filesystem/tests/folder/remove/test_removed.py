from process_kit.filesystem import Folder


def test_remove_deletes_the_file_or_folder_and_leaves_the_rest(case, tree):
    Folder(tree / "work").remove(case["relative"])
    assert {str(path.relative_to(tree)) for path in tree.rglob("*") if path.is_file()} == set(case["expect"]["files"])
