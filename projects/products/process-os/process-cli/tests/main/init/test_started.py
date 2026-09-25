def test_a_new_repository_starts_with_init_and_runs_the_action_written_in_it(case, tree, results):
    expects = [call["expect"] for call in case["runs"]]
    assert [(code, out, len(err)) for code, out, err in results] == [(one["code"], one["stdout"], len(one["starts"])) for one in expects]
    assert len(list((tree / "processos-workspace" / "runs").glob("*/run.yaml"))) == case["runs_saved"]
