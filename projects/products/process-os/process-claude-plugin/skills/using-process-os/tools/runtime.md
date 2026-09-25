# runtime — init, check, list, show, write, remove, mcp

process-cli's own workspace commands: start one, check every definition in it, list what it holds,
read or create-or-replace-or-delete one definition or record by id or path, or serve it all over
MCP. Two of these have their own slash command — `/check`, and `/init` (which also installs
`process-cli` itself first, if it is not already here) — the rest are reached for less often, by
name, than the schema/template/process commands are. `process-os.help` (group `runtime`) serves
this same content live, for an MCP client with no skill to read a bundled file from.

## `process-cli init [FOLDER] [--scope NAME]`

Writes `processos.yaml`, `processos-workspace/definitions/<scope>/scope.yaml` (`main` by default) and `processos-workspace/.gitignore` in `FOLDER` (the current folder by default), and writes over nothing: when any of the three is already there, or `--scope` is not a valid scope name, nothing is written and the errors say why. It needs no config, since it makes one.

Prints: each path it wrote, from `FOLDER`, or the `exists`/naming errors

## `process-cli check`

Reads every scope: the folder rules, every type, schema, action, template and process, every name they use, and every process's own graph — every input is in the context and of its type, every switch reads a real value, a branch that stops produces nothing, and the output is produced.

Prints: `<config>: ok`, or one line for each error

## `process-cli list [kind] [prefix] [--long] [--scope S] [--namespace N] [--limit N]`

Lists the ids of one kind (`type`, `schema`, `template`, `action`, `process`), given as the optional argument, or of every kind when none is given — read from the catalog, not the file system directly. `kind records`, with an optional `prefix`, lists the paths under the runtime's records instead: a record has no catalog id. `--long` also shows each definition's own one-line description; `--scope`/`--namespace`/`--limit` filter or cap the bare id list before anything past it is loaded.

Prints: `<id>` (one kind) or `<kind> <id>` (every kind), its own description too with `--long`; or a record's own path

## `process-cli show KIND ID`

Prints a single-file definition's (`type`, `schema`, `process`) raw text verbatim, or an `action`'s own files, each under a `# path` header, in order. `KIND` may also be `record`, the one case where `ID` is not a catalog id at all but a path under the runtime's records — a record has never had one.

Prints: The raw text, or `unknown_id` when there is none

## `process-cli write KIND ID FILE`

For `type`, `schema` or `process`, writes `FILE`'s own text. For `action`, `FILE` is a local folder instead, holding `action.yaml`, its named scripts (`check.sh`, `pre.sh`, `action.sh`, `post.sh`) and anything under `assets/` — the whole folder's tracked content, replaced, not patched. Checked twice before anything is kept: the definition's own shape, then everything it reaches; a failure restores what was there before, or removes what was written if it was new. Refuses an id that is not scope, namespaces, then a name, a scope that does not exist, or a scope that is not `readwrite`.

Prints: `<kind> <id>: written`, or one line for each error

## `process-cli remove KIND ID`

Deletes a file (`type`, `schema`, `process`) or a whole folder (`action`); `KIND` may also be `record`, addressed by path, not a catalog id, the same as `show`. Refuses an unknown id and, for a definition, a scope that is not `readwrite`. Deletes unconditionally beyond that: nothing checks whether another definition still refers to it — `check` is still the tool for that.

Prints: `<kind> <id>: removed`, or one line for the error

## `process-cli mcp`

Starts a real MCP server (the official Python SDK, stdio transport) over whatever workspace the found config points at. `tools/list` shows one tool per action and process in every scope, each with real JSON Schema built from its own input types; `tools/call` runs it, blocking until the run stops, however it stops — a `Run`'s own full state comes back as `structuredContent`. Only a call that never starts (bad input, an unknown id) is a protocol-level error. `--json` is refused: the protocol owns standard output for as long as this runs.

Prints: nothing of its own; the protocol's own messages, on stdio
