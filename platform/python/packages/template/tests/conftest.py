"""Gives each test its cases. A test file `tests/<entity>/<action>/test_<scope>.py` is fed by
`fixtures/<entity>/<action>/<scope>.yaml`, one call per case, with the plan's id as the test id."""

from pathlib import Path

import pytest
import yaml
from process_kit.schema import Schema
from process_kit.template import Template
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
        (tmp_path / name).write_bytes(text if isinstance(text, bytes) else text.encode("utf-8"))
    return tmp_path


@pytest.fixture
def types(case):
    """A `Types` holding the case's `types`, each the text of a definition."""
    known = Types()
    for text in case.get("types", []):
        schema = Schema.loads(text)
        assert isinstance(schema, Schema), schema
        known.add(schema.name, schema)
    return known


@pytest.fixture
def template(case, tree):
    """The case's template, written into `tree/x` and loaded: `input` (else `string` for text data and `mapping`), its `pattern`, its
    `files` (paths from `files/`), and `after`, files written over the template's own once it is loaded."""
    data = case.get("data")
    ports = {"name": "x", "input": case.get("input", "mapping" if isinstance(data, dict) else "string")}
    if "pattern" in case:
        ports["pattern"] = case["pattern"]
    (tree / "x" / "files").mkdir(parents=True)
    (tree / "x" / "template.yaml").write_text(yaml.safe_dump(ports, sort_keys=False), encoding="utf-8")
    for name, content in case.get("files", {}).items():
        (tree / "x" / "files" / name).parent.mkdir(parents=True, exist_ok=True)
        (tree / "x" / "files" / name).write_bytes(content if isinstance(content, bytes) else content.encode("utf-8"))
    loaded = Template.load(tree / "x")
    assert isinstance(loaded, Template), loaded
    for name, content in case.get("after", {}).items():
        (tree / "x" / "files" / name).write_bytes(content)
    return loaded


class Held:
    """A `Source` over a mapping of path to text."""

    def __init__(self, files):
        self.files = files

    def read(self, path):
        return self.files.get(path)


@pytest.fixture
def source(case):
    return Held(case.get("source", {}))


@pytest.fixture
def located(tree):
    """A function that shows errors as `(path, code)` pairs, a file's path written from the temporary folder."""

    def show(errors):
        return [(tuple(step.removeprefix(f"{tree}/") if isinstance(step, str) else step for step in error.path), error.code) for error in errors]

    return show
