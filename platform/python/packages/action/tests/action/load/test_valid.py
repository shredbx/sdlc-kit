from process_kit.action import Action


def test_load_gives_the_action_the_case_describes(case, tree):
    action = Action.load(tree / case["folder"], case.get("namespace", ""))
    assert isinstance(action, Action)
    expect = case["expect"]
    assert (action.id, action.kind, action.description, action.timeout) == (expect["id"], "shell", expect["description"], expect["timeout"])
    assert (list(action.input.items()), list(action.output.items())) == (list(expect["input"].items()), list(expect["output"].items()))
    assert action.home == tree / case["folder"]
