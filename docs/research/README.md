# Research index — reconnaissance

Last updated: 2026-09-23
tags: research, index, sbx-sdlc-kit

Purpose: understand process-os's own patterns and prior art (shredbx = "sbx v1", sbx.framework =
"sbx v2") before any modeling begins in `sbx-sdlc-kit`'s own scope ("v3"). Nothing is built yet —
this is the "getting to know each other" phase before planning. Two documents read directly
(process-os, the sdlc-factory-vision doc), three produced by background research agents (verified at
folder/manifest/representative-sample depth, not a full code read — treat exhaustiveness claims
accordingly).

## The five documents

1. **[`process-os-patterns.md`](./process-os-patterns.md)** — read directly, primary source. The
   concrete shape of process-os's five definition kinds (type/schema/action/template/process), the
   shell-action contract, the `requires:`-extension pattern, and — confirmed, not speculative —
   cross-repo scope reuse via a `libraries:` config entry.
2. **[`shredbx-codebase-inventory.md`](./shredbx-codebase-inventory.md)** — agent-produced. What
   exists in shredbx by platform: Go (a ~140-package `sbx` core framework + thin client APIs),
   Python (assistant-kit, sbx-next), TypeScript/Svelte (the dominant web stack, a reusable UI
   package family, a games engine), Swift/SwiftUI (one mature app), Kotlin (one mature app + a
   reusable config module), plus infra/scripts/docs.
3. **[`shredbx-sdlc-system.md`](./shredbx-sdlc-system.md)** — agent-produced. shredbx's own
   hand-built SDLC engine: `.sbx/` (framework/runtime/decisions/domain-experts), 9 Claude Code
   skills implementing an FDD1–FDD5 pipeline, 5 role agents, hooks (including a real
   filesystem-scoped edit sandbox), and root governance docs.
4. **Client-specific deep-dive research** — two agent-produced docs verifying a domain-specific-vs-
   already-generic split of one client's largest product, the confirmed "13 capabilities" list and
   where it lives, the exact tech environment of the Next.js app an early bootstrap should match, and
   (follow-up) a verified real-world example of an AI assistant with tool-calling property search,
   real SSE streaming, and a reusable public-chat session/quota pattern — moved to that client's own
   repo to keep client specifics out of sdlc-kit; not duplicated here.
5. **[`reference-repos-index.md`](./reference-repos-index.md)** — the lookup table for every
   reference repo added so far (including `shredbx-workspace-reference`'s 9 older/parallel projects —
   3 real client iOS Swift apps, a Python whisper transcription service, two more Next.js starters).
   Check here before re-scanning any repo from scratch.
6. **[`sbx-framework-inventory.md`](./sbx-framework-inventory.md)** — agent-produced. `sbx.framework`
   ("sbx v2") turns out to be the direct design ancestor of process-os itself — see the headline
   finding below — plus a fully-specified, never-built AI-assistant plan directly relevant to this
   workspace's own future AI Assistant product, and an explicit, considered rejection of
   capability-as-namespace organization.
7. **[`pydantic-ai-session-memory-patterns.md`](./pydantic-ai-session-memory-patterns.md)** —
   agent-produced. PydanticAI session/state management for `agent-framework`: what's already
   correctly shaped (one shared `Agent` + stateless tools + explicit history load/save), two real
   anti-patterns (mutable deps racing under parallel tool calls; an unauthenticated session id is
   not an identity boundary for accumulated facts), and why `StepPersistence`/`Memory` are real but
   premature before a model is actually in the loop.

## Headline findings

shredbx already solved this problem once, a year ago, in Go. `projects/sbx` (source of the `sbx`
binary at the repo root) is a mature, actively-used prior implementation of "model everything,
generate/build/deploy through one governed pipeline" — the same territory process-os and
`sbx-sdlc-kit` are now building, but as a monolithic, subject-aware Go CLI rather than a generic,
subject-agnostic YAML engine. `shredbx/docs/plans/2026-01-24-sdlc-factory-vision.md` (read directly)
is its founding vision doc: a 3-layer model (Idea → Model → Implementation) with full goal→code
traceability and a "Hub" UI meant to replace Figma/VS Code/Vercel/Supabase/CRM.

Worth noting soberly: most of that roadmap (Phases 2–5 — code generation, design studio, AI
integration, deployment automation) is still unchecked eight months later; only Phase 1 (editing
foundation) partially landed. This isn't a dismissal — the parts that *did* ship got used hard for
458 real tasks — but it's a caution worth discussing before `sbx-sdlc-kit` reaches for the same
ceiling early.

## Second headline finding: `sbx.framework` is process-os's actual origin

`sbx.framework` ("sbx v2") is not just another prior attempt — its own roadmap's next milestone was
literally named **"ProcessOS v0.1"**: a fully-specified (1,083-line plan, runnable inline Python) but
never-built AI-assistant/chat product, before the underlying five-primitive engine
(`schema · record · template · action · process` — nearly identical to process-os's own
`type · schema · action · template · process`) got extracted into its own standalone repo instead of
that milestone being built. This is almost certainly the concrete moment behind "I stopped to build
processos and start all over." sbx.framework itself was mature when it paused: 7.5 weeks, 856
commits, TDD-disciplined, 958 passing tests, a working SvelteKit admin IDE.

**A direct tension with the "13 capability scopes" idea**: sbx.framework's own architecture
*explicitly rejected* organizing rows by a flat "capability" namespace — a 2026-08-01 ruling retired
it in favor of a process's own decision tree doing that organizing work instead (verbatim from
`sbx.yml`: *"what a capability was becomes a subprocess"*). The 13-value list is real and confirmed
(`modeling, architecture, implementation, infrastructure, configuration, testing, documentation,
security, automation, deployment, observability, operations, continuity` — source:
`shredbx/.sbx/workspace/capability-roadmap.yml`), but reviving it as 13 capability *scopes* for
sbx-sdlc-kit would reverse a considered decision v2 made and kept for its entire build. Worth
discussing directly, not silently deciding either way — see the open questions below.

`sbx.framework`'s Milestone T plan (`docs/plans/2026-09-15-milestone-t-processos-v0-1.md`) — a
provider-agnostic chat backend with tool-calling bound to process positions and org/project
multi-tenancy — is directly relevant, ready-to-read prior art for this workspace's own future AI
Assistant product, whenever that gets planned.

## What shredbx's system got right — worth deliberately keeping the shape of

- **Template-rendered, dispatch-annotated steps, archived per run.** A step's reusable definition
  (`step.md`: gates, dispatch target, knowledge tags, output contract) is separate from its
  rendered, timestamped, task-specific instance (`steps/{n}.xml`). Directly comparable to how a
  process-os `process` definition vs. a saved `Run` are already split — worth comparing in detail
  once we design our own multi-step SDLC workflow.
- **Adaptive step-skipping from a small live classifier**, with a timestamped, reasoned audit trail
  when it changes mid-task (`scope_answers` + `reclassifications[]`). One step catalogue serves a
  trivial task and a full-stack task without either over- or under-stepping.
- **Two governance strengths, kept deliberately separate**: an LLM-driven audit (`reconcile`'s
  gap/drift framing — requirements are truth, disagreeing code is drift to flag, not adopt) versus a
  deterministic, always-enforced CLI gate (`sbx decision scan --lint --strict`, wired as a blocking
  step in `/merge`). process-os's own `process-cli check` is the same idea in miniature. Worth asking
  where *our* workflow needs a hard deterministic gate versus where an LLM-mediated check suffices.
- **Filesystem-scoped edit sandboxing tied to live task state** — a cheap, deterministic backstop
  against scope creep that doesn't rely on the model remembering to self-police.
- **Decision records born-ratified, with a mandatory documented wrong-pattern and a pre/post gate**
  against re-deciding a live ruling or leaving a superseded one unmarked — stronger ADR discipline
  than most hand-rolled systems bother with, and portable as a habit regardless of tooling.

## What shredbx's system got wrong — named so it isn't repeated

- **A fully speculative, never-implemented governance hook** (`capability-validation.yml`) — a
  complete spec, ready-to-paste settings snippet included, whose prerequisite CLI command was never
  built. A doc describing enforcement that isn't wired up reads as a guarantee and is worse than no
  doc. Lesson: don't write the governance spec before the check exists.
- **Three real guard hooks silently disabled together, undocumented** — a commit that turned off a
  git-mutation-on-main guard, a PR-base guard, and a bypass-mode-proof permission re-check, with a
  message that never mentions them, never revisited since. The layer they were built to backstop is
  now the only layer left. Lesson: disabling a safety control needs to be as visible and deliberate
  as enabling one.
- **Two full generations of a task-record system left in the tree**, and **triplicated per-step
  output storage** (the same content in `task.yml`, `artifacts/*.yml`, and `steps/*.xml`). Neither
  is broken; both are debt nobody prunes.
- **General repo hygiene was never itself a governed step** — root-level clutter, inconsistent
  status-enum values, stub records, and un-pruned ghost worktrees all accumulated because nothing in
  the pipeline owned "keep the workspace clean" as a checked concern. Directly relevant to
  `sbx-sdlc-kit`'s own namespace-discipline goal: discipline needs an owner and a check, not just an
  intention in a doc.
- **The grand unified vision shipped its foundation, not its ceiling.** The parts that got used hard
  for eight months were the narrower, mechanically enforceable pieces (the step engine, task
  records, decisions, sandboxing) — not the ambitious Hub/generation/design-studio layer. Reinforces
  `sbx-sdlc-kit`'s own `CLAUDE.md` instruction to model one entity at a time, discussed and
  confirmed, rather than building ahead of a real need.

## Porting candidates surfaced (full detail in `shredbx-codebase-inventory.md`)

- `projects/sbx/packages/core/go` — richest prior art for SDLC/governance/deployment territory;
  read before designing overlapping process-os entities.
- `projects/sbx/packages/*` Svelte/TS UI library family — small, focused, multi-consumer packages;
  a natural seed for a Svelte package base.
- The games engine/product split (`sbx/packages/games/*` + 5 concrete games) — a clean
  framework-package-vs-thin-product template, not just for games.
- `clients/egor/.../apps/ios/swift` ("SevenTree") — the one mature SwiftUI+CloudKit+fastlane asset;
  a strong seed for a Swift app-starter even independent of porting the product itself.
- `clients/yura/projects/hhh-android/packages/configuration-module` — the one mature, formally
  modeled Kotlin library; both a candidate package and a portable pattern (typed config lifecycle).
- `scripts/infra/bootstrap/*` — a clean, ordered, client-agnostic server-provisioning runbook.
- `docs/domain-example/` and `docs/books/a-practical-guide-to-feature-driven-development` — worked
  FDD examples and a full FDD writeup, cheap to read, likely to save re-deriving modeling
  conventions `sbx-sdlc-kit` will want anyway.

## Open questions for the next planning conversation

1. **Capability scopes: revive or not? RESOLVED (2026-09-24).** Yes, literally — `sdlc.<capability>.…`
   as the outermost namespace, applied lazily (a path is created only when a real definition needs
   it, never pre-scaffolded, which is what avoids `sbx.framework`'s "1000 capability files" failure
   while keeping the real win: `process-cli list --namespace` becomes a genuine "show me everything
   for this capability" query). Ordering within a capability is action-first. Platform position
   (second-level or deeper) is still open, deliberately deferred until the `types` port gives real
   content to test it against — see `docs/proposals/porting-and-modeling-process.md` and the revision
   note atop `docs/proposals/filestructure-and-starter-set.html`.
2. **Scope-reuse decision** — depend on process-os's own `sdlc`/`std` scopes via a readonly
   `libraries:` entry, or start our namespace from scratch and treat process-os's copy as
   reference-only? (raised in `process-os-patterns.md`)
3. **Namespace shape** — does our platform-dispatch namespace mirror process-os's `sdlc` →
   `sdlc.python` shape directly, or does it need a different top-level name, given `sbx-sdlc-kit` is
   already the scope name itself?
4. **What from shredbx's SDLC engine deserves a process-os-native equivalent now vs. later** — e.g.
   is a `process-cli check`-style deterministic gate enough for a while, or do we want a
   `sdlc-sandbox.sh`-style filesystem edit guard from day one?
5. **First real entity to model** — per the near-term direction, likely the Next.js app-starter
   (matching a client's confirmed real-world environment: Next.js 15 + React 19, App Router/Turbopack,
   Tailwind v4, shadcn/ui "new-york", Radix, react-hook-form+zod, next-themes — detail lives in that
   client's own repo, not duplicated here) or the chat-UI experiment itself, once an MVP conversation
   happens. The Svelte UI package family, the Swift app-starter, and the Go core framework remain
   plausible alternate starting points.
6. **The `sbx-next` question** — shredbx already has an early, thin, Python-based next-generation
   attempt at replacing `sbx` (`projects/sbx-next`), aimed at exactly the problem `sbx-sdlc-kit` is
   now solving with process-os instead. Worth a quick look at its `LAYOUT.md`/`IMPORT-BATCH-1.md`
   before assuming process-os supersedes it cleanly, in case it already encodes decisions worth
   knowing about.
7. **Client-app genericization** — a bigger, separate future effort (per the user). The
   domain-vs-generic split researched for one client's largest product is the starting map for that
   whenever it's taken up — that research now lives in that client's own repo.
