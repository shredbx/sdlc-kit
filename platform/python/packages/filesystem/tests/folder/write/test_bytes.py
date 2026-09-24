from process_kit.filesystem import Folder


def test_write_puts_the_bytes_in_the_file_as_they_are(case, tree):
    Folder(tree / "work").write(*case["write"])
    assert (tree / case["expect"]["file"]).read_bytes() == case["expect"]["bytes"]
