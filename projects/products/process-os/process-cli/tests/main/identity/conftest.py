import sys

import pytest

from process_cli.main import PROGRAM_ENVIRONMENT


@pytest.fixture
def result(case, tree, monkeypatch, capsys):
    """Runs `main` with `sys.argv[0]` set to `<tree>/<argv0>` (`not-process-cli` by default: a name plainly not
    `process-cli`, and not really an executable file, so the name-based check alone would refuse a shim, and there
    is nothing there for `--config`'s fourth step to find unless the case makes one). `$PROCESS_CLI_PROGRAM` is set
    to the case's `program`, when it gives one. A case's `symlink` (`{name: target}`) is made a real symlink first,
    so `argv0` can point through it."""
    from process_cli import main

    for name, target in case.get("symlink", {}).items():
        (tree / name).parent.mkdir(parents=True, exist_ok=True)
        (tree / target).parent.mkdir(parents=True, exist_ok=True)
        (tree / name).symlink_to(tree / target)
    monkeypatch.chdir(tree / case.get("cwd", "."))
    monkeypatch.delenv("PROCESS_CLI_CONFIG", raising=False)
    monkeypatch.delenv(PROGRAM_ENVIRONMENT, raising=False)
    monkeypatch.setattr(sys, "argv", [str(tree / case.get("argv0", "not-process-cli")), *case["args"]])
    if "program" in case:
        launcher = tree / case["program"]
        launcher.chmod(0o755)
        monkeypatch.setenv(PROGRAM_ENVIRONMENT, str(launcher))
    code = main(case["args"])
    out, err = capsys.readouterr()
    return code, out.splitlines(), err.splitlines()
