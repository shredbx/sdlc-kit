def test_start_gives_the_run_the_case_expects(case, built):
    run = built.runner.start(built.main, case["inputs"], {"s": "1.0"}, "abc")
    shown = built.shown(run)
    assert {key: shown[key] for key in case["expect"]} == case["expect"]
    assert (run.id, run.target, run.scopes, run.digest) == ("r1", "x", {"s": "1.0"}, "abc")
