import yaml


def test_run_of_an_action_gives_the_run_saves_it_and_tells_the_host_nothing(case, tree, framework):
    made = framework.run(case.get("action", "sdlc.greet"), {**case.get("inputs", {"msg": "hi"}), **case.get("values", {})}, case.get("typed"))
    shown = {
        "status": made.status,
        "reason": made.reason,
        "done": made.done,
        "skipped": made.skipped,
        "output": made.output,
        "errors": [{"path": list(error.path), "code": error.code} for error in made.errors],
    }
    assert shown == {"reason": None, "skipped": [], "output": {}, "errors": [], **case["expect"]["run"]}
    saved = sorted((tree / "runs").glob("*/run.yaml"))
    assert [path.parent.name for path in saved] == [made.id]
    loaded = yaml.safe_load(saved[0].read_text(encoding="utf-8"))
    assert {key: loaded.get(key) for key in case["expect"]["saved"]} == case["expect"]["saved"]
    assert {path: (tree / path).read_text() if (tree / path).is_file() else None for path in case["expect"].get("files", {})} == case["expect"].get("files", {})
    assert framework.host.events == []
