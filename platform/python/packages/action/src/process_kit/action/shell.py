"""The shell contract: check, pre, action and post, run with bash in the output folder."""

import os
import subprocess
import tempfile
from collections.abc import Mapping
from pathlib import Path
from typing import TYPE_CHECKING, Any, TypedDict

import yaml
from process_kit.schema import parse
from process_kit.types import Error

from .context import Context
from .result import Outcome, Result

if TYPE_CHECKING:
    from .action import Action

SKIP = 77  # the exit code of a check that skips, as in the Automake and Meson test harnesses


class _Captured(TypedDict):
    stdout: str
    stderr: str


class ShellExecutor:
    """Runs an action's scripts under the contract in `example-build-application.md` §3."""

    def execute(self, action: "Action", inputs: Mapping[str, Any], context: Context) -> Result:
        """A `context.output` that is not a folder is a wrong call and raises. A script that fails is a `failed` result.
        Every hook's standard output is read in full, then written back out, so it still reaches the terminal, just
        not while the script is still running; its standard error is only ever read. Both, across every hook that
        ran, come back on the `Result`."""
        if not context.output.is_dir():
            raise NotADirectoryError(f"{context.output} is not a folder: the caller makes the folder an action works in")
        with tempfile.TemporaryDirectory() as scratch:
            given, written = Path(scratch) / "inputs", Path(scratch) / "outputs"
            given.mkdir()
            written.mkdir()
            for name, value in inputs.items():
                (given / f"{name}.yaml").write_text(yaml.safe_dump(value, sort_keys=False), encoding="utf-8")
            env = (
                os.environ
                | dict(context.environment)
                | {
                    "PROCESS_OUTPUT": str(context.output),
                    "ACTION_HOME": str(action.home),
                    "ACTION_INPUTS": str(given),
                    "ACTION_OUTPUTS": str(written),
                    **_variables(inputs),
                }
            )
            out_parts: list[str] = []
            err_parts: list[str] = []

            def run(hook: str) -> subprocess.CompletedProcess[str] | None:
                """The hook's own result, or `None` when it ran longer than `action.timeout` (`0` means no limit) — the
                caller decides the outcome a timeout gives, since `check.sh`'s and the other hooks' own timeouts are not
                the same outcome kind."""
                try:
                    raw = subprocess.run(
                        ["bash", str(action.home / f"{hook}.sh")],
                        cwd=context.output,
                        env=env,
                        stdin=subprocess.DEVNULL,
                        capture_output=True,
                        timeout=action.timeout or None,
                    )
                except subprocess.TimeoutExpired:
                    return None
                out = raw.stdout.decode("utf-8", errors="replace")
                err = raw.stderr.decode("utf-8", errors="replace")
                out_parts.append(out)
                err_parts.append(err)
                os.write(1, raw.stdout)  # the raw file descriptor, not sys.stdout: --json's session redirects fd 1, not the Python object
                return subprocess.CompletedProcess(raw.args, raw.returncode, out, err)

            def captured() -> _Captured:
                return {"stdout": "".join(out_parts), "stderr": "".join(err_parts)}

            if (action.home / "check.sh").is_file():
                checked = run("check")
                if checked is None:
                    return Result(Outcome.stopped, reason=f"check.sh timed out after {action.timeout}s", **captured())
                if checked.returncode == SKIP:
                    return _read(action, written, Outcome.skipped, **captured())
                if checked.returncode != 0:
                    return Result(Outcome.stopped, reason=checked.stderr.strip() or f"check.sh exited {checked.returncode}", **captured())
            for hook in ("pre", "action", "post"):
                if not (action.home / f"{hook}.sh").is_file():
                    if hook == "action":
                        return Result(Outcome.failed, reason="the action has no action.sh", **captured())
                    continue
                done = run(hook)
                if done is None:
                    return Result(Outcome.failed, reason=f"{hook}.sh timed out after {action.timeout}s", **captured())
                if done.returncode != 0:
                    detail = f": {done.stderr.strip()}" if done.stderr.strip() else ""
                    return Result(Outcome.failed, reason=f"{hook}.sh exited {done.returncode}{detail}", **captured())
            return _read(action, written, Outcome.done, **captured())


def _variables(inputs: Mapping[str, Any]) -> dict[str, str]:
    """`INPUT_<NAME>_<FIELD>` for each top-level scalar field of a mapping input, and `INPUT_<NAME>` for a scalar input."""
    found = {}
    for name, value in inputs.items():
        fields = value.items() if isinstance(value, dict) else [(None, value)]
        for field, item in fields:
            if not isinstance(item, (dict, list)):
                parts = ("input", name) if field is None else ("input", name, field)
                found["_".join(str(part).upper().replace("-", "_") for part in parts)] = _scalar(item)
    return found


def _scalar(value: Any) -> str:
    if value is None:
        return ""
    return str(value).lower() if isinstance(value, bool) else str(value)


def _read(action: "Action", written: Path, outcome: Outcome, stdout: str, stderr: str) -> Result:
    """The outputs the script wrote, as data. A file that is not YAML fails the action."""
    outputs: dict[str, Any] = {}
    errors: list[Error] = []
    for name in action.output:
        file = written / f"{name}.yaml"
        if file.is_file():
            data, problems = parse(file.read_text(encoding="utf-8"))
            errors.extend(Error((name, *problem.path), problem.code, problem.message) for problem in problems)
            outputs[name] = data
    if errors:
        return Result(Outcome.failed, reason="the outputs are wrong", errors=tuple(errors), stdout=stdout, stderr=stderr)
    return Result(outcome, outputs, stdout=stdout, stderr=stderr)
