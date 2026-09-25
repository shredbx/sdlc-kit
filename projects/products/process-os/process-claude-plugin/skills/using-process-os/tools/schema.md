# schema — validate, create, edit

process-cli's own schema commands: check a data file against a type or a schema, build a new
instance of one and write it for real, or merge field values into one that already exists. Read
this file for a `/schema` request; `process-os.help` (group `schema`) serves the same content live,
for an MCP client with no skill to read a bundled file from.

## `process-cli validate TYPE FILE`

Checks the data in `FILE` against the type or schema `TYPE`, returning every error found — `path`, `code` and `message` — rather than raising on the first one. `FILE` not existing is reported the same way, before the check runs.

Prints: `<file>: ok`, or one line for each error, each in the file

## `process-cli create SCHEMA --set NAME=VALUE... --into FOLDER [--name NAME]`

Builds a new instance of the mapping schema `SCHEMA` field by field: each `--set NAME=VALUE` is cast from text the same way `run`'s own literal records are, the whole instance checked against the schema, then written to `<into>/<NAME or SCHEMA's own last name>.yaml` under the runtime's records — refusing to write over one that is already there. `--into` is required: unlike `render`, there is no preview mode. `--name`, checked the same way any other name is, lets one folder hold more than one instance of the same schema.

Prints: `<file>: written`, or one line for each error

## `process-cli edit SCHEMA PATH --set NAME=VALUE...`

Casts each `--set NAME=VALUE` the same way `create` casts a new record, merges it into what is already at `PATH` (a path under the runtime's records, not a catalog id), and checks the *whole* merged result against `SCHEMA` before writing anything — nothing is touched until it is already known to be valid, so there is no separate rollback to reason about.

Prints: `<path>: edited`, or one line for each error
