from process_kit.action import Action
from process_kit.process import Process


def test_resume_with_the_one_step_process_of_an_action_goes_on_from_where_it_stopped(case, built):
    process = Process.of_action(Action.load(built.tree / "actions" / "a1"))
    run = built.runner.start(process, case["inputs"], {"s": "1.0"}, "abc")
    for name in case.get("remove", []):
        (built.tree / name).unlink()
    resumed = built.runner.resume(run, process)
    shown = built.shown(resumed)
    assert {key: shown[key] for key in case["expect"]} == case["expect"]
