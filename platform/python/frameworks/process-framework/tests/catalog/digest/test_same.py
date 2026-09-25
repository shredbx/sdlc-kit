import shutil

from process_framework import Catalog


def test_two_catalogs_on_the_same_files_give_one_digest(case, tree):
    shutil.copytree(tree / "defs", tree / "copy")
    first, second = Catalog.open(tree / "defs"), Catalog.open(tree / "copy")
    digest = first.digest(first.node(case["ident"]))
    assert digest == second.digest(second.node(case["ident"]))
    assert len(digest) == 64
