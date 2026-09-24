from process_framework import Records
from process_kit.filesystem import Folder


def test_gather_gives_exactly_the_errors_in_the_case_in_order(case, tree, types, located):
    values, errors = Records(Folder(tree / "records")).gather(case["input"], case["requires"], case.get("records", {}), case.get("typed", {}), types)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
