from process_framework import Catalog
from process_kit.config import Library


def test_a_scope_gives_the_same_digest_from_the_default_folder_or_a_library(case, tree):
    from_folder = Catalog.open(tree / "defs")
    library = Library(case["ident"].split(".")[0], tree / "lib" / case["ident"].split(".")[0], "readonly", None)
    from_library = Catalog.open(tree / "empty", [library], tree / "processos.yaml")
    assert isinstance(from_folder, Catalog)
    assert isinstance(from_library, Catalog)
    digest = from_folder.digest(from_folder.node(case["ident"]))
    assert digest == from_library.digest(from_library.node(case["ident"]))
    assert len(digest) == 64
