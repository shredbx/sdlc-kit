# template — render, conform

process-cli's own template commands: make the files a template produces from a data file, or check
that files already written still hold what a template requires. Read this file for a `/template`
request; `process-os.help` (group `template`) serves the same content live, for an MCP client with
no skill to read a bundled file from.

## `process-cli render TEMPLATE DATA [--into FOLDER]`

Renders every file the template `TEMPLATE` makes from the data in `DATA`, after checking everything. With `--into`, the files are written under `FOLDER`, a folder of the runtime's output (not of the current folder) — a bad render writes nothing. Without `--into`, the files are only listed, never written.

Prints: one line for each file, by its path from `FOLDER`, or one line for each error

## `process-cli conform TEMPLATE DATA FOLDER`

Checks that the files already in `FOLDER` still hold every line the template `TEMPLATE`'s own pattern requires, for the data in `DATA` — the read-only half of `render`, for a folder made another way, or edited since.

Prints: `<FOLDER>: ok`, or one line for each error, each at its file
