import json

import pytest


@pytest.fixture
def answers(case, tree, monkeypatch, capfd):
    """Runs `main` for each call in the case's `runs`, in the temporary folder, under `capfd`: what the descriptors saw. Before a call, its `before` is
    applied (a path to its new text, or to `null` to remove it), and `{run}` in its `args` is the id of the run the call before saved. Gives the exit
    code, standard output and standard error of each call."""
    from process_cli import main

    monkeypatch.chdir(tree)
    monkeypatch.delenv("PROCESS_CLI_CONFIG", raising=False)
    found, saved = [], ""
    for call in case["runs"]:
        for name, text in call.get("before", {}).items():
            if text is None:
                (tree / name).unlink()
            else:
                (tree / name).parent.mkdir(parents=True, exist_ok=True)
                (tree / name).write_text(text, encoding="utf-8")
        code = main([arg.replace("{run}", saved) for arg in call["args"]])
        out, err = capfd.readouterr()
        found.append((code, out, err))
        saved = json.loads(out.splitlines()[0]).get("run", {}).get("id", saved) if out.strip() else saved
    return found


@pytest.fixture
def holds(tree):
    """A function that says whether a value holds what was written for it: a mapping has the keys, a list has the items and no more, and `{tree}` in
    a string is the temporary folder."""

    def has(value, expect):
        if isinstance(expect, dict):
            return isinstance(value, dict) and all(key in value and has(value[key], one) for key, one in expect.items())
        if isinstance(expect, list):
            return isinstance(value, list) and len(value) == len(expect) and all(has(item, one) for item, one in zip(value, expect))
        if isinstance(expect, str):
            return value == expect.replace("{tree}", str(tree))
        return value == expect

    return has
