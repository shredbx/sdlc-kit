"""The command line's side of the framework's `Host`, and where everything it says goes: text lines, or one JSON document."""

import json
import os
import sys
from collections.abc import Iterator, Sequence
from contextlib import contextmanager
from pathlib import Path
from typing import Any

from process_framework import Error, Event


def shown(file: str | Path) -> str:
    """A file as the user sees it: relative to the folder the command runs in."""
    return os.path.relpath(file)


class Printer:
    """The command line's `Host`. `command` is the program that is running, which `main` sets when it is `process-cli`, so an action can call it
    back. `report` says what became of each node of a process as it ends. With `json`, the answer of a command is one JSON document on standard
    output and nothing else: `session` keeps what scripts write there out of it."""

    command: Path | None = None

    def __init__(self, json: bool = False) -> None:
        self.json = json
        self.events: list[dict[str, Any]] = []
        self._answer: int | None = None

    @contextmanager
    def session(self) -> Iterator[None]:
        """In JSON mode, point the program's standard output at its standard error while it works, and keep the real one for the answer. In text mode, nothing."""
        if not self.json:
            yield
            return
        sys.stdout.flush()
        self._answer = os.dup(1)
        os.dup2(2, 1)
        try:
            yield
        finally:
            os.dup2(self._answer, 1)
            os.close(self._answer)
            self._answer = None

    def report(self, event: Event) -> None:
        """One line for each node that ends: indented by how deep it is, then its own id, what became of it and why. A node that starts says nothing.
        In JSON mode, every event is kept, for the answer."""
        if self.json:
            self.events.append({"node": event.node, "state": event.state, "reason": event.reason})
        elif event.state != "started":
            reason = f" — {event.reason}" if event.reason else ""
            print(f"{'  ' * (event.node.count('/') + 1)}{event.node.rpartition('/')[2]}: {event.state}{reason}")

    def result(self, lines: Sequence[str], data: dict[str, Any], errors: Sequence[Error] = (), ok: bool | None = None) -> None:
        """The answer of a command. Text: the `lines` on standard output, then the errors, one to a line, on standard error. JSON: one line on standard
        output, `{"ok", **data, "errors"}`. `ok` is that there are no errors unless it is given."""
        if not self.json:
            for line in lines:
                print(line)
            self._print(errors)
            return
        answer = {"ok": not errors if ok is None else ok, **data, "errors": [{"path": list(error.path), "code": error.code, "message": error.message} for error in errors]}
        assert self._answer is not None, "the answer is given inside a session"
        os.write(self._answer, (json.dumps(answer, ensure_ascii=False) + "\n").encode("utf-8"))

    def errors(self, errors: Sequence[Error]) -> None:
        """The answer of a command that found only errors."""
        self.result([], {}, errors)

    def message(self, text: str) -> None:
        print(text, file=sys.stderr)

    def note(self, text: str) -> None:
        """A line of advice on standard error, in text mode only."""
        if not self.json:
            self.message(text)

    def _print(self, errors: Sequence[Error]) -> None:
        """One line each: the file, the path in it, the code and the message. The first step of a path is the file, and an error with no path has no file."""
        for error in errors:
            file, *rest = error.path or ("",)
            where = "".join(f"[{step}]" if isinstance(step, int) else f".{step}" for step in rest).removeprefix(".")
            self.message(": ".join(part for part in (shown(file) if file else "", where, error.code, error.message) if part))
