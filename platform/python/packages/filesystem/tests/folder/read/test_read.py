from process_kit.filesystem import Folder


def test_read_gives_the_text_of_the_file_or_none(case, tree):
    assert Folder(tree / "work").read(case["relative"]) == case["expect"]
