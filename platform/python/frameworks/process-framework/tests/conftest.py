"""Gives each test its cases. A test file `tests/<entity>/<action>/test_<scope>.py` is fed by
`fixtures/<entity>/<action>/<scope>.yaml`, one call per case, with the plan's id as the test id."""

from pathlib import Path
from types import SimpleNamespace

import pytest
import yaml
from process_framework import Catalog, initialize

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
    for name in case.get("without", []):
        (tmp_path / name).unlink()
    return tmp_path


@pytest.fixture
def reached(case, tree):
    """What `Catalog.types` gives for the case's `references` (`where` and `name`), written in its `file`."""
    references = [(tuple(one["where"]), one["name"]) for one in case["references"]]
    return Catalog.open(tree / "defs").types(references, tree / case.get("file", "defs/sdlc/action/a/action.yaml"))


@pytest.fixture
def host(case, tree):
    """A host that keeps every event it is told, in `events`. Its `command` is the case's file of that name, made executable, when the case has one."""
    command = tree / case["command"] if "command" in case else None
    if command is not None:
        command.chmod(0o755)
    events = []
    return SimpleNamespace(command=command, report=events.append, events=events)


@pytest.fixture
def framework(tree, host):
    """The framework for the config the case writes as `processos.yaml`."""
    made = initialize(tree / "processos.yaml", host)
    assert not isinstance(made, list), made
    return made


@pytest.fixture
def located(tree):
    """A function that shows errors as `(path, code)` pairs, a file's path written from the temporary folder."""

    def show(errors):
        return [(tuple(step.removeprefix(f"{tree}/") if isinstance(step, str) else step for step in error.path), error.code) for error in errors]

    return show
