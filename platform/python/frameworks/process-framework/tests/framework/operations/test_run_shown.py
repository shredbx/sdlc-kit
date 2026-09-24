def test_show_run_gives_the_run_kept_under_that_id(case, tree, framework):
    run = framework.run(case["target"], case["inputs"])
    shown = framework.show_run(run.id)
    assert (shown.id, shown.status, shown.done) == (run.id, case["expect"]["status"], case["expect"]["done"])
