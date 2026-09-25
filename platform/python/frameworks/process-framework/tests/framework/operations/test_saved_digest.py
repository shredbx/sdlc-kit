import yaml


def test_the_saved_run_holds_the_digest_of_what_it_ran(case, tree, framework):
    made = framework.run(case["target"], case["inputs"])
    saved = yaml.safe_load((tree / "runs" / made.id / "run.yaml").read_text(encoding="utf-8"))
    assert saved["digest"] == framework.catalog.digest(framework.catalog.node(case["target"]))
