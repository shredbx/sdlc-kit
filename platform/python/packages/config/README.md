# process-kit-config

Config. Read the config file, check it, and give every path in it resolved. Built on `process-kit-types` and `process-kit-schema`.

## Overview

The config says two things: where the definitions are, and where runs work.

```yaml
# processos.yaml
version: 0.1.0                  # the shape of this file, optional; None for a config written before it existed
definitions: processos-workspace/definitions   # the folder of scopes
runtime:
  root: .                         # the folder runs work in
  records: processos-workspace/records  # inside root; default: records
  output: .                       # inside root; default: output
  runs: processos-workspace/runs  # inside root; default: runs
  history: 10                     # the most recent done runs kept; default: 10
libraries:                        # optional: a scope kept in another folder
  - name: sdlc                    # must equal the scope's own name in its scope.yaml
    path: processos-workspace/definitions/sdlc   # relative to this file
    access: readonly              # readonly (default) or readwrite
    version: 0.2.0                # optional: must equal the scope's own version
```

## Install

Not installed standalone: pulled in as a dependency by `process-framework`
(`uses: [process-kit-config]`). From a checkout of this repository, `uv sync --all-packages --group dev`
builds it into the workspace for its own tests.

## Usage

```python
from process_kit.config import Config

config = Config.load("processos.yaml")  # a Config, or a list of Error
config.definitions  # /repo/definitions
config.runtime.output  # /repo
```

## What it holds

| | |
|---|---|
| `Config` | `file`, `definitions`, `runtime`, `version` (the config's own shape, or `None` for one written before it existed) and `libraries` (a tuple of `Library`, `()` by default), every path absolute |
| `Runtime` | `root`, `records`, `output` and `runs`, every path absolute; `history`, an `int`, 10 by default |
| `Library` | one `libraries` entry: `name`, `path` (absolute), `access` (`readonly` or `readwrite`) and an optional `version`. Whether the folder is really there, and really names this scope and version, is not checked here — that needs the folder read, which is the catalog's business |
| `Config.load(file)` | the config in a file, or every error in it — never both |
| `register(types)` | adds the schemas of the file to a `Types`, as `process_kit.config.config`, `.runtime`, `.path`, `.libraries`, `.library`, `.access`, `.history`, `.text` and `.version`, so the shape of a config can be listed and checked like any other type. It returns the errors, and none when it worked |

## Rules

- **Paths.** `definitions` and `runtime.root` are relative to the folder the config is in. `records`,
  `output` and `runs` are relative to `runtime.root`, and default to their own names. A path comes
  back absolute, with every `..` folded away and no symlink followed.
- **Wrong data is returned.** An error in the file, a `definitions` or `runtime.root` that is not a
  folder (`not_found`), and a `records`, `output` or `runs` that leaves `runtime.root`
  (`outside_root`), are all `Error`s. Each path in an error starts with the file, as an absolute path.
  All the errors of one file come back together, in a fixed order.
- **A wrong call raises.** A file that does not exist, or is a folder, raises `FileNotFoundError` or
  `IsADirectoryError`.
- **Nothing looks for anything.** `Config.load` takes the file it is given. Finding `processos.yaml`
  is the application's job, and `process_framework.find_config` offers the search.
- **Nothing is created.** `output` and `runs` need not exist. A run makes them.
- The file is read with `process_kit.schema.parse`, so a key written twice is an error.
