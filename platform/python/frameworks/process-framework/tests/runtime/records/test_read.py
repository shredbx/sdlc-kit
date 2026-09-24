from process_framework import Records
from process_kit.filesystem import Folder


def test_read_gives_the_data_in_the_file_and_no_errors(case, tree):
    records, errors = Records(Folder(tree / "records")).read(case["name"], case["folder"])
    assert (records, errors) == (case["expect"], [])
