from process_framework import create_workspace, initialize


def test_a_created_workspace_is_one_initialize_accepts_and_check_finds_nothing_wrong_in(case, tree, host):
    create_workspace(tree)
    framework = initialize(tree / "processos.yaml", host)
    assert not isinstance(framework, list), framework
    assert (framework.check(), framework.list("action")) == ([], [])
