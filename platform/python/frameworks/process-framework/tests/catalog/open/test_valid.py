from process_framework import Catalog
from process_kit.config import Library


def test_open_gives_the_scopes_in_the_case_in_order(case, tree):
    libraries = [Library(one["name"], tree / one["path"], one.get("access", "readonly"), one.get("version")) for one in case.get("libraries", [])]
    catalog = Catalog.open(tree / "defs", libraries, tree / "processos.yaml")
    assert isinstance(catalog, Catalog)
    assert catalog.root == tree / "defs"
    shown = [(name, scope.name, scope.version, scope.description, list(scope.uses), scope.folder, scope.access) for name, scope in catalog.scopes.items()]
    assert shown == [
        (
            one["name"],
            one["name"],
            one["version"],
            one.get("description"),
            one.get("uses", []),
            tree / one.get("folder", f"defs/{one['name']}"),
            one.get("access", "readwrite"),
        )
        for one in case["expect"]
    ]
