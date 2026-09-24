def test_show_run_gives_the_one_run_id_asked_for_not_just_any_run(case, tree, framework):
    made = [framework.run(target["id"], target["inputs"]) for target in case["targets"]]
    for run in made:
        assert framework.show_run(run.id).target == run.target
