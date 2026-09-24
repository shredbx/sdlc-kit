"""Gives each test its cases. A test file `tests/<entity>/<action>/test_<scope>.py` is fed by
`fixtures/<entity>/<action>/<scope>.yaml`, one call per case, with the plan's id as the test id."""

from pathlib import Path

import pytest
import yaml

TESTS = Path(__file__).parent
FIXTURES = TESTS.parent / "fixtures"


def pytest_generate_tests(metafunc):
    if "case" not in metafunc.fixturenames:
        return
    module = Path(metafunc.module.__file__)
    scope = module.stem.removeprefix("test_").replace("_", "-")
    source = FIXTURES / module.parent.relative_to(TESTS) / f"{scope}.yaml"
    cases = yaml.safe_load(source.read_text(encoding="utf-8"))
    metafunc.parametrize("case", cases, ids=[case["id"] for case in cases])


@pytest.fixture
def tree(case, tmp_path):
    """The case's `folders` and `files`, written into a temporary folder, which is what the fixture gives."""
    for folder in case.get("folders", []):
        (tmp_path / folder).mkdir(parents=True, exist_ok=True)
    for name, text in case.get("files", {}).items():
        (tmp_path / name).parent.mkdir(parents=True, exist_ok=True)
        (tmp_path / name).write_text(text, encoding="utf-8")
    return tmp_path


@pytest.fixture
def shape():
    """A function that writes a step as plain data, to compare it with a case."""
    from process_kit.process import Call, Stop, Switch

    def show(step):
        if isinstance(step, Call):
            return {"call": step.id}
        if isinstance(step, Stop):
            return {"stop": step.reason}
        assert isinstance(step, Switch), step
        default = None if step.default is None else [show(one) for one in step.default]
        return {"switch": step.on, "cases": {key: [show(one) for one in steps] for key, steps in step.cases.items()}, "default": default}

    return show


@pytest.fixture
def located(tree):
    """A function that shows errors as `(path, code)` pairs, a file's path written from the temporary folder."""

    def show(errors):
        return [(tuple(step.removeprefix(f"{tree}/") if isinstance(step, str) else step for step in error.path), error.code) for error in errors]

    return show
