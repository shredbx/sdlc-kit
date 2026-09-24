from process_kit.action import Action
from process_kit.process import Process


def test_of_action_is_a_process_of_one_call_with_the_ports_and_home_of_the_action(case, tree):
    action = Action.load(tree / "a1")
    process = Process.of_action(action)
    shown = {
        "id": process.id,
        "input": dict(process.input),
        "requires": dict(process.requires),
        "output": dict(process.output),
        "steps": [f"{type(step).__name__}:{step.id}" for step in process.steps],
    }
    assert shown == case["expect"]
    assert process.home == action.home
    assert process.description == action.description
