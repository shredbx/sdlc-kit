# process-framework

The framework: the config, the definitions and the runtime joined, and entered through `initialize`. Built on `process-kit-types`, `process-kit-schema`, `process-kit-config`, `process-kit-action`, `process-kit-filesystem` and `process-kit-template`.

## Overview

An application, such as the command line, never uses the packages of process-kit itself. It hands
the framework a config file and a `Host`, and gets back the framework, or the errors that stopped it:

## Install

Not installed standalone: pulled in as a dependency by `process-cli` (`uses: [process-framework]`).
From a checkout of this repository, `uv sync --all-packages --group dev` builds it into the
workspace for its own tests.

## Usage

```python
from process_framework import Error, find_config, initialize

file = find_config(".")  # the first processos.yaml here or above, or None
framework = initialize(file, host)  # a ProcessFramework, or a list of Error
framework.config.definitions  # /repo/definitions
framework.catalog.scopes  # {"process-os": Scope(...), "sdlc": ..., "std": ...}
framework.catalog.locate("action", "sdlc.resolve-platform")  # /repo/definitions/sdlc/action/resolve-platform
```

## What it holds

| | |
|---|---|
| `initialize(config_file, host)` | the one way in. It loads the config, opens the catalog of definitions (its default folder and any `libraries`), and gives a `ProcessFramework`, or the list of `Error`s from the config or from a `scope.yaml`. A file that does not exist raises |
| `find_config(start)` | the first `processos.yaml` that is a file, in the folder `start` or one above it, or `None`. It is a help for an application, which is what looks for things; the framework's own code never calls it |
| `read_build(file)` | what a `process-cli.build` stamp holds — `version`, `commit`, `date`, `digest` — or `None` when `file` is not there or does not parse as a mapping. Nothing here writes one; `process-os.build-cli` does |
| `create_workspace(folder, scope="main")` | `(paths, errors)`: writes a `processos.yaml`, an empty scope and the runtime's `.gitignore` in `folder`, which it makes if it is not there, and writes over nothing: a file that is there is an `exists` error and nothing is written. `check_scope_name(name)` is the schema's answer for a scope's name |
| `ProcessFramework` | what `initialize` gives: `config`, `catalog`, the runtime (`output`, `records`, `runs`), `executor` and `host`; and every operation — `check`, `list`, `validate`, `create`, `render`, `conform`, `run`, `resume`, `list_runs`, `show_run` |
| `Catalog` | the definitions folder, opened: `scopes` (name → `Scope`), `locate(kind, id)` (where an id leads, or `None`) and `entries(kind)` (every definition of a kind, by id). `Catalog.open(root, libraries=(), config_file=None)` reads every `scope.yaml` of `root`, plus one scope per `libraries` entry (each `readonly` unless it says `readwrite`, its `path`, `name` and optional `version` checked against its own `scope.yaml`, every error located at `config_file`). **The layout of the folder lives here and only here** |
| `Catalog.node(id)` | the `Action` or the `Process` with that id, or the errors: `unknown_id` when there is none, `duplicate` when an action and a process both have it — neither located in a file, since there is none, or more than one, to name |
| `Catalog.types_of(node)` | the `Types` that a node reaches: its own ports, and, for a process, every node its steps call at any depth, each once. A step that names nothing is passed over (the graph check reports it). Errors are located in the file of the node whose reference they are. It is the one `Types` of a run and of its graph check |
| `Catalog.digest(node)` | a SHA-256 over what a run depends on: the file of every process reached, every file of every action reached, and the file of every type and schema reached. It changes when one of them does, and only then. It raises `ValueError` when `types_of` has errors |
| `Catalog.template(id)` | the `Template` with that id, or the errors (`unknown_id` when there is none). Its data's type is `types`'s business |
| `Catalog.types(references, file)` | the `Types` that a set of `(where, name)` references, written in `file`, reach: every type and schema they name, and the ones those name. A run loads only what its id reaches |
| `Catalog.check()` | every problem in every scope, each located in its file, in a set order: the folder rules, every type and schema, every action, template and process, then every name they use, then the graph of every process (`process_kit.process.check`, once for each, its errors located in the file of the process they are in). It returns `[]` when the definitions are right, and never raises |
| `ProcessFramework.check()`, `list(kind)` | the catalog's `check()` and `entries(kind)` |
| `Records`, `Runs` | the runtime folders. `Records.read(name, folder)` gives `(data, errors)` for `<records>/<folder>/<name>.yaml`, and `Records.file` its path; `Records.gather(input, requires, records, typed, types)` reads and checks what a run starts with, each name in `records` resolved by its own declared type — a file at that exact path, an id under `runtime.records`, or, for a scalar with no such file, the text itself, cast — and finds a `requires` at `<folder>/extension/<name>.yaml` among the ids the call actually read from. `cast_value(text, kind, types, path)` is the module-level casting `--records`'s own literal fallback and `create`'s own `--set` both reuse: a string, an integer, a float, a boolean, or a sequence of one, comma-split. `Runs` is a `RunStore`: `new_id(target)`, `save(run)` and `load(run_id)` name, write and read `<runs>/<run_id>/run.yaml`; `log(run_id, node, stdout, stderr)` writes `<run_id>/<node>/stdout.log` and `stderr.log`; `list()` gives every run id kept, most recent first; `prune(keep)` keeps the `keep` most recent `done` runs and deletes the rest — a `stopped` or `failed` run is never touched, however many there are |
| `ProcessFramework.validate(type_id, file)` | the errors of the data in a file against a type or schema, each located in the file |
| `ProcessFramework.create(schema_id, values, into)` | `(file, errors)`: a new instance of the mapping schema `schema_id`, cast from `values` field by field (the same casting `run`'s own literal values use) and checked against the schema whole, written to `<into>/<schema's own last name>.yaml` under `runtime.records` — refusing to write over one that is already there |
| `ProcessFramework.render(template_id, file, into=None)` | `(paths, errors)`: the files the template makes from the data in `file`, and, given `into`, a folder inside `runtime.output`, written there. Every name, path and file is rendered and checked first, so a bad render writes nothing. Without `into` it only says what it would write |
| `ProcessFramework.conform(template_id, file, folder)` | the errors in the files of `folder`, inside `runtime.output`, against the template and the data: `missing` for a file, `missing_line` for a line, each at that file |
| `ProcessFramework.run(id, records, typed=None)` | runs a process, or an action as a process of one step, `records` mapping each input's name to text, resolved by its own declared type (`Records.gather`'s own rule). It gives the `Run`, saved after every node, or the errors that kept it from starting (nothing is saved then). A process tells the host each node |
| `ProcessFramework.resume(run_id)` | goes on with a run that stopped or failed, from its first node not done, and gives the `Run`; or the errors: `unknown_id`, `invalid` (the file is not a run), and `changed` when a definition it depends on is not what it was when the run started |
| `ProcessFramework.list_runs()` | every run kept, most recent first (`Runs.list`'s order); one whose file is not a run is quietly skipped |
| `ProcessFramework.show_run(run_id)` | the run kept under `run_id`, exactly as `resume` would load it, without going on with it — the same `unknown_id`/`invalid` errors |
| `Scope`, `Entry` | one scope (`name`, `version`, `description`, `uses`, `folder`, `access`: `readonly` or `readwrite`), and one definition a kind has (`id`, `kind`, `file`) |
| `Host` | what only the application knows: `command`, the program an action calls back as `process-cli`, and `report(event)`, to show progress |
| `Error` | the kit's `Error`, passed through, so an application can read errors without importing the kit |

## The catalog

The layout is `<definitions>/<scope>/<namespaces>/<kind>/<name>`, and the kind folder is not part of the id.
A kind is `type`, `schema`, `template`, `action` or `process`. A file `<name>.yaml` is a type, a schema or a
process, and a folder `<name>/` holding `action.yaml` or `template.yaml` is an action or a template. A name is
lower case words joined by hyphens, and an id is two or more names joined by dots.

- **Every folder in the definitions folder is a scope**, and holds a `scope.yaml`, unless its name starts with
  a dot. A file there, such as a README, is not a scope. The file says `name` (the folder's name), `version`,
  `description` and `uses`, and is checked against a schema this package ships, `process_framework.scope`.
- **A scope can also come from a `libraries` entry of the config**, read from its own `path` instead of the
  default folder — the id it is found by is exactly the same either way. Two scopes with one name (a library
  and one in the default folder, or two libraries) is `duplicate`.
- **`uses` is checked when the catalog opens**, and only if every `scope.yaml` is right: a name that is no scope is
  `unknown_scope`, and a scope that uses itself, straight or through others, is `cycle`.
- **`locate` and `entries` read no file.** A broken definition is listed, and found when it is loaded. A name
  that is not a name, such as `Bad.yaml`, is not listed and not found.
- **Errors name the file**: the first step of a path is the absolute path of the `scope.yaml`.
- **`check` applies the layout**: a scope and a namespace hold kind folders and namespaces, and nothing else but the
  `scope.yaml` of a scope; a `type/`, `schema/` or `process/` folder holds `<name>.yaml` files, and an `action/` or
  `template/` folder holds folders named by a name. A stray one is `unexpected`. Names that start with a dot are skipped.
  One id names one definition: a type and a schema with the same id, or an action and a process, are `duplicate`.
- **`types` checks each name it follows**: `unknown_id` when it leads nowhere, `not_used` when its scope is neither the
  referring file's own nor in that scope's `uses`, `duplicate` when a type and a schema have the one id, and `invalid`
  when the file's `name` is not its file name, when a mapping is in `type/`, or when a schema is not a mapping. Each
  definition loads once, even when several name it or it names itself.

## Running

`run("sdlc.resolve-platform", {"application": "hello-app", "build": "hello-app/extension"})` reads
`<records>/hello-app/application.yaml` and `<records>/hello-app/extension/build.yaml` and checks each against its type, loads the
action and the types it reaches, and runs the action in `<output>` with `PROCESS_CLI_CONFIG` set. A process is run the same way,
after its graph is checked; what it `requires` is found beside the ids read, at `<folder>/extension/<name>.yaml`, or given by name
like an input. Every run that starts is saved after each node, whatever the outcome, as `<runs>/yymmdd-hhmmss-<name>/run.yaml`:

```yaml
id: 260921-143000-resolve-platform
process: sdlc.resolve-platform          # an action is a process of one step
scopes: {process-os: 0.1.0, sdlc: 0.1.0, std: 0.1.0}
digest: 1e267e36…                       # what `resume` compares
status: done                            # or running, stopped, failed
done: [sdlc.resolve-platform]           # nodes done or skipped; a nested node is <process>/<node>
skipped: []
context: {application: {…}, build: {…}, platform: {…}}
nested: {}
output: {platform: {name: python, language: python, framework: typer, runtime: '3.13', tool: uv}}
```

An unknown id, a definition that does not load, a graph that does not hold, and an input that is not given, not there, not YAML
or not of its type, come back as errors and nothing is saved; an error in an input is located in its file. The output folder is
made if it is not there. `resume` refuses a run whose definitions have changed, by the digest.

What each action node printed is saved beside the run, right after it runs — whenever at least one of its scripts
actually started (so a skipped node's own `check.sh` is still logged; only a node that never reached the executor at
all, such as one with a bad input, has nothing to save): `<runs>/<run_id>/<node>/stdout.log` and `stderr.log`, a nested node under its own prefixed key, such as
`sdlc.python.build-application/sdlc.python.create-project`. `list_runs()` and `show_run(run_id)` read these back without going
on with a run, for a front end's own `runs`/`runs RUN`.

## Rules

- **Wrong data is returned; a wrong call raises**, as everywhere in the kit. A config with errors is a
  list of `Error`; a config file that does not exist, or a `start` that is not a folder, raises.
- **The framework does not print, read the command line or exit.** That is the application's.
- **It holds no logic a package of the kit should hold.** When something in here starts checking a graph
  or rendering a file, it belongs in a package.
