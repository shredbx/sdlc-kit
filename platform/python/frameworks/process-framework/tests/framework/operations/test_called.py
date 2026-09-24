def test_call_serves_a_tools_call_through_run_with_typed_arguments(case, tree, framework):
    made = framework.call(case["target"], case["arguments"])
    shown = {"status": made.status, "done": made.done, "output": made.output}
    assert shown == case["expect"]
