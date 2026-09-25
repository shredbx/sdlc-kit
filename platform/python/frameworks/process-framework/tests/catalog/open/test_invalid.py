from pathlib import Path

from process_framework import Catalog
from process_kit.config import Library


def test_open_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree):
    libraries = [Library(one["name"], tree / one["path"], one.get("access", "readonly"), one.get("version")) for one in case.get("libraries", [])]
    errors = Catalog.open(tree / "defs", libraries, tree / "processos.yaml")
    assert isinstance(errors, list)
    shown = [((Path(error.path[0]).relative_to(tree).as_posix(), *error.path[1:]), error.code) for error in errors]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
