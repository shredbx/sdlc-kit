from process_framework import Catalog


def test_a_change_to_what_a_run_depends_on_changes_the_digest_and_no_other_does(case, tree):
    before = Catalog.open(tree / "defs")
    first = before.digest(before.node("sdlc.p"))
    for name, text in case["change"].items():
        (tree / name).parent.mkdir(parents=True, exist_ok=True)
        (tree / name).write_text(text, encoding="utf-8")
    after = Catalog.open(tree / "defs")
    assert (after.digest(after.node("sdlc.p")) != first) is case["differs"]
