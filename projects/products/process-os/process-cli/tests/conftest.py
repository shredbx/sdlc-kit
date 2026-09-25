"""Gives each test its cases. A test file `tests/<entity>/<action>/test_<scope>.py` is fed by
`fixtures/<entity>/<action>/<scope>.yaml`, one call per case, with the plan's id as the test id."""

import os
import sys
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
    for name in case.get("without", []):
        (tmp_path / name).unlink()
    return tmp_path


@pytest.fixture
def result(case, tree, monkeypatch, capsys):
    """Runs `main` as the case says, in the case's folder `cwd`, and gives its exit code and the lines it printed."""
    from process_cli import main

    monkeypatch.chdir(tree / case.get("cwd", "."))
    monkeypatch.delenv("PROCESS_CLI_CONFIG", raising=False)
    for name, value in case.get("env", {}).items():
        monkeypatch.setenv(name, str(tree / value))
    code = main(case["args"])
    out, err = capsys.readouterr()
    return code, out.splitlines(), err.splitlines()


@pytest.fixture
def results(case, tree, monkeypatch, capsys):
    """Runs `main` for each call in the case's `runs`, in the case's tree, with this environment's `bin` first on the `PATH` so that an action
    that calls `process-cli` finds this one. Before a call, its `before` is applied: a path to its new text, or to `null` to remove the file.
    Gives the exit code and the lines printed, on standard output and on standard error, of each call."""
    from process_cli import main

    monkeypatch.chdir(tree)
    monkeypatch.delenv("PROCESS_CLI_CONFIG", raising=False)
    monkeypatch.setenv("PATH", f"{Path(sys.executable).parent}{os.pathsep}{os.environ['PATH']}")
    found = []
    for call in case["runs"]:
        for name, text in call.get("before", {}).items():
            if text is None:
                (tree / name).unlink()
            else:
                (tree / name).parent.mkdir(parents=True, exist_ok=True)
                (tree / name).write_text(text, encoding="utf-8")
        code = main(call["args"])
        out, err = capsys.readouterr()
        found.append((code, out.splitlines(), err.splitlines()))
    return found
