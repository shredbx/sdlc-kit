from types import SimpleNamespace

from process_framework import ProcessFramework, initialize


def test_initialize_gives_a_framework_holding_the_config_the_catalog_and_the_host(case, tree):
    host = SimpleNamespace(command=None, report=lambda event: None)
    framework = initialize(tree / case["file"], host)
    assert isinstance(framework, ProcessFramework)
    assert framework.host is host
    assert framework.config.definitions == tree / case["expect"]["definitions"]
    assert list(framework.catalog.scopes) == case["expect"]["scopes"]
