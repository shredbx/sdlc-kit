from pathlib import Path
import shutil

import pytest

REPO = Path(__file__).parents[7]  # sbx-sdlc-kit's root: upstream had this at [5], one layout shallower
FIXTURES = Path(__file__).parents[3] / "fixtures" / "main" / "demo-libraries"


@pytest.fixture
def tree(tmp_path):
    """The demo's input, copied from this suite's own fixtures, and a config whose default folder holds only
    `main`, with `sdlc` and `std` given as libraries pointing at this repository's `libraries/sdlc` and
    `libraries/std` (verbatim copies of process-os's `definitions/`)."""
    shutil.copytree(FIXTURES / "hello-app", tmp_path / "records" / "hello-app")
    (tmp_path / "definitions" / "main").mkdir(parents=True)
    (tmp_path / "definitions" / "main" / "scope.yaml").write_text("name: main\nversion: 0.1.0\n", encoding="utf-8")
    config = (
        "definitions: definitions\n"
        "runtime:\n  root: .\n"
        "libraries:\n"
        f"  - {{name: sdlc, path: {REPO / 'processos-workspace' / 'libraries' / 'sdlc'}}}\n"
        f"  - {{name: std, path: {REPO / 'processos-workspace' / 'libraries' / 'std'}}}\n"
    )
    (tmp_path / "processos.yaml").write_text(config, encoding="utf-8")
    return tmp_path
