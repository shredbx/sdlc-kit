def test_resume_goes_on_from_the_first_node_not_done(case, built):
    run = built.runner.start(built.main, case["inputs"], {"s": "1.0"}, "abc")
    for name in case.get("remove", []):
        (built.tree / name).unlink()
    resumed = built.runner.resume(run)
    shown = built.shown(resumed)
    assert {key: shown[key] for key in case["expect"]} == case["expect"]
