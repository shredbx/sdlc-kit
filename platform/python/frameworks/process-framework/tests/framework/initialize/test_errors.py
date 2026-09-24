from pathlib import Path
from types import SimpleNamespace

from process_framework import initialize


def test_initialize_gives_exactly_the_errors_in_the_case_and_no_framework(case, tree):
    errors = initialize(tree / "processos.yaml", SimpleNamespace(command=None, report=lambda event: None))
    assert isinstance(errors, list)
    shown = [((Path(error.path[0]).relative_to(tree).as_posix(), *error.path[1:]), error.code) for error in errors]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
