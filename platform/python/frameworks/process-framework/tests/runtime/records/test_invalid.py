from process_framework import Records
from process_kit.filesystem import Folder


def test_read_gives_exactly_the_errors_in_the_case_and_no_data(case, tree, located):
    records, errors = Records(Folder(tree / "records")).read(case["name"], case["folder"])
    assert records is None
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
