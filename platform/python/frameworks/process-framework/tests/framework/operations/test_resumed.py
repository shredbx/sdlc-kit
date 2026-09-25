def test_resume_goes_on_with_the_same_run_from_the_first_node_not_done(case, tree, framework):
    first = framework.run(case["target"], case["inputs"])
    for name in case.get("then_remove", []):
        (tree / name).unlink()
    resumed = framework.resume(first.id)
    shown = {
        "same_id": resumed.id == first.id,
        "status": resumed.status,
        "done": resumed.done,
        "output": resumed.output,
        "trace": (tree / "output" / "trace.txt").read_text().split(),
        "runs": len(list((tree / "runs").glob("*/run.yaml"))),
    }
    assert shown == case["expect"]
