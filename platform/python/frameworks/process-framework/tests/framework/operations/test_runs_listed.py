def test_list_runs_gives_every_saved_run_as_a_run(case, tree, framework):
    ids = [framework.run(target["id"], target["inputs"]).id for target in case["targets"]]
    listed = [found.id for found in framework.list_runs()]
    assert sorted(listed) == sorted(ids)
