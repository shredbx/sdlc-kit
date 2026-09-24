import sys

import pytest


@pytest.fixture
def result(case, tree, monkeypatch, capsys):
    """As the top-level `result`, but with `sys.argv[0]` set to `<tree>/process-cli`, so `--version` looks for a
    stamp beside a program that is not really there — this folder never writes one that must be executable."""
    from process_cli import main

    monkeypatch.chdir(tree / case.get("cwd", "."))
    monkeypatch.delenv("PROCESS_CLI_CONFIG", raising=False)
    monkeypatch.setattr(sys, "argv", [str(tree / "process-cli"), *case["args"]])
    code = main(case["args"])
    out, err = capsys.readouterr()
    return code, out.splitlines(), err.splitlines()
