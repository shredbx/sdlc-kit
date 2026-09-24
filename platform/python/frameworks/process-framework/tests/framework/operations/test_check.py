from pathlib import Path

from process_framework import initialize


def test_check_gives_the_errors_of_the_catalog(case, tree, host):
    errors = initialize(tree / "processos.yaml", host).check()
    shown = [((Path(error.path[0]).relative_to(tree).as_posix(), *error.path[1:]), error.code) for error in errors]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
