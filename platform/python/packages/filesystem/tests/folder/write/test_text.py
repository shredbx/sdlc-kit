from process_kit.filesystem import Folder


def test_write_puts_the_text_in_the_file_as_utf8(case, tree):
    Folder(tree / "work").write(*case["write"])
    assert (tree / case["expect"]["file"]).read_bytes() == case["expect"]["text"].encode("utf-8")
