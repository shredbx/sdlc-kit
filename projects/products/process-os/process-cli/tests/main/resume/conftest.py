import pytest


@pytest.fixture
def run_twice(case, tree, monkeypatch, capsys):
    """Runs `main` with the case's `first` arguments, makes the case's change (`then_remove`, `then_files`), then runs it with `args`, where `{run}`
    is the id of the run the first call saved. Gives the exit code and the lines of the second call."""
    from process_cli import main

    monkeypatch.chdir(tree)
    monkeypatch.delenv("PROCESS_CLI_CONFIG", raising=False)
    main(case["first"])
    capsys.readouterr()
    for name in case.get("then_remove", []):
        (tree / name).unlink()
    for name, text in case.get("then_files", {}).items():
        (tree / name).write_text(text, encoding="utf-8")
    run_id = sorted(path.name for path in (tree / "runs").iterdir())[0]
    code = main([arg.replace("{run}", run_id) for arg in case["args"]])
    out, err = capsys.readouterr()
    return code, out.splitlines(), err.splitlines()
