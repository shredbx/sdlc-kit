# process-claude-plugin

This repository's own Claude Code plugin — `process-cli` over MCP and its own command reference (schema, template, process, runtime), and a skill for when to reach for each.

```bash
/init [scope name]                      # install process-cli if missing, start the workspace if it hasn't been, then ask what to build
/check                                  # process-cli check, printed back verbatim
/schema <what you want to do>           # validate or create, routed to using-process-os
/template <what you want to do>         # render or conform, routed to using-process-os
/process <what you want to do>          # run, resume or runs, routed to using-process-os
```

## Install

Needs `process-cli` on your own `PATH` already. From a checkout of this repository (see
`products/process-cli`'s own README):

```bash
uv tool install ./products/process-cli
```

or, with no checkout at all, straight from git (`shredbx/process-os` is private, so this needs the
caller's own git access to it):

```bash
uv tool install --from "git+ssh://git@github.com/shredbx/process-os.git#subdirectory=products/process-cli" process-cli
```

Then, in Claude Code, add this repository as a marketplace and install the plugin from it:

```
/plugin marketplace add <path-or-url-to-process-os>
/plugin install process-claude-plugin@process-os
```

Restart Claude Code — a plugin's commands, skill and MCP server only load on a fresh session, not
the one that just installed it. After that, `/mcp` shows `process-claude-plugin` connected once
`process-cli` is on `PATH` (run `/init` first if it is not yet — see below), and `/help` (or
`/check`, directly) shows the commands.

## Update

**`plugin.json`'s own `version` must go up with every content change, not only a deliberate
release.** Claude Code's local plugin cache is keyed by that exact string
(`~/.claude/plugins/cache/<marketplace>/<plugin>/<version>/`); reinstalling with the version
unchanged reuses whatever is already cached there instead of re-extracting the new content, however
many times `marketplace update` itself re-fetches the underlying repository. This bit the `/init`
fix directly: the marketplace's own clone had it, `/plugin install` reported success, and the cache
was still serving the broken file from before the fix — because `version` had stayed `0.1.0` since
task 03. Bump it (`0.2.0` here, once, to force a clean cache path) on every change from here on.

Once this plugin's own content (a command, the skill, `plugin.json`) changes, its version bumped,
and both pushed to `main`, a session that already has it installed does not see the change until
told to look again:

```
/plugin marketplace update process-os
/plugin uninstall process-claude-plugin@process-os
/plugin install process-claude-plugin@process-os
```

`marketplace update` refreshes the listing itself; reinstalling is what actually picks up the new
plugin content — this is the same cycle Claude Code's own docs give for iterating on a plugin under
development, not something specific to this one. Restart Claude Code afterward, same as a first
install: commands, skill and MCP server only (re)load on a fresh session. `process-cli` itself is
separate and unaffected by any of this — see Install, above, for updating *that*.

## Usage

Once the plugin is installed and Claude Code restarted, run `/init` once in the target repository:
it installs `process-cli` itself if it is missing, starts the workspace if it has not been, and
brings that repository's own `CLAUDE.md` up to date so a future session knows this is in use here
without repeating any of it. After that, this plugin needs no further setup: every action and
process of the workspace Claude Code is pointed at becomes an MCP tool automatically, described by
its own real JSON Schema — nothing to explain by hand. `process-cli`'s own commands (`validate`,
`create`, `render`, ...) are not workspace tools, so the skill `using-process-os` carries those
instead, one reference file per capability group, read only when a request needs it. `/init`,
`/check`, `/schema`, `/template` and `/process` are one-keystroke shortcuts into the same skill.

## What it installs

- **The MCP server `process-claude-plugin`**, started with `process-cli mcp` — every action and
  process of whatever workspace the current folder's own `processos.yaml` points at, listed with
  real JSON Schema and served through `run`, plus one further tool, `process-os.help`, for
  `process-cli`'s own commands. Wired inline in this plugin's own `plugin.json`, not a separate
  `.mcp.json` a person has to write by hand.
- **The skill `using-process-os`** — not what each MCP tool does (its own `description`, read
  straight off the wire, already says that), but when to reach for which one: check before you run,
  read a stopped run's own state before retrying blind, prefer a checked process over its own steps
  by hand — and, in its own `tools/` subfolder, one file per capability group documenting
  `process-cli`'s own commands. See `skills/using-process-os/SKILL.md` for the real content.
- **The commands `/init`, `/check`, `/schema`, `/template` and `/process`** — one-keystroke
  shortcuts into the skill: `/init` bootstraps (installs `process-cli` if missing, starts the
  workspace if it hasn't been, updates this repository's own `CLAUDE.md`); `/check` runs
  `process-cli check`, printed back verbatim; the other three carry a free-text request into the
  matching group's own reference file.

## Before it does anything useful

The workspace it is pointed at needs `process-cli init` run in it first, and at least one real
definition after that: `init`'s own scope starts empty, so there is nothing yet for `tools/list` to
show, and nothing for `/check` to be interesting about. `/init` does the first part for you; the
second — real content — is still a real design decision, not something any command should guess.

## No config pinned

`plugin.json` sets no `PROCESS_CLI_CONFIG`. Installed into an arbitrary repository, the MCP server
it starts finds that repository's own config the same way every other `process-cli` command does:
the nearest `processos.yaml`, looking upward from wherever Claude Code started it — never this
repository's own.
