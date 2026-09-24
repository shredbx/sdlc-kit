from pathlib import Path

from process_framework import Error


def test_types_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree, reached):
    assert isinstance(reached, list)
    shown = [((Path(error.path[0]).relative_to(tree).as_posix(), *error.path[1:]), error.code) for error in reached]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(isinstance(error, Error) and error.message for error in reached)
