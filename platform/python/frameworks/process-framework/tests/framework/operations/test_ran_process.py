import yaml


def test_run_of_a_process_gives_the_run_tells_the_host_each_node_and_saves_it(case, tree, framework):
    made = framework.run(case["target"], case["inputs"])
    shown = {"status": made.status, "reason": made.reason, "done": made.done, "skipped": made.skipped, "output": made.output}
    assert shown == {"reason": None, "skipped": [], "output": {}, **case["expect"]["run"]}
    assert [[event.node, event.state] for event in framework.host.events] == case["expect"]["events"]
    saved = yaml.safe_load((tree / "runs" / made.id / "run.yaml").read_text(encoding="utf-8"))
    assert {key: saved.get(key) for key in case["expect"]["saved"]} == case["expect"]["saved"]
