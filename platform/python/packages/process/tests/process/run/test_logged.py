def test_a_started_run_logs_what_each_node_printed_by_its_own_key(case, built):
    run = built.runner.start(built.main, case["inputs"], {}, "abc")
    shown = built.shown(run)
    assert {key: shown[key] for key in case["expect"]} == case["expect"]
