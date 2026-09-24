"""The one way in: `initialize`, and the search an application may use to find the config."""

import os
from pathlib import Path
from typing import Any

from process_kit.action import ShellExecutor
from process_kit.config import Config
from process_kit.filesystem import Folder
from process_kit.schema import parse
from process_kit.types import Error

from .catalog import Catalog, check_scope_name
from .framework import ProcessFramework
from .host import Host
from .runtime import Records, Runs

CONFIG_FILE = "processos.yaml"
BUILD_FILE = "process-cli.build"


def read_build(file: str | Path) -> dict[str, Any] | None:
    """What a `process-cli.build` stamp holds — `version`, `commit`, `date`, `digest`, as an application chooses to
    read them — or `None` when `file` is not there, or does not parse as a mapping. Nothing here writes one."""
    file = Path(file)
    if not file.is_file():
        return None
    data, errors = parse(file.read_text(encoding="utf-8"))
    return data if not errors and isinstance(data, dict) else None


def find_config(start: str | Path) -> Path | None:
    """The first `processos.yaml` that is a file, in the folder `start` or in one above it, or `None`.
    A `start` that is not a folder is a wrong call and raises `NotADirectoryError`."""
    folder = Path(os.path.abspath(start))
    if not folder.is_dir():
        raise NotADirectoryError(f"{folder} is not a folder")
    for one in (folder, *folder.parents):
        if (one / CONFIG_FILE).is_file():
            return one / CONFIG_FILE
    return None


def initialize(config_file: str | Path, host: Host) -> ProcessFramework | list[Error]:
    """The framework for the config in `config_file`, or the errors that stopped it: those of the config, or of a
    `scope.yaml`. A file that does not exist raises, as it does in `Config.load`."""
    config = Config.load(config_file)
    if isinstance(config, list):
        return config
    catalog = Catalog.open(config.definitions, config.libraries, config.file)
    if isinstance(catalog, list):
        return catalog
    runtime = config.runtime
    return ProcessFramework(config, catalog, Folder(runtime.output), Records(Folder(runtime.records)), Runs(Folder(runtime.runs)), ShellExecutor(), host)


_CONFIG = """\
# The config of this workspace: where its definitions are, and where runs work. process-cli finds it by
# looking upward from the current folder. Paths are relative to this file.
version: 0.1.0                    # the shape of this file; changes only when process-cli's own config format does
definitions: processos-workspace/definitions   # a folder of scopes; every folder in it is one
runtime:
  root: processos-workspace       # runs work here
  records: records                # the records given to runs, one folder per instance; kept in git
  output: output                  # where actions write, and their working folder
  runs: runs                      # the state of each run, one folder per run; git-ignored
"""

_SCOPE = """\
# A scope: a name, a version, and the scopes whose definitions this one may name. Its folder holds the
# definitions, one folder for each kind: type/, schema/, template/, action/ and process/.
name: {name}
version: 0.1.0
uses: []
"""

_IGNORE = """\
# What runs write. The records given to runs, in records/, is kept.
output/
runs/
"""


def create_workspace(folder: str | Path, scope: str = "main") -> tuple[list[str], list[Error]]:
    """`(paths, errors)`: a config, an empty scope named `scope`, and the `.gitignore` of the runtime, written in `folder`, which is made
    if it is not there, and named from it. Nothing is written over: when one of them is there, or `scope` is not a scope name, the errors are
    returned and nothing is written. A `folder` that is a file raises `NotADirectoryError`, and one that is not a path, or a `scope` that is
    not text, raises `TypeError`."""
    if not isinstance(folder, (str, Path)) or not isinstance(scope, str):
        raise TypeError(f"a folder is a path and a scope is text, got {folder!r} and {scope!r}")
    root = Path(os.path.abspath(folder))
    if root.exists() and not root.is_dir():
        raise NotADirectoryError(f"{root} is not a folder")
    if errors := [Error(("scope", *error.path), error.code, error.message) for error in check_scope_name(scope)]:
        return [], errors
    files = {
        CONFIG_FILE: _CONFIG,
        f"processos-workspace/definitions/{scope}/scope.yaml": _SCOPE.format(name=scope),
        "processos-workspace/.gitignore": _IGNORE,
    }
    if errors := [Error((str(root / name),), "exists", "a file is there already, and init writes over nothing") for name in files if (root / name).exists()]:
        return [], errors
    target = Folder(root)
    target.make()
    for name, text in files.items():
        target.write(name, text)
    return list(files), []
