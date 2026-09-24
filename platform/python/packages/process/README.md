# process-kit-process

Processes. Read a process from its file and check it. Its steps are values, and the type of a step is written in code, since a step is one of three shapes. Check the whole graph before anything runs. Run a process one node at a time, save the run after each node, and go on from where it stopped. Built on `process-kit-types`, `process-kit-schema` and `process-kit-action`.

## Overview

```yaml
# definitions/sdlc/process/build-application.yaml
name: build-application                    # the file's name
input:    {application: application}       # name -> the name of its type
requires: {build: build}                   # what it needs beside its input: found beside the input it extends
output:   {artifact: build-artifact}
steps:
  - resolve-platform                       # a node: an action or a process, by id
  - switch: platform.name                  # a choice on a value in the context: a name, then fields
    cases:
      python: [sdlc.python.build-application]
    default:
      - stop: no build process for this platform yet       # the end of the run, on purpose
```

## Install

Not installed standalone: pulled in as a dependency by `process-framework`
(`uses: [process-kit-process]`). From a checkout of this repository, `uv sync --all-packages --group dev`
builds it into the workspace for its own tests.

## Usage

```python
from process_kit.process import Process

process = Process.load("definitions/sdlc/process/build-application.yaml", "sdlc")  # a Process, or a list of Error
process.id  # sdlc.build-application
process.steps  # (Call(id='sdlc.resolve-platform'), Switch(on='platform.name', ...))
```

## What it holds

| | |
|---|---|
| `Process` | a frozen dataclass: `id` (in full), `description` (or `None`), `input`, `requires` and `output` (name → the full name of its type, read-only, in the order written), `steps` (a tuple of steps) and `home`, the file |
| `Process.load(file, namespace="")` | the process in a file named `<name>.yaml`, or every error in it, never both |
| `Process.of_action(action)` | the process of one step that runs the action, with its input and output, so an action is run as any process is |
| `Process.references()` | the types the three ports name, in full, with where each is written |
| `Call`, `Switch`, `Stop` | the three steps: a node by id, a choice on a value with its cases and an optional default, and an end with its reason. `Step` is any of them |
| `StepType` | the type of a step in a definition, for `validate`; it refers to no type, since a step names nodes |
| `check(process, resolver, types)` | every error in the graph of a process, before anything runs, or `[]` (below) |
| `Resolver` | what says what an id names: `node(id)` gives an action, a process, or the errors. A class is one by having `node` |
| `Runner(resolver, types, executor, store, context, on_event=None)` | runs processes. `start(process, inputs, scopes, digest)` makes a `Run` and walks it; `resume(run, process=None)` goes on from the first node not done, with the given process (whose id is the run's target) or the resolver's. It reads no file: everything it uses is handed to it |
| `Run` | the whole state of one run: `id`, `target`, `scopes`, `digest`, `status` (`running`, `done`, `stopped`, `failed`), `done`, `skipped`, `context`, `nested`, `output`, `reason`, `errors`. `to_data()` and `Run.from_data(data)` are each other's inverse |
| `Event`, `RunStore` | what the runner tells (`run`, `node`, `state`, `reason`), and where a run is kept: `new_id`, `save`, `load`, and `log(run_id, node, stdout, stderr)`, keeping what a node's scripts printed |
| `register(types)` | adds `process_kit.process.process` and `.steps`, and `.step`, the type written in code. The types `process_kit.action.name` and `.ports` must be there: register `action` first |

## The graph check

`check` walks the steps with a **context**: name → the full name of a type. It starts with the process's `input` and `requires`.

- **A node** (a `Call`) must exist (`unknown_id`), must not be a process that is running (`cycle`), and must not run twice on one path (`duplicate`). Each input it takes must be in the
  context (`missing`) and of its type (`type_mismatch`). What it gives joins the context: for a process, only its declared `output`.
- **A switch** reads `name.field.field`, followed through the types' fields: the name must be in the context (`missing`) and each field must exist (`unknown_field`). When the value's type
  has an `enum`, each case must be a value of it (`invalid`), and without a `default` every value needs a case (`unhandled`). A branch that ends in a `stop` produces nothing and is left
  out; after the switch the context holds what all the other branches give, with one type (`type_mismatch`), and, when no `default` or `enum` covers every value, what the path that takes no case gives.
- **A stop** ends the flow, and a step after it is `unreachable`.
- **The output**: at the end, each declared output must be in the context (`not_produced`) and of its type (`type_mismatch`), unless the flow always stops.
- **Errors are located in the file of the process they are in**, at the step. A nested process is checked once, and its errors are in its own file.

## The runner

- **A node's key** is its id in full, and inside a nested process `<the process node's key>/<the id>`. A key joins `done` when its node is `done` or `skipped`, and a process node's when its steps are. A skipped action's key joins `skipped` too, and so does a process node's when every node in it was skipped.
- **The context is saved after each node** (S9), and a node's outputs are kept only once the node has gone through. A nested process's context is kept in `run.nested` while it runs, so a resume goes on inside it. Only the
  outputs a process declares leave it (S7).
- **A run ends** `done`, `stopped` (a `stop` step, or an action whose `check.sh` says stop) or `failed` (an action failed, no case fits a switch, or the declared output is not right). `resume` clears the reason and goes on from the first
  node not done; a run that is `done` is returned as it is.
- **The runner trusts** that the graph was checked and that the inputs were checked by whoever read them. It does not check the digest of the definitions.

## Rules

- **A step is an id, a `switch` or a `stop`.** Nothing else, and only one. A `switch` has `cases`, a mapping of value to steps, and may have a `default`; a `stop` has a reason. Ids and a switch's path are names
  joined by dots. A `switch` reads a value that is in the context, never an expression.
- **Names are made full when the process loads.** A bare id in a namespace gets it, a dotted one stays (`process_kit.schema.qualify_id`), and so do the type names of the ports (`qualify`).
- **Wrong data is returned; a wrong call raises.** Errors are `Error`s, each path starting with the file, and `steps` errors are located at the step: `[<file>, steps, 1, cases, python, 0]`. A file that is not there, a
  folder, a name that is not `<name>.yaml`, and a wrong namespace raise.
- **A YAML key that is a boolean shows as text in a path** (`yes` reads as `true`), as everywhere in the kit.
