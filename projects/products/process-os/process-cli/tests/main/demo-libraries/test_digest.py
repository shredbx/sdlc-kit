from pathlib import Path

from process_kit.config import Library

from process_framework import Catalog

REPO = Path(__file__).parents[7]  # sbx-sdlc-kit's root: upstream had this at [5], one layout shallower


def test_the_digest_from_libraries_equals_the_digest_from_one_folder(case, tmp_path):
    from_folder = Catalog.open(REPO / "processos-workspace" / "libraries")
    (tmp_path / "empty").mkdir()
    libraries = [Library(name, REPO / "processos-workspace" / "libraries" / name, "readonly", None) for name in case["libraries"]]
    from_libraries = Catalog.open(tmp_path / "empty", libraries, tmp_path / "processos.yaml")
    assert isinstance(from_folder, Catalog)
    assert isinstance(from_libraries, Catalog)
    node = from_folder.node(case["ident"])
    assert not isinstance(node, list)
    digest = from_folder.digest(node)
    assert digest == from_libraries.digest(from_libraries.node(case["ident"]))
    assert len(digest) == 64
