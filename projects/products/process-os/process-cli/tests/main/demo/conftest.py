from pathlib import Path
import shutil

import pytest

REPO = Path(__file__).parents[7]  # sbx-sdlc-kit's root: upstream had this at [5], one layout shallower
FIXTURES = Path(__file__).parents[3] / "fixtures" / "main" / "demo"


@pytest.fixture
def tree(tmp_path):
    """The demo's input, copied from this suite's own fixtures, and a config that names this repository's
    `libraries/` (a verbatim copy of process-os's `definitions/`) by its full path."""
    shutil.copytree(FIXTURES / "hello-app", tmp_path / "records" / "hello-app")
    (tmp_path / "processos.yaml").write_text(f"definitions: {REPO / 'processos-workspace' / 'libraries'}\nruntime:\n  root: .\n", encoding="utf-8")
    return tmp_path
