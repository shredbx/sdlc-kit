---
name: using-process-os
description: This repository's own Claude Code plugin — process-cli over MCP and its own command reference (schema, template, process, runtime), and a skill for when to reach for each.
allowed-tools: Bash(process-cli check:*)
---

# using-process-os

This workspace's own actions and processes are already listed, with real JSON Schema, as MCP tools
— what each one does and what it takes is in its own `description`, read straight off the wire.
`process-cli`'s own commands (`validate`, `create`, `edit`, `render`, `conform`, `run`, `resume`,
`runs`, `init`, `check`, `list`, `show`, `write`, `remove`, `mcp`) are not — nothing in `tools/list`
describes them, since they operate on the workspace itself, not through it; read this skill's own
`tools/` subfolder for those. Either way, this skill is what to do before, between and after
calling one.

## Install process-cli first, if it is not already here

Installing this plugin does not install `process-cli` — the plugin only wires up its MCP server
(`process-cli mcp`) and this skill; if the command itself is missing, that server shows as
disconnected in `/mcp` and nothing below this line works yet. Nothing is published to PyPI, so it
installs from this same repository, by git, no checkout needed:

```bash
uv tool install --from "git+ssh://git@github.com/shredbx/process-os.git#subdirectory=products/process-cli" process-cli
```

or, for one call with nothing kept installed,
`uvx --from "git+ssh://git@github.com/shredbx/process-os.git#subdirectory=products/process-cli"
process-cli <command>` in place of `process-cli <command>` below. `shredbx/process-os` is private,
so this needs the caller's own git access to it (SSH key, or an HTTPS URL with a PAT) — the same
access this repository's own commits already push with. No release is tagged yet, so either line
tracks `main`'s own `HEAD`; `uv tool install --reinstall ...` picks up a change made since the
first install.

Then, if the current folder (or one above it) has no `processos.yaml` yet, this workspace has never
been started: `process-cli init [--scope NAME]` writes one, empty, before anything past this point
means anything. Once it exists, update this repository's own `CLAUDE.md` (create one, briefly, if
there is none) to say it uses process-os: the workspace's own location, and that this skill and its
commands are what to reach for here — nothing else loads automatically at the start of a future
session, so without this, the next one starts from nothing, the same way this one did.

`/init` is the one-keystroke shortcut through this whole section: pass a scope name as its own
argument, or leave it blank to be asked.

## Check before you run

`process-cli check` is not one of the tools `tools/list` shows — it is not an action or a process,
it is what proves every definition in the workspace is still sound. Run it before the first
`tools/call` of a session, and again after editing any definition, not only after a call has
already failed in a confusing way. A tool built from a broken definition either is not listed at
all or fails with a wrong-looking error; `check` says which definition, and why, in one place.

Run `process-cli check` yourself, with your own Bash tool, and report exactly what it says. Not
inlined as a live substitution here on purpose: an inline-executed line in a skill file runs the
instant the skill is invoked at all, before any section is "reached" — including from `/init`,
which invokes this same skill specifically to read "Install process-cli first," above, when neither
`process-cli` nor the workspace is guaranteed to exist yet. An inlined check here would run anyway,
and fail before bootstrapping ever got a chance to (this broke exactly that way once already).

## process-cli's own commands, by group

Read exactly one of `tools/schema.md` (`validate`, `create`, `edit`), `tools/template.md` (`render`,
`conform`), `tools/process.md` (`run`, `resume`, `runs`) or `tools/runtime.md` (`init`, `check`,
`list`, `show`, `write`, `remove`, `mcp`) — whichever the request is about, never the whole
subfolder: each names the exact command line, what it does and what it prints, for that group
alone. `/schema`, `/template` and
`/process` each already say which file matches; for any other request that turns out to need one,
read it the same way. `process-os.help(group)` serves the same content live, for an MCP client with
no skill to read a bundled file from — not needed here, where `Read` already reaches the same file
for nothing.

## Read a stopped run before retrying blind

A `tools/call` against an action or a process blocks until that run stops, however it stops.
`isError` is only ever true for a call that never started — wrong input, an unknown id. A `Run`
that comes back `stopped` or `failed` is not that: it is a normal result, its full state in
`structuredContent`, the same shape `process-cli runs RUN` prints on the command line. Read its
`status`, its `context` and, for a `failed` run, the reason on its own failed node, before calling
the same tool again or reaching for `process-cli resume` — most of the time the reason is right
there, and a blind retry only repeats it.

## Prefer a process's own graph over its steps by hand

Where a process already exists for a goal — `sdlc.build-application`, `sdlc.python.build-application`
today — call it, not the actions it sequences one at a time
(`sdlc.python.create-project`/`install`/`package`/`test`, or `sdlc.resolve-platform`). A process is
checked end to end before it can even run: every input is there, every step's output feeds what
needs it, every branch produces something. Calling its steps by hand instead means re-deriving that
order and those handoffs from memory, with nothing checking that it is right. Reach for the
individual actions only when no process yet covers the goal, or when the goal genuinely is one
step.
