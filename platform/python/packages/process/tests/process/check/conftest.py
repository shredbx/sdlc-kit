from pathlib import Path

import pytest
import yaml

FIXTURES = Path(__file__).parents[3] / "fixtures"


@pytest.fixture
def checked(case, tree):
    """`check` on the case's `process`, written as `x.yaml`, in the world of `world.yaml` with what the case adds to it."""
    from pathlib import Path
    from types import MappingProxyType

    from process_kit.action import Action
    from process_kit.process import Process, check
    from process_kit.schema import Schema
    from process_kit.types import Error, Types

    world = yaml.safe_load((FIXTURES / "process" / "check" / "world.yaml").read_text(encoding="utf-8"))
    types = Types()
    for data in [*world["types"], *case.get("types", [])]:
        schema = Schema.from_data(data)
        assert isinstance(schema, Schema), schema
        types.add(schema.name, schema)
    actions = {**world["actions"], **case.get("actions", {})}
    texts = {**world["processes"], **case.get("processes", {})}
    (tree / "p").mkdir()
    (tree / "x.yaml").write_text(case["process"], encoding="utf-8")
    main = Process.load(tree / "x.yaml")
    assert isinstance(main, Process), main
    nodes = {"x": main}
    for name, text in texts.items():
        (tree / "p" / f"{name}.yaml").write_text(text, encoding="utf-8")
        loaded = Process.load(tree / "p" / f"{name}.yaml")
        assert isinstance(loaded, Process), loaded
        nodes[name] = loaded
    for name, ports in actions.items():
        nodes[name] = Action(name, "shell", None, MappingProxyType(ports.get("input", {})), MappingProxyType(ports.get("output", {})), 60, Path("/x") / name)

    class Nodes:
        def node(self, id):
            return nodes.get(id) or [Error((), "unknown_id", f"there is no node {id!r}")]

    return check(main, Nodes(), types)
