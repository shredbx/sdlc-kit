from pathlib import Path

from process_kit.template import Template


def test_load_gives_exactly_the_errors_in_the_case_and_nothing_else(case, tree):
    errors = Template.load(tree / case["folder"])
    assert isinstance(errors, list)
    shown = [((Path(error.path[0]).relative_to(tree).as_posix(), *error.path[1:]), error.code) for error in errors]
    assert shown == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
