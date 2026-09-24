# process-cli

The command line: finds the config, starts the framework, and prints. Built on `process-framework`, and it imports nothing else of this repository.

```bash
process-cli init [FOLDER] [--scope NAME]                                           # The three files a workspace starts with, in a folder.
process-cli check                                                                  # Every definition of every scope.
process-cli list [kind] [prefix] [--long] [--scope S] [--namespace N] [--limit N]  # The ids of the definitions of one kind, or of every kind, or the paths of the records.
process-cli validate TYPE FILE                                                     # A data file against a type or a schema.
process-cli show KIND ID                                                           # The raw text of a definition, by id, or of a record, by its own path under the runtime's records.
process-cli write KIND ID FILE                                                     # Creates or replaces a definition, checked before it is kept.
process-cli remove KIND ID                                                         # Deletes a definition, by id, or a record, by its own path under the runtime's records.
process-cli create SCHEMA --set NAME=VALUE... --into FOLDER [--name NAME]          # A new instance of a schema, from field values given by --set, written under a folder of the runtime's records.
process-cli edit SCHEMA PATH --set NAME=VALUE...                                   # Merges field values into an existing record, checked against its schema whole.
process-cli run ID [--records NAME=TEXT]...                                        # One action or one process, on inputs given by name, each resolved by its own declared type.
process-cli resume RUN                                                             # Go on with a run that stopped or failed, from its first node that is not done.
process-cli runs [RUN]                                                             # Every run kept, most recent first, when RUN is not given; or one run's status, its nodes, and where its logs are.
process-cli render TEMPLATE DATA [--into FOLDER]                                   # The files a template makes from a data file, and, with --into, written under a folder of the output.
process-cli conform TEMPLATE DATA FOLDER                                           # Whether the files in a folder of the output still hold what a template requires.
process-cli mcp                                                                    # Serve this workspace's tools and processes over MCP, on standard input and output, until the client disconnects.
process-cli --config FILE check                                                    # a config that is not processos.yaml
process-cli --json run ID ...                                                      # any command: the answer as one JSON document
process-cli --version                                                              # what it is; needs no config
```

## Install

From a checkout of this repository:

```bash
uv tool install ./products/process-cli              # process-cli on the PATH, with the framework and the kit it needs
uv tool install --reinstall ./products/process-cli  # after a change: uv reuses the build of a version it has seen
uv tool install --editable ./products/process-cli   # or follow the source
uv tool uninstall process-cli
```

Then, in any repository:

```bash
process-cli init                                    # processos.yaml, processos-workspace/definitions/main/scope.yaml, processos-workspace/.gitignore
mkdir -p processos-workspace/definitions/main/action/hello   # write action.yaml and action.sh there
process-cli check
process-cli run main.hello                          # runs write under processos-workspace/, which git-ignores output/ and runs/
```

`init` writes over nothing, and the scope it makes uses no other. The scope `std` of this repository is not shipped: copy it if its types help (`analysis.md`, decision 46). A scope can also live outside `processos-workspace/definitions/`, in a named `libraries` entry of the config, with its own access mode (`readonly` or `readwrite`) and an optional version pin — see `Config`/`Library` (`process_kit.config`).

## Usage

```text
$ process-cli check
processos.yaml: ok

$ process-cli list action
sdlc.resolve-platform

$ process-cli run sdlc.resolve-platform --records application=hello-app --records build=hello-app/extension
sdlc.resolve-platform: done
  platform: {"name": "python", "language": "python", "framework": "typer", "runtime": "3.13", "tool": "uv"}

$ process-cli run sdlc.build-application --records application=hello-app
  sdlc.resolve-platform: done
    sdlc.python.create-project: skipped
    sdlc.python.install: done
    sdlc.python.test: done
    sdlc.python.package: done
  sdlc.python.build-application: done
sdlc.build-application: done
  artifact: {"path": "apps/hello-app", "files": ["dist/hello_app-0.1.0-py3-none-any.whl", "dist/hello_app-0.1.0.tar.gz"]}
```

## What it holds

| | |
|---|---|
| `main(argv=None)` | the entry point (`process-cli`). It reads the arguments, finds the config, calls `initialize`, runs the command, and gives the exit code |
| `commands.py` | one function for each command, each one operation of the framework: `check`, `list_`, `validate`, `run`, `create`, `resume`, `render` and `conform` |
| `Printer` | the command line's `Host`, and where every line goes: results and the lines of the nodes to stdout, errors and messages to stderr. `report` prints one line for each node of a process that ends, indented by how deep it is, and its summary follows |

## The commands

| command | does | prints |
|---|---|---|
| `init [FOLDER] [--scope NAME]` | Writes `processos.yaml`, `processos-workspace/definitions/<scope>/scope.yaml` (`main` by default) and `processos-workspace/.gitignore` in `FOLDER` (the current folder by default), and writes over nothing: when any of the three is already there, or `--scope` is not a valid scope name, nothing is written and the errors say why. It needs no config, since it makes one. | each path it wrote, from `FOLDER`, or the `exists`/naming errors |
| `check` | Reads every scope: the folder rules, every type, schema, action, template and process, every name they use, and every process's own graph — every input is in the context and of its type, every switch reads a real value, a branch that stops produces nothing, and the output is produced. | `<config>: ok`, or one line for each error |
| `list [kind] [prefix] [--long] [--scope S] [--namespace N] [--limit N]` | Lists the ids of one kind (`type`, `schema`, `template`, `action`, `process`), given as the optional argument, or of every kind when none is given — read from the catalog, not the file system directly. `kind records`, with an optional `prefix`, lists the paths under the runtime's records instead: a record has no catalog id. `--long` also shows each definition's own one-line description; `--scope`/`--namespace`/`--limit` filter or cap the bare id list before anything past it is loaded. | `<id>` (one kind) or `<kind> <id>` (every kind), its own description too with `--long`; or a record's own path |
| `validate TYPE FILE` | Checks the data in `FILE` against the type or schema `TYPE`, returning every error found — `path`, `code` and `message` — rather than raising on the first one. `FILE` not existing is reported the same way, before the check runs. | `<file>: ok`, or one line for each error, each in the file |
| `show KIND ID` | Prints a single-file definition's (`type`, `schema`, `process`) raw text verbatim, or an `action`'s own files, each under a `# path` header, in order. `KIND` may also be `record`, the one case where `ID` is not a catalog id at all but a path under the runtime's records — a record has never had one. | The raw text, or `unknown_id` when there is none |
| `write KIND ID FILE` | For `type`, `schema` or `process`, writes `FILE`'s own text. For `action`, `FILE` is a local folder instead, holding `action.yaml`, its named scripts (`check.sh`, `pre.sh`, `action.sh`, `post.sh`) and anything under `assets/` — the whole folder's tracked content, replaced, not patched. Checked twice before anything is kept: the definition's own shape, then everything it reaches; a failure restores what was there before, or removes what was written if it was new. Refuses an id that is not scope, namespaces, then a name, a scope that does not exist, or a scope that is not `readwrite`. | `<kind> <id>: written`, or one line for each error |
| `remove KIND ID` | Deletes a file (`type`, `schema`, `process`) or a whole folder (`action`); `KIND` may also be `record`, addressed by path, not a catalog id, the same as `show`. Refuses an unknown id and, for a definition, a scope that is not `readwrite`. Deletes unconditionally beyond that: nothing checks whether another definition still refers to it — `check` is still the tool for that. | `<kind> <id>: removed`, or one line for the error |
| `create SCHEMA --set NAME=VALUE... --into FOLDER [--name NAME]` | Builds a new instance of the mapping schema `SCHEMA` field by field: each `--set NAME=VALUE` is cast from text the same way `run`'s own literal records are, the whole instance checked against the schema, then written to `<into>/<NAME or SCHEMA's own last name>.yaml` under the runtime's records — refusing to write over one that is already there. `--into` is required: unlike `render`, there is no preview mode. `--name`, checked the same way any other name is, lets one folder hold more than one instance of the same schema. | `<file>: written`, or one line for each error |
| `edit SCHEMA PATH --set NAME=VALUE...` | Casts each `--set NAME=VALUE` the same way `create` casts a new record, merges it into what is already at `PATH` (a path under the runtime's records, not a catalog id), and checks the *whole* merged result against `SCHEMA` before writing anything — nothing is touched until it is already known to be valid, so there is no separate rollback to reason about. | `<path>: edited`, or one line for each error |
| `run ID [--records NAME=TEXT]...` | Runs the action or process `ID`, after reading and checking its inputs — each `--records NAME=TEXT` resolved by the input's own declared type, in order: an existing file at that exact path, read verbatim; else an id, read as `<id>/NAME.yaml` under the runtime's records; else, for a string, an integer, a float or a boolean, the text itself, cast as that. What the target `requires` is found the same way, among the ids actually read. The run is saved under the runtime's runs after each node, so a stopped or failed run can be resumed. | `<id>: <outcome>`, with ` — <reason>` when there is one, then `  <output>: <value>` for each output |
| `resume RUN` | Goes on with the run `RUN` — a folder of the runtime's runs — from its first node not yet done. Refuses when a definition it depends on has changed since the run started (its digest no longer matches). | as `run`; only the nodes that run now report |
| `runs [RUN]` | Without `RUN`: lists every run kept under the runtime's runs, most recent first, a folder that is not a run left out quietly. With `RUN`: one run's status, the reason when it did not end `done`, which of its nodes were done or skipped, and where each node's own `stdout.log`/`stderr.log` is. | one line per run (`<id>: <process> <status> (<when>)`), or, for one run, `<process>: <status>` then a line per node then `logs: <folder>` |
| `render TEMPLATE DATA [--into FOLDER]` | Renders every file the template `TEMPLATE` makes from the data in `DATA`, after checking everything. With `--into`, the files are written under `FOLDER`, a folder of the runtime's output (not of the current folder) — a bad render writes nothing. Without `--into`, the files are only listed, never written. | one line for each file, by its path from `FOLDER`, or one line for each error |
| `conform TEMPLATE DATA FOLDER` | Checks that the files already in `FOLDER` still hold every line the template `TEMPLATE`'s own pattern requires, for the data in `DATA` — the read-only half of `render`, for a folder made another way, or edited since. | `<FOLDER>: ok`, or one line for each error, each at its file |
| `mcp` | Starts a real MCP server (the official Python SDK, stdio transport) over whatever workspace the found config points at. `tools/list` shows one tool per action and process in every scope, each with real JSON Schema built from its own input types; `tools/call` runs it, blocking until the run stops, however it stops — a `Run`'s own full state comes back as `structuredContent`. Only a call that never starts (bad input, an unknown id) is a protocol-level error. `--json` is refused: the protocol owns standard output for as long as this runs. | nothing of its own; the protocol's own messages, on stdio |

## `--json`

`process-cli --json <command> ...` answers with one JSON object on one line of standard output, and nothing else there: what an action's scripts write to standard output goes to standard error while it works. The object always has `ok` (the exit code is 0 exactly when it is true) and `errors` (`[{path, code, message}]`, the `path` a list with the file first), and:

| command | adds |
|---|---|
| `check` | `config`, and `scopes`: `[{name, access, library}]` for every scope, `library` true for one read from `libraries`, not `definitions` |
| `list` | `entries`: `[{kind, id, file}]` |
| `validate` | `file` |
| `render` | `paths` |
| `conform` | `folder` |
| `init` | `paths` |
| `create` | `file` |
| `run`, `resume` | `run`: the whole run, as `run.yaml` holds it, and `events`: `[{node, state, reason}]`. A run that did not start has no `run` |
| `runs` | `runs`: `[{id, process, status, when}]`, most recent first |
| `runs RUN` | `run`: the whole run, as `run.yaml` holds it, and `logs`: the run's own folder |

A problem before the framework is asked (no config, a file that is not there) is one line on standard error and exit 1, and a wrong call is the usage and exit 2, as in text.

## `--version`

`process-cli --version` prints what it is and needs no config. Beside a built executable
(`process-os.build-cli`'s output), `process-cli.build` is a stamp — `version`, and optionally
`commit` and `date` (a `digest` of the sources it was built from, for the build action's own use) —
and, when it is there and names a `version`, that is what `--version` shows: `process-cli 0.2.0
(built 2026-09-22, commit 1a2b3c4)`. Without one, as in a checkout, it shows the version this
package was installed as: `process-cli 0.1.0 (source checkout)`.

## MCP

`process-cli mcp` is a real MCP server (the official Python SDK, stdio transport) over whatever
workspace its config points at — the same `processos.yaml`-finding rule every other command uses,
nothing MCP-specific about it. Point a client at it directly, in `.mcp.json`:

```json
{
  "mcpServers": {
    "process-os": { "command": "process-cli", "args": ["mcp"] }
  }
}
```

(From inside this repository's own uv workspace, where `process-cli` is not on `PATH` as such, use
`{"command": "uv", "args": ["run", "--package", "process-cli", "process-cli", "mcp"]}` instead — see
this repository's own `.mcp.json`.) A Claude Code plugin can wire the same thing inline, in its own
`plugin.json`, instead of a separate file — `products/process-claude-plugin` does exactly that; see
its own README.

Once connected: `tools/list` shows one tool per action and process, in every scope, each with real
JSON Schema built from its own input types (no hand-written description of the shape — the schema
*is* the description), and one further tool, `process-os.help`, that is not one of them —
`process-cli`'s own command surface, by capability group (`schema`, `template`, `process`,
`runtime`), read from the same records this README's own "## The commands" table comes from, for a
client with no bundled skill file to read it from otherwise. `tools/call` runs a workspace tool,
blocking until the run stops, however it stops — a `Run`'s own full state comes back as
`structuredContent`, the same shape `runs RUN` prints on the command line; only a call that never
starts (bad input, an unknown id) is a protocol-level error. The server's own `instructions` — read
by a client before any tool is even called — say to check first, the same governance a Claude Code
session gets from the plugin's own skill instead.

## A placed copy

A folder such as `tools/process-os/cli/` — made in this repository with
`process-cli run process-os.build-cli --records target=tools/process-os/cli`, or simply copied from
one that was — needs nothing installed. It holds:

```text
tools/process-os/cli/
  process-cli            the executable (a zipapp; the target machine needs Python 3.13)
  process-cli.build      the stamp --version reads
  processos.yaml         found beside it automatically (Rules, below), unless one is found first
  libraries/std/         a read-only copy of std, so a workspace placed here can still use it
```

A second build in this repository skips once its stamp still matches the sources; it never writes
over a `processos.yaml` or a `libraries/<name>` already there, so an edit you made survives a
rebuild. Placing the folder is nothing more than copying it: nothing in it points back at this
repository, or needs `uv`, once it is built.

## Rules

- **Where the config is**, in this order: `--config FILE`; the environment variable
  `PROCESS_CLI_CONFIG`; the first `processos.yaml` looking upward from the current folder; one
  beside the program (a symlink is followed to the real file first). The third still wins over the
  fourth: a workspace's own config, found by looking upward, is never shadowed by one placed beside
  a tool used across several workspaces. A named file that does not exist is an error, and so is
  finding nothing. `--config` goes before the command.
- **Exit codes:** 0 everything is fine, 1 errors were found or an action stopped or failed, 2 a wrong
  call: an unknown command or kind, a missing argument (`create`'s own `--into` included, since it has
  no default), a `--records` or a `--set` that is not `NAME=TEXT`, or a name given more than once.
- **An error prints as one line**: the file relative to the current folder, the path in it, the
  code, and the message, such as `processos.yaml: definitions: missing: "definitions" is required`. An
  error that names no file, such as an id that leads nowhere, prints the code and the message.
- **An action can call `process-cli`.** When the program running is named `process-cli`, `run` puts one on the `PATH`
  of the action, so a script can call `process-cli render` and `process-cli conform` on the same workspace:
  `PROCESS_CLI_CONFIG` is set too. When it is not — a renamed copy, or a launcher that starts it another way —
  `PROCESS_CLI_PROGRAM`, set to the real program before it runs, takes its place. Nothing that merely happens to
  share the running interpreter, such as a test runner, is ever offered as a stand-in without one of these two.
- **A run that stops or fails is saved, and says so** on stderr: `run <id> is saved; go on with: process-cli resume <id>`.
- **It never imports `process_kit`.** Everything it needs, `Error`, `Run` and `Event` included, comes from
  `process_framework`.
