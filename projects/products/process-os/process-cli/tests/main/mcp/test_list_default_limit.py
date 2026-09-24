"""process-os.list's own limit defaults to 100 when the caller gives none (decision 105, task 09) — the
one behaviour genuinely new to this tool, not just mirrored from the CLI's own list (whose default stays
unbounded, unchanged). Needs more than 100 real definitions to prove anything, so this is its own,
non-case-driven test, not fixture data: 105 hand-written entries would be unreadable."""

from types import SimpleNamespace

from process_framework import initialize

from process_cli.mcp import LIST_DEFAULT_LIMIT, _list


def test_list_caps_at_100_when_no_limit_is_given(tmp_path):
    (tmp_path / "defs" / "sdlc" / "type").mkdir(parents=True)
    (tmp_path / "processos.yaml").write_text("definitions: defs\nruntime:\n  root: .\n", encoding="utf-8")
    (tmp_path / "defs" / "sdlc" / "scope.yaml").write_text("name: sdlc\nversion: 0.1.0\n", encoding="utf-8")
    count = LIST_DEFAULT_LIMIT + 5
    for index in range(count):
        (tmp_path / "defs" / "sdlc" / "type" / f"t{index:03}.yaml").write_text(f"name: t{index:03}\ntype: string\n", encoding="utf-8")

    framework = initialize(tmp_path / "processos.yaml", SimpleNamespace(command=None, report=lambda event: None))
    assert not isinstance(framework, list), framework
    assert len(framework.list("type")) == count  # the framework method itself stays unbounded without a limit

    result = _list(framework, {"kind": "type"})
    assert len(result.content[0].text.splitlines()) == LIST_DEFAULT_LIMIT
    assert result.structured_content["entries"][0]["id"] == "sdlc.t000"
