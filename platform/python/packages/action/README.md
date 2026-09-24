# process-kit-action

Actions. Read an action from its folder, and run it, with its inputs and outputs checked against their types. Built on `process-kit-types`, `process-kit-schema` and PyYAML.

## Overview

An action is a folder. `action.yaml` says what it takes and gives, and `action.sh` is what it does:

```yaml
# definitions/sdlc/action/resolve-platform/action.yaml
name: resolve-platform          # the folder's name
type: shell                     # every action is a shell script for now
input:
  application: application      # a name, and the name of its type
  build: build
output:
  platform: platform
```

## Install

Not installed standalone: pulled in as a dependency by `process-framework`
(`uses: [process-kit-action]`). From a checkout of this repository, `uv sync --all-packages --group dev`
builds it into the workspace for its own tests.

## Usage

```python
from process_kit.action import Action

action = Action.load("definitions/sdlc/action/resolve-platform", "sdlc")  # an Action, or a list of Error
action.id  # sdlc.resolve-platform
action.input  # {"application": "sdlc.application", "build": "sdlc.build"}

result = action.run(inputs, Context(output=Path("out")), types, ShellExecutor())
result.outcome  # Outcome.done
result.outputs  # {"platform": {...}}
```

## What it holds

| | |
|---|---|
| `Action` | a frozen dataclass: `id` (in full), `kind`, `description` (or `None`), `input` and `output` (name → the full name of its type, read-only, in the order written), `timeout` (seconds one hook may run; 60 when `action.yaml` leaves it out, 0 meaning no limit), and `home`, the folder |
| `Action.load(folder, namespace="")` | the action in a folder, or every error in it — never both |
| `Action.references()` | each type name the inputs and then the outputs name, in full, with where it is written: `[(("input", "application"), "sdlc.application"), …]`. It is what a caller needs to load the types an action reaches |
| `Action.run(inputs, context, types, executor)` | checks each input against its type, runs the action through the executor, and checks each output against its type. It gives a `Result`, and does not raise for a wrong input or output |
| `Result`, `Outcome` | what one run gave: `outcome` (`done`, `skipped`, `stopped` or `failed`), `outputs` (name → value, none unless done or skipped), `reason` (why it did not go on) and `errors` (what was wrong with a value) |
| `Context` | all an action is handed besides its inputs: `output`, the folder it works in, and `environment`, the variables its caller adds |
| `Executor` | the protocol for how the scripts run: `execute(action, inputs, context) -> Result`. A class is one by having `execute` |
| `ShellExecutor` | the executor for shell actions: the contract below |
| `register(types)` | adds the schemas of `action.yaml` to a `Types`, as `process_kit.action.action`, `.kind`, `.name`, `.ports`, `.timeout` and `.type-name`. It returns the errors, and none when it worked |

## Rules

- **Type names are made full when the action loads.** `application` in the namespace `sdlc` is
  `sdlc.application`. A name with a dot, and a YAML type such as `integer`, stay as written
  (`process_kit.schema.qualify`). The action's own id is the folder's name with the namespace before it.
- **Wrong data is returned.** An error in `action.yaml`, a `name` that is not the folder's name, and a
  shell action with no `action.sh` are all `Error`s. Each path starts with the file, as an absolute
  path. The errors of the schema come first, and when there are any the two later checks are not made.
- **A wrong call raises.** A folder that is not there, that has no `action.yaml`, or that is a file
  raises `FileNotFoundError` or `NotADirectoryError`; a namespace that is not text, or is `.a` or
  `a..b`, raises. Finding the folder of an id is the framework's catalog, which asks first.
- **Names are lower case words joined by hyphens.** That holds for the action, for each input and each
  output, and it is checked as a key (`keys`), not only as a value. Input names become `INPUT_<NAME>`
  variables when the action runs, so what they may contain matters.
- **The schemas ship with the package.** `register` loads them from `schemas/`, so the format of an
  action can be listed and checked like any other type.

## The shell contract

`ShellExecutor` runs each script as `bash <action folder>/<name>.sh`, in `context.output`, with no
standard input. The folder must exist, or the call raises `NotADirectoryError`. Each script, on its
own, may run up to `action.timeout` seconds — 60 when `action.yaml` does not say, `0` meaning no
limit — before it is killed and failed as timed out.

| given to the scripts | |
|---|---|
| `PROCESS_OUTPUT` | the folder the script works in |
| `ACTION_HOME` | the action's own folder; its files are in `$ACTION_HOME/assets` |
| `ACTION_INPUTS` | a folder with `<input>.yaml` for each input: the whole value |
| `ACTION_OUTPUTS` | an empty folder. The script writes `<output>.yaml` for each output |
| `INPUT_<NAME>_<FIELD>` | each top-level field of a mapping input that is not a mapping or a list; `INPUT_<NAME>` for an input that is not a mapping. Upper case, `-` as `_`; a boolean is `true` or `false` |
| the rest | the caller's environment, then `context.environment`. The variables above win |

| script | when | exit |
|---|---|---|
| `check.sh` | first, if there | 0 go on · 77 **skip**, and the outputs it wrote are the result · other **stop**, with its standard error as the reason |
| `pre.sh` | before `action.sh`, if there | 0, or the action **fails** |
| `action.sh` | always, unless skipped | 0, or the action fails |
| `post.sh` | after `action.sh`, if there | 0, or the action fails and its outputs are not kept |

A failed script's reason is `<name>.sh exited N`, and its standard error after a colon when it wrote
any. A script that ran longer than `action.timeout` gives a different reason instead, `<name>.sh timed
out after Ns` — never foldable into the exit-code wording above — and `check.sh`'s own timeout still
**stops** the run, the same outcome a `check.sh` that says no already gives, not a new kind of failure.
Standard output is not captured: it goes where the process's own goes. After the scripts, each
output file is read as YAML and checked against its type. A wrong one, or a missing one, fails the
action, and so does a skip that wrote one.
