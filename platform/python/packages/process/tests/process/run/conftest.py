from pathlib import Path

import pytest
import yaml

FIXTURES = Path(__file__).parents[3] / "fixtures"


class Store:
    """A `RunStore` in memory that keeps a copy of the run each time it is saved, and what each node's scripts printed
    each time it is logged."""

    def __init__(self):
        self.saved = []
        self.logged = {}

    def new_id(self, target):
        return "r1"

    def save(self, run):
        self.saved.append(run.to_data())

    def load(self, run_id):
        from process_kit.process import Run

        return Run.from_data(self.saved[-1]) if self.saved else None

    def log(self, run_id, node, stdout, stderr):
        self.logged[node] = {"stdout": stdout, "stderr": stderr}


class Built:
    """A runner over the world of `run/world.yaml` and what the case adds, and what it needs to show the result."""

    def __init__(self, runner, main, store, events, tree):
        self.runner, self.main, self.store, self.events, self.tree = runner, main, store, events, tree

    def shown(self, run):
        trace = self.tree / "out" / "trace.txt"
        return {
            "status": run.status,
            "reason": run.reason,
            "done": run.done,
            "skipped": run.skipped,
            "context": run.context,
            "nested": run.nested,
            "output": run.output,
            "errors": [[list(error.path), error.code] for error in run.errors],
            "events": [[event.node, event.state] for event in self.events],
            "trace": trace.read_text().split() if trace.is_file() else [],
            "saves": [[data["status"], data["done"]] for data in self.store.saved],
            "logged": self.store.logged,
        }


@pytest.fixture
def built(case, tree):
    """The world, and what the case adds: `types`, `actions`, `checks` (an action's `check.sh`), `fail` (actions that exit 4 while
    `out/fail.flag` is there), `flag`, `process` (the main one, `x.yaml`) and `processes` (`p/<name>.yaml`)."""
    from process_kit.action import Action, Context, ShellExecutor
    from process_kit.process import Process, Runner
    from process_kit.schema import Schema
    from process_kit.types import Error, Types

    world = yaml.safe_load((FIXTURES / "process" / "run" / "world.yaml").read_text(encoding="utf-8"))
    types = Types()
    for data in [*world["types"], *case.get("types", [])]:
        schema = Schema.from_data(data)
        assert isinstance(schema, Schema), schema
        types.add(schema.name, schema)
    (tree / "out").mkdir()
    if case.get("flag"):
        (tree / "out" / "fail.flag").write_text("", encoding="utf-8")
    for name, spec in {**world["actions"], **case.get("actions", {})}.items():
        folder = tree / "actions" / name
        folder.mkdir(parents=True)
        ports = {key: spec[key] for key in ("input", "output") if key in spec}
        (folder / "action.yaml").write_text(yaml.safe_dump({"name": name, "type": "shell", **ports}, sort_keys=False), encoding="utf-8")
        script = f"echo {name} >> trace.txt\n" + ("[ -e fail.flag ] && exit 4\n" if name in case.get("fail", []) else "") + spec.get("script", "") + "\n"
        (folder / "action.sh").write_text(script, encoding="utf-8")
        if name in case.get("checks", {}):
            (folder / "check.sh").write_text(case["checks"][name] + "\n", encoding="utf-8")
    (tree / "p").mkdir()
    (tree / "x.yaml").write_text(case.get("process", world["main"]), encoding="utf-8")
    for name, text in case.get("processes", {}).items():
        (tree / "p" / f"{name}.yaml").write_text(text, encoding="utf-8")
    main = Process.load(tree / "x.yaml")
    assert isinstance(main, Process), main

    class Nodes:
        def node(self, id):
            if id == "x":
                return Process.load(tree / "x.yaml")
            if (tree / "p" / f"{id}.yaml").is_file():
                return Process.load(tree / "p" / f"{id}.yaml")
            if (tree / "actions" / id).is_dir():
                return Action.load(tree / "actions" / id)
            return [Error((), "unknown_id", id)]

    store, events = Store(), []
    runner = Runner(Nodes(), types, ShellExecutor(), store, Context(tree / "out"), events.append)
    return Built(runner, main, store, events, tree)
