from process_framework import Catalog
from process_kit.process import Call, Process


def test_node_gives_the_process_with_the_id_and_its_names_in_full(case, tree):
    process = Catalog.open(tree / "defs").node(case["ident"])
    assert isinstance(process, Process)
    assert (process.id, dict(process.input), [step.id for step in process.steps if isinstance(step, Call)]) == (
        case["ident"],
        case["expect"]["input"],
        case["expect"]["steps"],
    )
