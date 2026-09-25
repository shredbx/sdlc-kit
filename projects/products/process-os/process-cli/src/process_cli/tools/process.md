# process — run, resume, runs

process-cli's own process commands: run an action or a process, go on with one that stopped or
failed, or read what past runs did. Read this file for a `/process` request; `process-os.help`
(group `process`) serves the same content live, for an MCP client with no skill to read a bundled
file from. See the skill's own "Read a stopped run before retrying blind" and "Prefer a process's
own graph over its steps by hand" for the judgment these commands need that no description alone
carries.

## `process-cli run ID [--records NAME=TEXT]...`

Runs the action or process `ID`, after reading and checking its inputs — each `--records NAME=TEXT` resolved by the input's own declared type, in order: an existing file at that exact path, read verbatim; else an id, read as `<id>/NAME.yaml` under the runtime's records; else, for a string, an integer, a float or a boolean, the text itself, cast as that. What the target `requires` is found the same way, among the ids actually read. The run is saved under the runtime's runs after each node, so a stopped or failed run can be resumed.

Prints: `<id>: <outcome>`, with ` — <reason>` when there is one, then `  <output>: <value>` for each output

## `process-cli resume RUN`

Goes on with the run `RUN` — a folder of the runtime's runs — from its first node not yet done. Refuses when a definition it depends on has changed since the run started (its digest no longer matches).

Prints: as `run`; only the nodes that run now report

## `process-cli runs [RUN]`

Without `RUN`: lists every run kept under the runtime's runs, most recent first, a folder that is not a run left out quietly. With `RUN`: one run's status, the reason when it did not end `done`, which of its nodes were done or skipped, and where each node's own `stdout.log`/`stderr.log` is.

Prints: one line per run (`<id>: <process> <status> (<when>)`), or, for one run, `<process>: <status>` then a line per node then `logs: <folder>`
