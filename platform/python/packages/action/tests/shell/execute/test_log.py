from process_kit.action import Context, ShellExecutor
from process_kit.types import Types


def test_the_result_carries_what_the_scripts_printed_and_the_terminal_still_gets_it(case, tree, built, capfd):
    (tree / "out").mkdir()
    result = built.run(case.get("inputs", {}), Context(tree / "out"), Types(), ShellExecutor())
    expect = case["expect"]
    assert (result.outcome.value, result.reason, result.stdout, result.stderr) == (
        expect["outcome"],
        expect.get("reason"),
        expect["stdout"],
        expect["stderr"],
    )
    assert capfd.readouterr().out == expect["stdout"]
