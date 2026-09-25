import yaml


def test_resume_after_a_definition_changed_gives_the_errors_and_leaves_the_run_as_it_was(case, tree, framework, located):
    first = framework.run(case["target"], case["inputs"])
    assert first.status == "failed"
    for name, text in case.get("then_files", {}).items():
        (tree / name).write_text(text, encoding="utf-8")
    for name in case.get("then_remove", []):
        (tree / name).unlink()
    errors = framework.resume(first.id)
    assert isinstance(errors, list)
    assert located(errors) == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
    assert yaml.safe_load((tree / "runs" / first.id / "run.yaml").read_text(encoding="utf-8"))["status"] == "failed"
