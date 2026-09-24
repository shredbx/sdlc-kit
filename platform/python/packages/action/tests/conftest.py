"""Gives each test its cases. A test file `tests/<entity>/<action>/test_<scope>.py` is fed by
`fixtures/<entity>/<action>/<scope>.yaml`, one call per case, with the plan's id as the test id."""

from pathlib import Path

import pytest
import yaml
from process_kit.action import Action, Context, ShellExecutor
from process_kit.types import Types

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
def built(case, tree):
    """The case's action, written into `tree/a` and loaded. `scripts` are its `<name>.sh`, `input` and `output` its ports,
    and `assets` its own files. A case with no `action` script gets `echo ran > ran.txt`."""
    folder = tree / "a"
    ports = {key: case[key] for key in ("input", "output", "timeout") if key in case}
    (folder / "assets").mkdir(parents=True)
    (folder / "action.yaml").write_text(yaml.safe_dump({"name": "a", "type": "shell", **ports}, sort_keys=False), encoding="utf-8")
    for name, body in {"action": "echo ran > ran.txt", **case.get("scripts", {})}.items():
        (folder / f"{name}.sh").write_text(body + "\n", encoding="utf-8")
    for name, text in case.get("assets", {}).items():
        (folder / "assets" / name).write_text(text, encoding="utf-8")
    action = Action.load(folder, case.get("namespace", ""))
    assert isinstance(action, Action), action
    return action


@pytest.fixture
def ran(case, tree, built):
    """The case's action, run in the folder `tree/out`, and what came back, shown as data."""
    (tree / "out").mkdir()
    result = built.run(case.get("inputs", {}), Context(tree / "out", case.get("environment", {})), Types(), ShellExecutor())
    files = case["expect"].get("files", {})
    return {
        "outcome": result.outcome.value,
        "reason": result.reason,
        "outputs": dict(result.outputs),
        "errors": [{"path": list(error.path), "code": error.code} for error in result.errors],
        "files": {path: (tree / path).read_text() if (tree / path).is_file() else None for path in files},
    }


@pytest.fixture
def expected(case):
    """What the case expects, with what it leaves out taken as nothing."""
    return {"reason": None, "outputs": {}, "errors": [], "files": {}, **case["expect"]}
