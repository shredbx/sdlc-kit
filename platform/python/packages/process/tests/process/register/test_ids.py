from process_kit.action import register as register_actions
from process_kit.process import register
from process_kit.types import Types


def test_register_adds_exactly_the_ids_in_the_case_and_leaves_nothing_dangling(case):
    types = Types()
    assert register_actions(types) == []
    assert register(types) == []
    assert sorted(name for name in types.known if name.startswith(case["prefix"])) == sorted(case["expect"]["ids"])
    assert types.check() == []
