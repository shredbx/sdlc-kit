"""The runtime folders the config names: the records a run is given, and the runs it leaves."""

from collections.abc import Callable, Mapping
from datetime import datetime
from pathlib import Path
from typing import Any

import yaml
from process_kit.filesystem import Folder
from process_kit.process import Run
from process_kit.schema import Schema, parse
from process_kit.types import BooleanType, Error, FloatType, IntegerType, Location, SequenceType, StringType, Types, wrong_type


class Records:
    """The records given to runs, in `runtime.records`. An input is a file named after it, in the folder of its instance."""

    def __init__(self, folder: Folder) -> None:
        self.folder = folder

    def file(self, name: str, folder: str) -> str | None:
        """`<records>/<folder>/<name>.yaml` as text, or `None` when `folder` leaves the records folder. A `name` or `folder` that is not
        text is a wrong call and raises."""
        if not isinstance(name, str) or not isinstance(folder, str):
            raise TypeError(f"an input is a name and a folder, both text, got {name!r} and {folder!r}")
        relative = f"{folder}/{name}.yaml"
        return str(self.folder.path(relative)) if self.folder.inside(relative) else None

    def read(self, name: str, folder: str) -> tuple[Any, list[Error]]:
        """`(data, errors)`: what `<records>/<folder>/<name>.yaml` holds, as `parse` gives it. A `folder` that leaves the records
        folder, a file that is not there, and a file that is not YAML are errors. A `name` or `folder` that is not text is a
        wrong call and raises."""
        file = self.file(name, folder)
        if file is None:
            return None, [Error((name,), "outside_root", f"{folder!r} is not inside the records folder")]
        text = self.folder.read(f"{folder}/{name}.yaml")
        if text is None:
            return None, [Error((file,), "missing", f"there is no record {name}.yaml in {folder!r}")]
        data, errors = parse(text)
        return data, [Error((file, *error.path), error.code, error.message) for error in errors]

    def gather(
        self, input: Mapping[str, str], requires: Mapping[str, str], records: Mapping[str, str], typed: Mapping[str, Any], types: Types
    ) -> tuple[dict[str, Any], list[Error]]:
        """`(gathered, errors)`: what a run starts with. `input` and `requires` map a name to the full name of its type; `records` maps
        an input's name to text, resolved in order: a file at that exact path, when one exists there, read verbatim; else an id, read as
        `<id>/<name>.yaml` under this `Records`'s own folder — the folder a caller once gave directly, now found the same way; else, when
        neither is there and the declared type is a string, an integer, a float or a boolean, the text itself, cast as that (the same
        casting `--set` always did). `typed` maps an input's name to an already-typed value instead — a real mapping, a real list, a real
        number — checked as it is, never cast. A name in `records` or `typed` that is not declared comes first; then each name declared,
        `input` then `requires`: `records`, when there is one, is trusted most, then `typed`. A `requires` not given directly is found at
        `<folder>/extension/<name>.yaml`, among the folders actually read by id for the names `records` did give (not a file given
        directly, and not a literal — neither one is a folder to search). `gathered` holds each one that is right. A `records` or `typed`
        that is not a mapping is a wrong call and raises."""
        if not isinstance(records, Mapping):
            raise TypeError(f"records must be a mapping of input name to text, got {records!r}")
        if not isinstance(typed, Mapping):
            raise TypeError(f"typed must be a mapping of input name to an already-typed value, got {typed!r}")
        declared = {**input, **requires}
        errors = [Error((name,), "unexpected", f"{name!r} is not an input of this run") for name in records if name not in declared]
        errors += [Error((name,), "unexpected", f"{name!r} is not an input of this run") for name in typed if name not in input]
        resolved: dict[str, tuple[Any, list[Error]]] = {}
        ids: dict[str, str] = {}
        for name, text in records.items():
            if name not in declared:
                continue
            kind = declared[name]
            if Path(text).is_file():
                file = Path(text)
                value, problems = parse(file.read_text(encoding="utf-8"))
                problems = [Error((str(file), *error.path), error.code, error.message) for error in problems]
                if not problems:
                    problems = [Error((str(file), *error.path), error.code, error.message) for error in types.validate(value, kind)]
            else:
                value, problems = self.read(name, text)
                if not problems:
                    located = self.file(name, text)
                    assert located is not None  # `read` reports a folder outside the records as an error, so it is inside here
                    problems = [Error((located, *error.path), error.code, error.message) for error in types.validate(value, kind)]
                if problems and len(problems) == 1 and problems[0].code == "missing" and _scalar(kind, types):
                    value, problems = cast_value(text, kind, types, (name,))
                    if not problems:
                        problems = types.validate(value, kind, (name,))
                else:
                    ids[name] = text  # a folder was genuinely read from, successfully or not — never a literal
            resolved[name] = (value, problems)
        gathered: dict[str, Any] = {}
        for name, kind in declared.items():
            if name in resolved:
                value, problems = resolved[name]
                if problems:
                    errors.extend(problems)
                else:
                    gathered[name] = value
                continue
            if name in input and name in typed:
                problems = types.validate(typed[name], kind, (name,))
                if problems:
                    errors.extend(problems)
                else:
                    gathered[name] = typed[name]
                continue
            if name in input:
                errors.append(Error((name,), "missing", f"{name!r} is required and was not given"))
                continue
            found = self._extensions(name, ids)
            if len(found) != 1:
                folders = ", ".join(dict.fromkeys(ids.values())) or "none"
                errors.append(
                    Error(
                        (name,),
                        "duplicate" if found else "missing",
                        f"extension/{name}.yaml is in " + (f"more than one input folder: {', '.join(found)}" if found else f"no input folder given: {folders}"),
                    )
                )
                continue
            folder = found[0]
            value, problems = self.read(name, folder)
            if not problems:
                located = self.file(name, folder)
                assert located is not None  # `read` reports a folder outside the records as an error, so it is inside here
                problems = [Error((located, *error.path), error.code, error.message) for error in types.validate(value, kind)]
            if problems:
                errors.extend(problems)
            else:
                gathered[name] = value
        return gathered, errors

    def _extensions(self, name: str, ids: Mapping[str, str]) -> list[str]:
        """The folders `<folder>/extension` of the ids given, each once, in order, that hold `<name>.yaml`."""
        folders = dict.fromkeys(f"{folder}/extension" for folder in ids.values())
        return [folder for folder in folders if self.file(name, folder) is not None and self.folder.path(f"{folder}/{name}.yaml").is_file()]


def _scalar(kind: str, types: Types) -> bool:
    """Whether `kind` resolves to a string, an integer, a float or a boolean — the only kinds `cast_value` can build from text."""
    found = types.get(kind)
    found = found.base if isinstance(found, Schema) else found
    return isinstance(found, (StringType, BooleanType, IntegerType, FloatType))


def cast_value(text: str, kind: str, types: Types, path: Location) -> tuple[Any, list[Error]]:
    """`(value, errors)`: `text` turned into what `kind` names. A string is kept as `text`, unturned; an integer or a float comes from
    digits, a boolean from `true`/`false`. A sequence whose `items` is one of those comes from `text` split on commas, each piece cast
    the same way, `[]` for empty text — `unsupported_type` when `items` is not one of those. Anything else `kind` names (a mapping, a
    schema based on one) cannot be built from text at all — `unsupported_type`, located at `path`."""
    found = types.get(kind)
    found = found.base if isinstance(found, Schema) else found
    if isinstance(found, StringType):
        return text, []
    if isinstance(found, BooleanType):
        if text in ("true", "false"):
            return text == "true", []
        return None, [wrong_type(path, "true or false", text)]
    if isinstance(found, IntegerType):
        try:
            return int(text), []
        except ValueError:
            return None, [wrong_type(path, "an integer", text)]
    if isinstance(found, FloatType):
        try:
            return float(text), []
        except ValueError:
            return None, [wrong_type(path, "a number", text)]
    if isinstance(found, SequenceType):
        item_kind = found.items
        item = types.get(item_kind) if item_kind else None
        item = item.base if isinstance(item, Schema) else item
        if item_kind is None or not isinstance(item, (StringType, BooleanType, IntegerType, FloatType)):
            return None, [Error(path, "unsupported_type", f"{kind!r} cannot be set from text: its items are not a string, an integer, a float or a boolean")]
        values: list[Any] = []
        errors: list[Error] = []
        for piece in text.split(",") if text else []:
            cast, problems = cast_value(piece, item_kind, types, path)
            values.append(cast)
            errors.extend(problems)
        return values, errors
    return None, [
        Error(
            path,
            "unsupported_type",
            f"{kind!r} cannot be set from text: it is a {type(found).__name__}, not a string, an integer, a float, a boolean or a sequence of one",
        )
    ]


class Runs:
    """The runs, one folder each in `runtime.runs`, holding a `run.yaml`."""

    def __init__(self, folder: Folder, now: Callable[[], datetime] = datetime.now) -> None:
        self.folder = folder
        self.now = now

    def new_id(self, target: str) -> str:
        """`yymmdd-hhmmss-<the last name of target>`, and with `-2`, `-3`… when a run of that name is already there."""
        if not isinstance(target, str):
            raise TypeError(f"a target is an id, text, got {target!r}")
        base = f"{self.now():%y%m%d-%H%M%S}-{target.rpartition('.')[2]}"
        found, count = base, 1
        while self.folder.exists(found):
            count += 1
            found = f"{base}-{count}"
        return found

    def save(self, run: Run) -> None:
        """Write `<run.id>/run.yaml`, `run.to_data()` as YAML in the order it has, replacing one that is there. A `run` that is not a
        `Run` is a wrong call and raises, and so does an id that leaves the folder."""
        if not isinstance(run, Run):
            raise TypeError(f"a run is a Run, got {run!r}")
        self.folder.write(f"{run.id}/run.yaml", yaml.safe_dump(run.to_data(), sort_keys=False))

    def load(self, run_id: str) -> Run | None:
        """The run kept under `run_id`, or `None` when there is none, or when the id leaves the folder. A file that is not a run raises
        `ValueError`, and a `run_id` that is not text raises `TypeError`."""
        if not isinstance(run_id, str):
            raise TypeError(f"a run id is text, got {run_id!r}")
        relative = f"{run_id}/run.yaml"
        text = self.folder.read(relative) if self.folder.inside(relative) else None
        if text is None:
            return None
        data, errors = parse(text)
        try:
            if errors:
                raise ValueError(errors[0].message)
            return Run.from_data(data)
        except (KeyError, TypeError, AttributeError, ValueError) as reason:
            raise ValueError(f"{relative} is not a run: {reason!r}") from reason

    def list(self) -> list[str]:
        """Every run's id, most recent first — each a folder directly under the runs folder that holds a `run.yaml`. A
        folder with none is left out, quietly. `[]` when the runs folder is not there yet."""
        if not self.folder.root.is_dir():
            return []
        return sorted(
            (found.name for found in self.folder.root.iterdir() if found.is_dir() and (found / "run.yaml").is_file()),
            reverse=True,
        )

    def delete(self, run_id: str) -> None:
        """Remove everything kept for run_id — its whole folder. Does nothing when there is none. A `run_id` that is
        not text is a wrong call and raises, and so does one that leaves the folder."""
        if not isinstance(run_id, str):
            raise TypeError(f"a run id is text, got {run_id!r}")
        self.folder.remove(run_id)

    def prune(self, keep: int) -> None:
        """Keep the `keep` most recent done runs — `list`'s own order, most recent first — and delete every done run
        older than that. A run that is `stopped` or `failed` is never touched, however many there are. One whose file
        cannot be read (`load` raising `ValueError`) is left alone too, quietly — not this method's problem to fix."""
        kept = 0
        for run_id in self.list():
            try:
                run = self.load(run_id)
            except ValueError:
                continue
            if run is None or run.status != "done":
                continue
            kept += 1
            if kept > keep:
                self.delete(run_id)

    def log(self, run_id: str, node: str, stdout: str, stderr: str) -> None:
        """Write what `node`'s scripts printed while `run_id` ran: `<run_id>/<node>/stdout.log` and `stderr.log`, replacing
        ones that are there. A stream with nothing printed writes no file at all. A `run_id`, `node`, `stdout` or `stderr`
        that is not text is a wrong call and raises, and so does a `run_id` or `node` that leaves the folder."""
        if not all(isinstance(value, str) for value in (run_id, node, stdout, stderr)):
            raise TypeError(f"a run id, a node, stdout and stderr are all text, got {run_id!r}, {node!r}, {stdout!r}, {stderr!r}")
        self.folder.path(f"{run_id}/{node}")  # raises for a run_id or node that leaves the folder, even when nothing is written below
        if stdout:
            self.folder.write(f"{run_id}/{node}/stdout.log", stdout)
        if stderr:
            self.folder.write(f"{run_id}/{node}/stderr.log", stderr)
