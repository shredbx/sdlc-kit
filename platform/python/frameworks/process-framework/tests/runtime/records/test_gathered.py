from process_framework import Records
from process_kit.filesystem import Folder


def test_gather_gives_the_values_the_case_expects_and_no_errors(case, tree, types):
    values, errors = Records(Folder(tree / "records")).gather(case["input"], case["requires"], case.get("records", {}), case.get("typed", {}), types)
    assert (values, errors) == (case["expect"], [])
