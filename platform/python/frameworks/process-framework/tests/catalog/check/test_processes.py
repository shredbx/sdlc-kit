from pathlib import Path

from process_framework import Catalog


def test_check_gives_exactly_the_errors_of_a_process_and_its_graph(case, tree):
    errors = Catalog.open(tree / "defs").check()
    shown = [((Path(error.path[0]).relative_to(tree).as_posix(), *error.path[1:]), error.code) for error in errors]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
