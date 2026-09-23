# process-os: concrete definition patterns

Last updated: 2026-09-23
tags: process-os, patterns, research

Read directly from `/Users/solo/Projects/workspaces/process-os` — its own `CLAUDE.md`, and the
actual YAML/shell files of its `process-os`, `std` and `sdlc`/`sdlc.python` scopes (process-os is
itself dogfooded as a process-os workspace). This is the primary source for how *we* should shape
`sbx-sdlc-kit`'s own definitions — not a copy target (see [[shredbx-sdlc-system]] for what not to
copy from elsewhere), but the tool's own reference implementation of itself.

## The five kinds, concretely

Every scope folder has up to five kind-subfolders: `type/`, `schema/`, `action/`, `template/`,
`process/`. An id is `scope.[namespace.]name` — the kind folder is never part of the id.

### `type` — a primitive value, constrained

One YAML file each. Three shapes seen in the wild:

```yaml
# sdlc/type/language.yaml — enum
name: language
type: string
enum: [python, typescript, javascript]
```

```yaml
# std/type/relative-path.yaml — regex-constrained string
name: relative-path
type: string
pattern: '(?!\.\.?(/|$))[A-Za-z0-9_.-]+(/(?!\.\.?(/|$))[A-Za-z0-9_.-]+)*'
```

```yaml
# std/type/file-list.yaml — sequence of another type
name: file-list
type: sequence
items: relative-path
```

Every type/schema file's leading comment states *why* the constraint exists, in plain words — this
is the house style worth carrying over verbatim (e.g. `relative-path`'s comment gives two concrete
examples: `apps/hello-app`, `dist/a-0.1.0.tar.gz`).

### `schema` — a `mapping`, always

```yaml
# sdlc/schema/application.yaml
name: application
type: mapping
fields:
  name:        {type: application-name, required: true}
  description: {type: string, required: true}
  version:     {type: std.version, required: true}
  platform:    {type: platform-name, required: true}
  language:    {type: language, required: true}
  framework:   {type: framework, required: true}
```

Fields reference types by bare name (same scope) or `scope.name` (cross-scope, only through
`uses:`). `type`/`schema` share one id lookup — the only difference is a schema is always
`mapping`, a type never is.

### `action` — a folder, shell-backed today

```yaml
# sdlc/action/resolve-platform/action.yaml
name: resolve-platform
type: shell
input:
  application: application
  build: build
output:
  platform: platform
```

Folder contents: `action.yaml` (contract), `action.sh` (the work), optional `check.sh` (idempotency
— e.g. `sdlc.python.create-project`'s `check.sh` refuses to overwrite an existing project),
`post.sh` (e.g. `process-os.create-package`'s `post.sh` adds the new package to the uv workspace and
imports it), `assets/` (extra files the scripts use, e.g. a shared `lib.sh`).

Shell contract: input fields arrive as `$INPUT_<PATH_IN_CAPS>` env vars (e.g.
`$INPUT_APPLICATION_PLATFORM`, `$INPUT_BUILD_RUNTIME` — nested field, underscored); the script
writes one YAML file per output field to `$ACTION_OUTPUTS/<field>.yaml`. `set -euo pipefail`, and
`exit 1` with a one-line stderr message is how an action stops the process cleanly (see
`resolve-platform`'s action.sh: an unrecognized platform or a language/platform mismatch exits 1
with a plain-English reason *before* anything is written).

### `template` — files to render, or a pattern to conform against

```yaml
# process-os/template/package/template.yaml
name: package
input: package-spec
pattern:                          # what conform() requires, file by file
  pyproject.toml:
    - 'name = "{{ dist }}"'
    - 'requires-python = ">=3.13"'
  "src/{{source}}/__init__.py":
    - '__all__ = ['
```

Under `files/`, anything ending `.jinja` is rendered (Jinja2) and loses the extension; anything else
is copied verbatim. File and folder *names* are templated too (`src/{{module}}/__init__.py.jinja`).
`render(data, types)` writes nothing itself, returns text/bytes per file (write-side effects live in
the action that calls it). `conform(data, source, types)` checks an existing folder still holds what
the `pattern:` requires — this is the mechanism behind "does this generated project still match its
template," reusable for drift-checking later, not just first generation.

### `process` — a checked graph, not a script

```yaml
# sdlc/process/build-application.yaml
name: build-application
input:
  application: application
requires:
  build: build                    # the application's own extension/build.yaml
output:
  artifact: build-artifact
steps:
  - resolve-platform               # → platform, added to context
  - switch: platform.name          # a context value, not an expression
    cases:
      python: [sdlc.python.build-application]
    default:
      - stop: no build process for this platform yet
```

Three step kinds: a bare action/process id (`Call`), `switch`/`cases`/`default` (`Switch`, reads a
literal context value, dispatches to a nested step list), `stop: <message>` (a branch that
deliberately produces nothing, and must not be expected to). `process-cli check` walks the whole
graph before any run is allowed: every input is available in context, every switch reads a value
that will actually exist, every branch that stops produces nothing, the declared output is actually
produced by some path. This is what "prefer a process over calling its actions by hand" (from
[[using-process-os]]) buys you — none of that is re-verified by a human each time.

## The `requires:` pattern — optional per-instance extension data

`sdlc.build-application` requires `build: build` — not part of the `application` schema itself, but
a sibling file, `extension/build.yaml`, that must sit beside a given application's own
`application.yaml`. The comment on `sdlc/schema/build.yaml` spells out why: *"The application
provides it as extension/build.yaml, beside its application.yaml, and a process that builds names it
in requires. The application schema does not change for it."* This is the join point between a
generic core entity (`application`) and a subject-specific concern (*how this one is built*) without
forking or bloating the core schema — directly relevant once we have many platforms and each wants
its own build/deploy/release extension data.

## Cross-scope and cross-repo reuse — confirmed, not speculative

`sdlc/scope.yaml`: `uses: [std]` — a scope names the other scopes (by name) whose definitions it may
reference. In process-os's own workspace, `sdlc` and `std` happen to sit in the same
`processos-workspace/definitions/` folder, but per `CLAUDE.md`: *"A scope can also live in a named
`libraries` entry of the config, in another folder, with an access mode and an optional version, so
`sdlc` and `std` need not sit in the one `definitions` folder."* — meaning a `libraries:` entry in
our own `processos.yaml` can point at a scope folder in a *different repository entirely* (e.g.
`/Users/solo/Projects/workspaces/process-os/processos-workspace/definitions/sdlc`), mark it
`readonly`, optionally pin a version, and our own scope's `uses:` list can then name it.

**This resolves the open question left in `sdlc-kit`'s `CLAUDE.md`** ("once we've confirmed
cross-repo scope reuse is actually how that's meant to work"): it is. We do not have to vendor or
reimplement `sdlc`/`sdlc.python` to build on top of them — we can depend on process-os's own copy
directly as a readonly library, at least for prototyping, and decide later whether to fork/vendor
once we're diverging in earnest. Worth raising with the user as a concrete decision point.

## The `sdlc` → `sdlc.python` shape is our multi-platform template

This is the single most directly reusable structural pattern for our own factory: one dotted
sub-scope per platform, all siblings under `sdlc`. `sdlc.build-application` knows nothing
platform-specific — it resolves the platform, then `switch`-dispatches by `platform.name` to
`sdlc.<platform>.build-application`. Adding a new platform means adding a new `cases:` line plus a
new `sdlc.<platform>` namespace with its own `type/schema/action/template/process` — never touching
the generic process. When we get to Go, TypeScript+Svelte, TypeScript+Next.js, Swift/SwiftUI,
Kotlin, this is very likely the shape to replicate: `sdlc.go`, `sdlc.svelte` or `sdlc.typescript`
(with a framework field the way `sdlc.application.framework` already distinguishes `none`/`typer`
within Python), `sdlc.swift`, `sdlc.kotlin`, etc. — each a sibling namespace, each free to define its
own project/build/artifact shapes, all reached through one generic `build-application` entry point.

Note `sdlc.application.framework` is currently `enum: [none, typer]` — Python-only today, since it's
process-os's own demo/dogfood content, not a general framework registry. We are not bound by that
enum; our own `sdlc`-equivalent scope defines its own.

## Open questions to raise with the user

- Depend on process-os's `sdlc`/`std` via a `libraries:` entry (readonly), or start our own from
  scratch and treat process-os's copy as read-only reference only, never linked? The former is less
  duplication; the latter keeps us fully decoupled from process-os's own demo content evolving under
  us.
- Do we want one `sdlc`-equivalent namespace inside `sbx-sdlc-kit` (mirroring process-os's own
  naming), or does "SBX SDLC Kit" call for a different top-level namespace word here, given
  `sbx-sdlc-kit` is already the scope name itself?
