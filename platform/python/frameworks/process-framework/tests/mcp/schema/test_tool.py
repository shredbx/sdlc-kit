from pathlib import Path

from process_framework import tool_of
from process_kit.action import Action
from process_kit.process import Process


def test_tool_of_names_the_node_and_leaves_out_a_description_it_does_not_have(case, types):
    common = {"id": case["node_id"], "description": case.get("description"), "input": case["input"], "home": Path(".")}
    node = Action(**common, kind="shell", output={}, timeout=60) if case["node"] == "action" else Process(**common, requires={}, output={}, steps=())
    assert tool_of(node, types) == case["expect"]
