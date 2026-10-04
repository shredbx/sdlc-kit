# CLAUDE.md

## What this repo is

**SBX SDLC Kit** — SBX is our workspace identity across repos; this one is the SDLC side of it.
It is a software development factory environment: a place to model Feature-Driven Development
and the workspace/process layer for building digital products of any kind — applications on any
platform, automations, configurations, services, and whatever else comes up — as reusable,
schema-defined entities rather than one-off scripts.

The long-term goal is to import and refactor the existing projects the user already has into this
unified codebase, growing a rich, consistent package base per platform that new products bootstrap
from. Every entity in this repo, and every repeated action in the workflow, gets discussed and
planned with the user before it is modeled and implemented — nothing is added speculatively ahead
of a real need.

## This is a process-os workspace

- Config: `processos.yaml` at repo root.
- Definitions live under `processos-workspace/definitions/`, one folder per scope.
- Current scope: **`sbx-sdlc-kit`** (`processos-workspace/definitions/sbx-sdlc-kit/scope.yaml`). What's
  modeled in it is not tracked here — see "Where things stand" below for how to read it directly.
- Runtime (`processos-workspace/records/`, `output/`, `runs/`) is workspace-local; `output/` and
  `runs/` are git-ignored, `records/` is kept in git.
- CLI: `process-cli`, installed **editable from this repo's own source** —
  `uv tool install --editable projects/products/process-os/process-cli` — not from the upstream
  git remote. `process-cli --version` reports `0.1.0 (source checkout)` when this is wired up
  correctly; if it ever prints a plain version with no `(source checkout)` suffix, something has
  regressed to an external install and needs re-running the command above. All 9
  process-cli/process-kit/process-framework packages resolve to local `file://` sources this way
  (`uv tool list` shows them); edit `platform/python/packages/*`, `platform/python/frameworks/
  process-framework/`, or `projects/products/process-os/process-cli/` directly to change behavior
  — no reinstall needed for pure-Python changes, since editable installs pick them up live.
- MCP server: `process-cli mcp`, wired up by the `process-claude-plugin` plugin — registered from
  this repo's own local marketplace (`projects/products/process-os/.claude-plugin/marketplace.json`,
  added to `.claude/settings.json` as `process-os-local`), **not** the external `process-os`
  marketplace. `enabledPlugins` should show `process-claude-plugin@process-os-local: true` and
  nothing named `@process-os`.
- A git worktree of this repo needs no special setup for either of the above: `process-cli`'s
  config resolution walks upward from the current folder to find `processos.yaml` (confirmed via
  `process-cli --help`), so running it from inside a worktree naturally scopes it to that
  worktree's own `processos-workspace/definitions/` and `records/` — never the main checkout's —
  while still running the one shared editable-installed binary.

**Always reach for the `process-claude-plugin:using-process-os` skill** for any schema, template,
process, or runtime work in this repo — it is the authoritative reference for `process-cli`'s
commands and for how to sequence tool calls correctly. Load it before acting, not from memory.
Key habits it establishes, worth repeating here because they're easy to skip under momentum:

- Run `process-cli check` before the first MCP tool call of a session, and again after editing any
  definition — it is the one thing that explains a broken definition instead of a confusing tool
  error.
- A stopped or failed run is a normal result, not an error — read its `structuredContent` (status,
  context, and for `failed`, the reason on the failed node) before retrying or resuming blind.
- Where a **process** already exists for a goal, call the process, not its steps by hand — a
  process is checked end to end (every input available, every handoff wired, every branch
  productive) in a way that manually chaining actions never is.

## Reference: the process-os repo itself

The source of process-os lives locally at `/Users/solo/Projects/workspaces/process-os` — it is
itself a process-os workspace (dogfooded), with scopes `process-os`, `std`, `sdlc`, and
`sdlc.python` under its own `processos-workspace/definitions/`. Check it when designing our own
types, schemas, actions, processes, and templates — it's the closest thing we have to prior art for
this tool, including its dotted sub-scope convention (`sdlc.python` nested under `sdlc`). Treat it
as a reference for good and bad decisions only: per prior workspace lessons, **never copy its
naming, governance, or workflow wholesale into this repo** — this repo's namespaces are decided
here, on purpose, not inherited.

`sdlc.build-application` / `sdlc.python.build-application` are existing processes there worth
studying before this workspace grows its own equivalents — reuse (via a scope's `uses:` list) is
preferable to redefining the same shape. Cross-repo scope reuse is confirmed to work this way (a
`libraries:` entry in `processos.yaml`, readonly, optionally version-pinned) — see
`docs/research/process-os-patterns.md` — so this is a real option, not just a hope, once we decide
whether to take it.

## Reference: sbx.framework (v2 — where process-os's engine was actually born)

`/Users/solo/Projects/workspaces/sbx.framework` is more than a paused attempt — it's a mature,
7.5-week, TDD-disciplined platform effort (958 passing tests, 1,008 rows, a working SvelteKit admin
IDE) whose own five-primitive engine (`schema · record · template · action · process`) is the direct
design ancestor of process-os. Its roadmap's next milestone was literally named **"ProcessOS v0.1"**
— a fully-specified but never-built AI-assistant/chat product
(`docs/plans/2026-09-15-milestone-t-processos-v0-1.md`) — before the engine itself got extracted into
its own repo (the real process-os) instead of that milestone being built. That extraction is almost
certainly the concrete "stopped to build processos and start over" moment.

**Tension to resolve with the user before modeling capability scopes**: sbx.framework's own
architecture *explicitly rejected* organizing rows by a flat "capability" namespace — a 2026-08-01
ruling retired it in favor of a process's own decision tree doing that organizing work instead
(`sbx.yml`: "what a capability was becomes a subprocess"). The 13-value list from shredbx
(`modeling, architecture, implementation, infrastructure, configuration, testing, documentation,
security, automation, deployment, observability, operations, continuity`) is real and verbatim
(confirmed source: `.sbx/workspace/capability-roadmap.yml` in shredbx), but reviving "13 capability
scopes" for this workspace would reverse a considered decision the v2 attempt made and kept for its
whole build — worth discussing why, and whether it still applies, before modeling it.

`docs/plans/2026-09-15-milestone-t-processos-v0-1.md`'s own design (provider-agnostic chat,
tool-calling bound to process positions, org/project multi-tenancy) is directly relevant prior art
for this workspace's own future AI Assistant product. Treat sbx.framework the same as process-os and
shredbx otherwise: prior art to learn from, never to copy wholesale. Full detail:
`docs/research/sbx-framework-inventory.md`.

## Namespace discipline

This workspace is expected to scale to many packages and frameworks per platform. From the first
entity onward:

- Scope and definition names stay clean, readable, and precise — no placeholder or ambiguous
  names "for now."
- Use dotted nesting for sub-scopes the way `sdlc.python` does upstream, not ad hoc flat names.
- Prefer extending or reusing an existing definition over introducing a near-duplicate with a
  slightly different name.
- When an entity needs new capability, choose one of four mechanisms — extend the schema
  directly, nest into a deeper folder, compose via a field typed as another schema, or extend via
  a process's own `requires:` + a sibling `extension/<name>.yaml` — never a fifth ad hoc, and never
  a pre-built category × dimension grid. See `docs/proposals/schema-evolution-principles.md`.

## Workflow with the user

The user drives, this session builds. For any new entity or any repeated action noticed in the
workflow: discuss intent → plan the shape (type/schema/template/action/process, as needed) →
**get the user's explicit approval on that specific definition** → implement it in
`processos-workspace/definitions/sbx-sdlc-kit/` → `process-cli check` → confirm before moving to the
next one. Approval is per-definition, not a one-time sign-off on a plan that then lets many
definitions get built unattended — the user has been explicit about this. Don't pre-build entities
the user hasn't asked to model yet.

**Before implementing any task, show the before/after file structure of exactly what will be
created or changed** — new files/folders only, nothing else implied — and wait for confirmation
before writing anything. This is how the user verifies the plan is understood correctly and catches
drift before it happens, not after. Deliverables (proposals, write-ups, diagrams) are written as real
files in this repo, never published via the Artifact tool.

**Every completion report has the same shape:** (1) the before/after file tree of what was actually
done, showing new/modified paths only, taken from the real tree, with the evidence for each gate;
(2) the next scope's exact before/after tree; (3) stop and wait for confirmation. A table or prose
summary is not a substitute, and the next tree is shown, never offered.

**We do not build a fully automated, step-catalogue SDLC pipeline** — not shredbx's FDD1–FDD5 step
engine, nor an equivalent of it. shredbx's own year of real use is the reason: a rigid, imposed step
sequence invites finding workarounds to skip steps that were mandated rather than actually required
(see `docs/research/shredbx-sdlc-system.md`). Instead we work **scope-based**: one thing discussed,
then that one part implemented — never a full automated run of a multi-step process end to end. And
we work **dependency-based, not step-based**: an action or process should declare only what it
actually `requires` to execute (process-os's own extension-file pattern is exactly this, see
`docs/research/process-os-patterns.md`) — a missing dependency when we try to run something is
itself the signal that something still needs to be modeled, not a checklist item to march through
regardless of whether it applies.

**Main rule, in the user's own words: schemas and processes exist to deliver a concrete result,
never to build routine for its own sake.** We reach for schema/process definitions because they
organize the work simply, not as an end in themselves. Every schema/process added should be in
service of a task actually in front of us right now — delivering one package, one app starter, one
demo, one import — not built ahead of that need on the theory it will probably be useful.

Alongside building, keep a running retrospective on the `process-os` MCP tool itself — its
efficiency and stability in real use — and flag it when a CLI-first (`process-cli` directly) or
other workflow would serve better than the MCP round-trip, with a concrete recommendation rather
than a vague complaint.

## Skills — invoke them, do not work from memory

The installed plugins carry the practices this repo expects. Before the work a skill covers, call it (Skill tool) and follow it. A skill that applies
and is skipped is a defect. When work is delegated to a subagent, its prompt carries this table and the standing rules below.

| When | Invoke before acting |
|---|---|
| A feature, a behaviour change, a restructure | `superpowers:brainstorming`, then `superpowers:writing-plans` |
| Writing or changing code | `superpowers:test-driven-development` |
| A bug, a failing test, anything unexpected | `superpowers:systematic-debugging` |
| Before saying done / works / passes, and before a commit or PR | `superpowers:verification-before-completion`, then `superpowers:requesting-code-review` |
| Python | `python-development:` `python-design-patterns`, `python-project-structure`, `python-code-style`, `python-type-safety`, `python-configuration`, `python-error-handling`, `python-testing-patterns`; review with `python-anti-patterns`; speed or memory with `python-performance-optimization`; dependencies with `uv-package-manager` |
| A Pydantic AI agent (tools, instructions, history, deps, tests) | `ai:building-pydantic-ai-agents` |
| Prompts and instructions for a model | `llm-application-dev:prompt-engineering-patterns`, then `llm-application-dev:llm-evaluation` to measure them |
| Sign-in, tokens, sessions | `developer-essentials:auth-implementation-patterns` |
| Supabase, Postgres, SQL | `supabase:supabase`, `supabase:supabase-postgres-best-practices`, `developer-essentials:sql-optimization-patterns` |
| Svelte | `svelte-skills:svelte-runes` before any `$effect`, then `svelte-skills:svelte-components`, `svelte-skills:svelte-styling` |
| Tracing, logs, metrics | `python-development:python-observability` |
| Editing any CLAUDE.md | `claude-md-management:claude-md-improver` |
| Two or more independent tasks | `superpowers:dispatching-parallel-agents` |

## Standing rules (each one was broken in real work)

1. **Verify, then claim.** Name what was run, where, and on which address or device. A fact about which service a request hit, what a document says, or
   what a number is comes from config, the raw source or a measurement — never from memory, and never from a page summary (read the raw text). What was
   not checked is said to be unchecked.
2. **Resources first.** We build high-end software: the fewest requests, bytes, tokens and CPU, short clean code, no cache, engine or abstraction
   without a measured need. Every tunable is a setting, never a literal inside logic.
3. **Names are the scope.** A folder or file is named for the concept it holds in the thing being built (for an agent: config, instructions, tools, deps,
   context, services, output validators). Never `utils`, `common`, `helpers`, `state`, `search`, `misc`. A name that needs explaining is the wrong name.
4. **Evidence or silence.** A suggestion carries its source and how it was checked; one with neither is dropped, not offered. Do not ask a question whose
   answer changes nothing.
5. **Smallest change that does the job.** No schema change, no unrequested refactor, no extra feature.
6. **Commit only when asked, in small commits.** Show the before/after tree before changing a structure, then wait for the go.

## Memory (MCP `memory` server)

When writing to the knowledge graph memory server, tag every entity/observation with a clear
topic tag plus a timestamp. When new information supersedes an old observation on the same topic,
remove or update the stale one rather than letting both stand — the graph should always reflect
the current understanding of a topic, not its full history of revisions.

## Where things stand

This section is deliberately not a status dump — the workspace's own artifacts are the source of
truth, and they don't go stale the way prose here does. To find out what's actually modeled or
decided, read them directly instead of this file:

- **What's modeled** (types, schemas, actions, templates, processes): `process-cli list` (or
  `--long` for descriptions), scoped with `--scope sbx-sdlc-kit` if needed.
- **Governance — the capability namespace list and the decision log**:
  `processos-workspace/records/sbx-sdlc-kit/architecture/{capability,decision}/` — `process-cli list
  records sbx-sdlc-kit/architecture/capability` (or `.../decision`) to see what's there,
  `process-cli show record <path>` to read one. Check the decision log before re-deciding something
  — each topic has its own folder, one file per call made about it over time.
- **Research** (prior-art inventories of process-os, shredbx, sbx.framework): `docs/research/README.md`
  indexes all of it.
- **Proposals** (design write-ups pending or past approval): `docs/proposals/`.
- **Plans** (task-by-task implementation logs, one per milestone): `docs/plans/`.

**As of 2026-09-24**: all seven `process-os` `process-kit` packages (`types`, `schema`, `action`,
`filesystem`, `config`, `template`, `process`) are ported verbatim into
`platform/python/packages/`, each proven against its own ported test suite, ruff/mypy-clean, and
wired into `.github/workflows/python-ci.yml` as its own step — see `docs/plans/2026-09-24-*-milestone-{2..8}.md`
for the per-package detail (what was copied, what ruff/mypy found, the coverage-gap check each one
got). `platform/python/pyproject.toml`'s workspace covers all seven; nothing in `process-kit` is
left unported. A combined `pytest packages/` across all of them fails at collection (shared relative
test paths collide) — expected, not a bug, which is exactly why CI runs one `pytest` step per
package.

**As of 2026-09-25**: two more milestones landed on top of that port. (1) The `process-framework`
package (an 8th package — the orchestration layer `process-cli` itself is built on) is ported into
`platform/python/frameworks/process-framework/`, and `process-cli`'s own source is imported at
`projects/products/process-os/process-cli/` — see `docs/plans/2026-09-25-definitions-library-milestone-9.md`,
`docs/plans/2026-09-25-process-framework-milestone-10.md`, and
`docs/plans/2026-09-25-process-os-apps-import-milestones-11-12.md`.
`process-cli` is now installed **editable from this repo's own source**, not the upstream git
remote (`process-cli --version` → `0.1.0 (source checkout)` confirms it), and its Claude Code
plugin is registered from a local marketplace (`process-claude-plugin@process-os-local`) instead of
the external one — see this file's own "This is a process-os workspace" section above for the
exact commands. (2) The `infrastructure` capability (one of the 13 already-locked capabilities, see
"Namespace discipline") is modeled — `service`/`service-bundle` schemas, a `bundle-compose`
template, `render-bundle`/`run-bundle` actions, a `bootstrap-bundle` process — and its first real
deliverable, a postgres + pgAdmin dev bundle, is built, tested end to end (`process-cli run
sbx-sdlc-kit.infrastructure.bootstrap-bundle`), and running at `projects/services/postgres-dev/`
(own `README.md`/`Makefile` for start/stop). Design: `docs/proposals/infrastructure-services-design.md`.

**As of 2026-09-27**: the "model a real product" step below was picked up — the product is `bos`, a constructor
platform for websites/apps, decomposed rung by rung from a running production app (the first consumer, mounted
at `consumers/clients/bestie/bestierealestate`; sdlc-kit itself holds none of that consumer's own information —
see `feedback_no-client-info-in-sdlc-kit` in project memory). Design: `docs/proposals/{bos-system-design,
bos-constructor,bos-app-roadmap}.md`; decisions D1 to D26 are recorded as prose inside those three
documents (not in `architecture/decision/`, which is this workspace's own governance log, a separate
namespace). **Milestone M0 is done**: the platform's Go
and Svelte packages are ported (Scopes 1 to 6e), the unit/package record model exists (Scopes U0 to U2c, one
record + rendered README per package), and the `bos` frameworks themselves are extracted from the running
original — `bos-go` (settings, fixed middleware chain, health/root routes, graceful shutdown, the `bos` CLI) and
`bos-svelte` (security headers, the same-origin `/api` pass-through, a placeholder home) — with the first
consumer's example app running both natively (`make dev`) and as a docker bundle (`make bundle-up`), each gated
offline and against the original's own golden files. Milestone plans: `docs/plans/2026-09-2[4-7]-*.md`,
especially the M0.x series for the frameworks and the app bundle. A resumption note for whoever picks this up in
a fresh worktree: `docs/plans/2026-09-27-handoff-next-worktree.md`.

Next step: **M1 — branding and tokens** (colors, logo, assets, fonts), per the roadmap's M1 section. Nothing about
M1 is modeled yet as of this note — pick it up fresh with the user rather than assuming scope from here.

Separately, `docs/proposals/product-example-scaffold.md` (process-os reuse vs. adapt, stdio-vs-HTTP MCP, no
domain picked) and a brainstorm start on postgres vector-search support (pgvector), adding redis to the
infrastructure capability, and researching DBOS for durable Pydantic-AI workflows remain unmodeled reconnaissance,
unrelated to the `bos` track.
