from pathlib import Path

from process_kit.process import Process


def test_load_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree):
    errors = Process.load(tree / "x.yaml")
    assert isinstance(errors, list)
    shown = [((Path(error.path[0]).relative_to(tree).as_posix(), *error.path[1:]), error.code) for error in errors]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
