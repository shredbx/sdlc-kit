# shredbx's Bespoke SDLC/Workflow System — Research Inventory

Last updated: 2026-09-23
tags: shredbx, sdlc-system, research

> Scope note: this is a **descriptive** inventory of `/Users/solo/Projects/workspaces/shredbx`'s
> hand-built SDLC tooling, written to compare against process-os before any porting decisions are
> made. It does not recommend what to keep or discard — that's a follow-up discussion. shredbx is
> treated as strictly read-only source material throughout.

---

## 1. `.sbx/` — the bespoke workspace/task system

`.sbx/` is shredbx's home-grown answer to "what process-os calls a workspace." Its own `README.md`
(14KB, written in first person, present-progressive — "we will be introducing…") describes it as an
in-progress design for a generic, YAML-schema-driven SDLC framework, explicitly intended to later
ship as its own Claude Code plugin. It borrows vocabulary from FDD, XP, Scrum, Kanban, Crystal
Clear, ETVX and KISS all at once. The directory is organized as:

```
.sbx/
  .framework/     # platform-agnostic engine: schemas, knowledge, the FDD step-template pipeline
  .runtime/        # everything that changes per-session: tasks, active pointers, logs, bin, ssh, vault
  articles/devlog/ # auto-generated (git-cliff) daily changelogs
  decisions/       # 355 ADR-style decision records (see below)
  domain-experts/  # ~30 instantiated "research an external technology" workspaces
  ports/           # machine-local port allocation registry
  tasks/           # an EARLIER, abandoned task-record prototype (8 records, March 2026 only)
  workspace/       # the live domain model of the workspace itself (clients, projects, fabric, adapters…)
  README.md
```

### `.framework/` — the engine

- `core/` — the metamodel: `types/` (adapter, catalogue, dictionary, function, money, rule, image,
  video, primitive…), `layers/` (the 4 FDD software layers: `pd` = domain/business rules, `md` =
  storage/mapper layer, `si` = service/system integration, `ui` = interface), `platforms/` (go,
  kotlin, python, sveltekit), `checks/` (analytics, cache, database, external-api, proxy).
- `knowledge/` — the reusable content library the step engine injects into prompts: `capabilities/`
  (4 files: run/build/deploy/test — see below), `guidelines/` (subdirs per platform + a general
  `architecture/`), `rules/` (52 files — e.g. `defensive-programming`, `coding-standard`,
  `commit-message-standard`, `fdd-two-week-iteration`), `roles/` (25), `personalities/` (21 —
  precise, creative, critical, pragmatic, rigorous…), `smells/` (36), `design-patterns/` (10),
  `task-templates/` (8).
- `lifecycle/sdlc/` — **the actual step engine**: 25 numbered directories (`010-FDD1.G` …
  `260-FDD5.CLOSE`), each holding one `step.md`. This file is the single source of truth for one
  SDLC step: YAML frontmatter (`fdd` id, `gate.requires`/`produces`, `agent`/`role`/`personality`/
  `skill`/`command` to dispatch, `knowledge.required`/`available` tags, `output.path`/`fields`,
  `gates` — plain-English exit predicates, `scope_paths`, `commands.discover`/`build`) followed by a
  Markdown prompt body with `{template}` placeholders (feature description, application, scope…).

  **`capabilities/` doc format** (`run.yml`, `build.yml`, `deploy.yml`, `test.yml`, `check.yml`):
  each documents one cross-cutting automation capability across three tiers (workspace / project /
  app) — what ENV/TARGET/PROJECT params it takes, what each tier does, the `app_contract` (required
  Makefile targets every app must expose), and the `sbx` entry point. The hard rule: `sbx` CLI calls
  exist **only** in the workspace-tier Makefiles; app Makefiles must be portable-by-copy with zero
  `sbx` dependency.

### `.runtime/` — where the mechanism actually runs

This is the part worth understanding in depth, since it's the part most parallel to a process-os
"run."

- **The step-rendering pipeline.** Running `sbx sdlc next` takes the active task's live state +
  the matching `.framework/lifecycle/sdlc/{NNN-FDDx.y}/step.md` template + the relevant
  `knowledge/*` hint docs, and renders them into
  `.sbx/.runtime/tasks/{id}/steps/{ordinal}.xml` — a literal, ready-to-dispatch prompt. Example
  structure of a rendered step (`FDD4.PD.TEST_RED`, task `2609-007`, `steps/100.xml`):
  ```xml
  <step name="Problem Domain - Red Tests" fdd="FDD4.PD.TEST_RED" position="10/25" type="creative">
    <knowledge brief="true"> <!-- ~10 hint docs: rules, guidelines, role, personality --> </knowledge>
    <fetch-hint>sbx knowledge get &lt;name&gt; to load full content</fetch-hint>
    <task-context> <!-- the ENTIRE task.yml dumped inline: goal, stories, scenarios, steps[] history --> </task-context>
    <output-contract>artifact: .../artifacts/test-red-result.yml, required_fields: test_count, failing_tests, ...</output-contract>
    <gates>- Tests must fail before this step completes (red phase) ...</gates>
    <dispatch agent="backend" role="tester" personality="precise" skill="implementation" command="implement" />
    <prompt> <!-- the full instructional prompt, with mandatory-test-coverage rules, TDD steps, exit gate --> </prompt>
  </step>
  ```
  So each SDLC step is not hand-invoked by name — it's **assembled** on demand from a template +
  live task state + a knowledge-tag lookup, and the assembled result is archived to disk as the
  literal record of "what was asked" for that step. `rendered-refs.yml` alongside it lists every
  knowledge artifact (guideline/personality/role/rule) touched by the task so far.

- **A real task record — what a "task" actually contains.** Task `2609-007` (7tree FAQ page) is a
  fully worked example. It lives at `.sbx/.runtime/tasks/2609-007/` with:
  - `task.yml` — one large YAML file that accumulates **everything** across the task's life: `id`,
    `description`, `application`, `scope[]` (path prefixes — enforced, see hooks §5), `scope_answers`
    (a small classifier: `go_package`/`svelte_route`/`persistence`/`server_integration`/
    `app_or_infra` booleans used to decide which of the 25 canonical steps actually apply),
    `goal{ref,title,summary}`, `stories[]` (each with `scenarios[]` in Given/When/Then form, typed
    positive/negative), `actors[]`, `model.entities[]`, `tests.strategy`, `implementation.files_changed[]`,
    `fixtures[]`, `test_cases[]` (each tied to a `scenario_ref`), `steps[]` (one entry per FDD
    sub-step: `fdd`, `status`, `started_at`/`completed_at`, and a free-text `note` — this is the
    literal audit trail of the whole task), and then, appended directly into the *same* file,
    the full structured output of every step that produces one: `build{}`, `coverage{}`,
    `page-model{}`, `usage-spec` (markdown blob), `visual{}`, `review{}` (governance rating with
    self vs reviewer scores per axis, a `requirements_trace` mapping every scenario to its proof,
    checklists for architecture/functional/security/code-quality).
  - `execution-state.yml` — a second, smaller file tracking `scope_classification`,
    `target_level`, and a `reclassifications[]` audit trail (e.g. this task was reclassified
    `step-template` → `enhancement` → `bug-fix` at steps 7 and 9, each with a timestamped reason).
  - `artifacts/*.yml` — the same structured outputs (`review-report.yml`, `build-result.yml`,
    `page-model.yml`, `coverage-audit.yml`, `visual-regression.yml`, `implementation-summary.yml`,
    `cross-result.yml`, `usage-spec.md`) **also** written as standalone files.
  - `steps/*.xml` + `steps/rendered-refs.yml` — the rendered prompts described above, kept as
    history, one per completed/active step.

  **Redundancy note**: the same governance content (e.g. the review scores, the page model) ends
  up persisted in three places — inline in `task.yml`, standalone in `artifacts/`, and re-embedded
  inside the relevant `steps/*.xml` at render time. Functionally coherent (each serves a different
  reader: `task.yml` is queried by `sbx task show`, `artifacts/*` is what downstream steps consume,
  `steps/*.xml` is the immutable prompt-as-sent record) but it's a lot of duplicated bytes per task
  for 458 tasks.

  **Status distribution** across all `.sbx/.runtime/tasks/*/task.yml` (`status:` field): 388
  `completed`, 39 `active`, 1 `pending`, 1 `deferred`, 1 `closed`, and 1 spelled `complete`
  (singular) instead of `completed` — a small inconsistency in what should be a closed enum.
  A few task folders (e.g. `2603-001`) contain only an empty `task.yml` and an `artifacts/` dir —
  stub/broken records left over from early use.

- **Adaptive pipeline.** The 25 canonical FDD steps are not all run for every task — a task's
  `scope_answers` (computed at `sbx sdlc init`) and later `reclassifications[]` (computed live, at
  specific steps, with a stated reason) decide which steps are skipped ("auto-skipped: out of scope
  (Decision #0291)" appears repeatedly for `FDD4.MD`/`FDD4.SI` on tasks with no persistence/server
  layer). This is the mechanism that lets a trivial static-content-page task and a full-stack
  feature task share one step catalogue without the small task waste-stepping through
  persistence/integration steps that don't apply to it.

- **Two task-record generations.** `.sbx/tasks/` (top-level, *not* under `.runtime/`) is an earlier
  prototype: only 8 sparse records (`2603-001`..`2603-013`, all dated March 3 and March 23, 2026),
  a flatter ETVX-style schema (`checklist`, `entry[]`/`exit[]` typed conditions, `decision_ref`,
  `nfr_constraints[]`, `goal_ref`, `phase: P1`, `blocks[]`). Its own `registry.yml` says "All M0-M11
  milestone tasks archived… No active runtime tasks — all milestones complete," and lists
  `tasks: []`. This generation was abandoned in favor of the much richer FDD1–FDD5 step-engine +
  rendered-XML system under `.runtime/tasks/` described above — a superseded prototype left in the
  tree rather than deleted.

- Other `.runtime/` contents: `active/` (129 entries — one file per branch, pointing at that
  branch's active task id; this is what the `sdlc-sandbox.sh` hook reads to scope edits, see §5),
  `bin/` (the compiled `sbx` binary lives here too, `.sbx/.runtime/bin/sbx`, distinct from the
  38MB one at repo root — see §8), `bootstrap*/`, `deploy-logs/`, `env/`, `run/` (157 entries —
  per-dev-server state), `ssh/`, `vault-tokens/` (1Password service-account tokens, explicitly
  denied to `Read`/`Edit` in `.claude/settings.json`).

### `decisions/` — 355 ADR-style records with a real lifecycle

One folder per decision (`NNNN-kebab-title/decision.yml`), each with `status` living **inside** the
file (`decided` → `superseded`/`deprecated` → `archived`), never encoded via directory moves except
optionally into a cold-tier `_archive/`/`_deprecated/` folder. Decisions are "born ratified" — there
is no proposal tier; a decision record is created already `decided`. Every record is required to
carry a documented `wrong_pattern` (what NOT to do and why, with a source) alongside the
recommendation, plus three mandatory `perspectives` (inductive, deductive, cybersecurity — the
`/decision` command runs these as parallel subagents, one per domain expert whose scope overlaps).
`sbx decision scan --lint --strict` is a real, wired-in mechanical gate (parse validity, duplicate
IDs, non-canonical status values, dangling supersede pointers) that runs as a **blocking check in
`/merge`** (Phase 1, Step 5.5) — one of the few fully automated, always-enforced invariants in this
whole system (contrast with the still-unbuilt `capability-validation.yml`, §5).

Example of the lifecycle working as intended: decision `0130` (1Password-in-orchestrator credential
strategy) is `status: archived` with `archived_reason: "Superseded by Decision #0248. SA token in
orchestrator is a critical security antipattern…"` — a real correction, recorded and pointed
forward, not silently dropped.

### `domain-experts/` — external-technology research workspaces

A reusable four-pipeline pattern (`create` → `research` → `validate` → `extend`) for building a
persistent knowledge base about one external technology or concern. ~30 instances exist (`svelte`,
`golang-api`, `supabase`, `clerk`, `r2-cloudflare`, `webkit-safari`, `1password`, `pgdump`, `seo`,
`cybersecurity`, `metamodeling`, `image-processing-cli`, `nextjs-react-19`, etc.), each a folder
with `domain-expert.yml` (instance + `sources[]`), `study-document.yml` (the structured output), and
often `notes/`/`knowledge/`. Decisions and SDLC steps reference these as a knowledge source (e.g.
`/decision`'s cybersecurity subagent is told to read `.sbx/domain-experts/cybersecurity/`).

### `ports/`, `workspace/`

`ports/registry.yml` is a machine-local port-allocation ledger (ranges per category: database,
external, internal, mcp, mobile, services; explicitly excluded from commits per root `README.md`).
`workspace/` is the live domain model of the workspace itself — `clients/` (per-client project
trees: andrei, bestie, egor, wanflo, yura, internal), `brands/`, `adapters/` (git, docker, image,
op/1Password, pgdump, port, package, client, brand, sbx, lab), `infrastructure/` (servers,
environments, deployments), `fabric/` — a small business-configuration framework (`ARCHITECTURE.md`,
`RECIPE.md`, `fabric.yml` + `modules/*.yml`: faq, contacts, media, canvas, documents, calendar,
identity, automation) that CLAUDE.md calls out as the target for "build a new product by
configuration, not code" — and `requirements/` (workspace-level NFR/functional requirements
cascade, `override_policy: "projects may add stricter requirements, never weaker"`).

---

## 2. `.claude/skills/*/SKILL.md` — the SDLC workflow steps

Nine skills exist. Each declares `allowed-tools` in frontmatter (a real capability boundary, not
just documentation — e.g. `review` and `reconcile`'s read path get no `Write`/`Edit`).

| Skill | Purpose | Core workflow |
|---|---|---|
| **requirements** | Discover-or-create goals/stories/scenarios/actors for the active SDLC task. | Load task → discover existing artifacts (`sbx goal/userstory/scenario list`) → gap-analyze → propose (wait for approval) → register (`sbx task add-*`) → verify (`sbx task show`). |
| **modeling** | Domain modeling via the **three-map architecture** (Decision #0019): Entities (containment hierarchy, cascade-delete with parent), Attributes (value-bound properties, parent lifecycle), References (associations, independent lifecycle). | Extract nouns/adjectives/verbs from stories → classify each noun via a 4-question decision table (own lifecycle? exists without parent? fixed-set value? unique id?) → define attributes with named types (never raw strings) → register (`sbx task add-model`). |
| **implementation** | TDD code implementation: Test Red → Implement → Test Green → Refactor → Build Clean. | Load task + coding-standard → check existing patterns first (reuse over invention) → red phase (FDD4.PD.TEST_RED) → green phase (FDD4.PD.IMPLEMENT/SI/UI) → `sbx build && sbx test && sbx check` must all pass before `sbx sdlc complete`. |
| **testing** | Test strategy: "scenarios ARE the tests." | Extract every BDD scenario → map 1:1 to a test case in a table → design fixtures from domain-model entities (never random data) → write tests that must fail first → register plan. |
| **review** | Read-only review: start from requirements, not code. | Load `task.yml` as source of truth → build a requirements checklist → trace each story/scenario to implementing code + a test → detect **drift** (git-diff'd file serves no story) and **scope violation** (touches files outside task scope) → check a fixed governance table (named types, no CSS copy-paste, no clipping, brand colors, right-aligned CTAs, OWASP) → report with file:line findings, severity-tagged (critical/warning/info). |
| **reconcile** | Standalone (no active SDLC task needed) requirements↔implementation reconciliation for one package/module/app/component. | Pick exactly ONE target, state it back → discover structure (`sbx describe`, `sbx entities list`) → gather three requirement classes (functional / NFR / system) → reconcile against code, recording **gaps** (spec, no code) and **drift** (code, no spec, or code that contradicts spec) → write requirements *by reference* into the target's own YAML (`requirements: [{id, rule, rationale}]`) → validate → mark `requirements_status: validated` so future tasks can skip re-gathering. |
| **deploy** | Governed web-app deploy (mirror sync → env audit → Dokploy push → migrate → health verify). | See §4 (its `SKILL.md` body is literally `cat .claude/commands/deploy.md`, tail-stripped of frontmatter — the skill and the command are the same document). |
| **merge** | Clean branch → PR → main workflow with zero leftovers. | Same pattern: `SKILL.md` shells out `cat .claude/commands/merge.md`. See §4 for the full phase breakdown. |
| **decision** | Source-backed decision-making for any question, at a chosen depth (shallow/standard/deep/immediate). | Same pattern: `SKILL.md` shells out `cat .claude/commands/decision.md`. See §4. |

**Chaining.** requirements → modeling → testing → implementation is the FDD1→FDD2→FDD3→FDD4 spine;
`review` sits at FDD4.REVIEW (self-review before build) and again conceptually gates `merge`;
`reconcile` is explicitly **outside** this chain — it's the tool for gathering/validating
requirements for a target that has no active SDLC task, or for auditing spec-vs-code drift
independently. `deploy` and `merge` are terminal, human-gated steps (both have hard "stop and ask
the user" triggers baked in, never fully autonomous to production).

---

## 3. `.claude/agents/*.md` — the persona layer

Five agents, each with `allowed-tools` scoped to its role (e.g. `reviewer` has no `Write`/`Edit` at
all — genuinely read-only, not just instructed to be).

| Agent | Role | Explicitly NOT for |
|---|---|---|
| **backend** | Go implementation: packages, Chi router endpoints, adapters, infra, Docker, build verification. TDD-first, minimal code, never raw `go`/`git` (always the `sbx` wrapper). | Svelte/CSS (→ frontend), requirements (→ planner), review (→ reviewer) |
| **frontend** | Svelte 5 / SvelteKit implementation. Component-first (core-ui library before new components), hard design rules baked into the agent file itself (only brand red `#e5392b`, CTAs always right-aligned, narratives never `max-width`-clipped, no inline `.replace()`/`.trim()` in templates, Playwright screenshot verification before done). | Go/API (→ backend) |
| **planner** | Requirements + domain modeling only — goals, stories, scenarios, actors, entity discovery. "Discover before create" philosophy, identical in spirit to the `requirements`/`modeling` skills. | Any implementation code |
| **reviewer** | Read-only review against requirements; drift/governance/OWASP/SOLID. Structurally the same 5-phase process as the `review` skill (load requirements → trace → detect drift → governance checklist → report). | Writing or editing any file |
| **tester** | Test strategy + test code only (FDD3 + FDD4.PD.TEST_RED). "Scenarios are the tests" philosophy, same as the `testing` skill. | Implementation code |

**Relationship to the step engine (§1):** these five agent files are static, hand-written personas —
but the *actual* per-step dispatch is computed dynamically. The rendered `steps/*.xml` for task
`2609-007`'s `FDD4.PD.TEST_RED` step carried `<dispatch agent="backend" role="tester"
personality="precise" skill="implementation" command="implement" />` — i.e. the **backend** agent
persona was dispatched but handed the **tester** role, the **precise** personality, and pointed at
the **implementation** skill for that specific step, all decided by that step's `step.md`
frontmatter (§1), not hardcoded in `backend.md` itself. The agent files describe the default/typical
shape of each persona; the live step template is what actually parameterizes any given dispatch.

---

## 4. `.claude/commands/*.md` — thin(ner) wrappers

Ten commands (`decision`, `deploy`, `implement`, `merge`, `modeling`, `reconcile`, `requirements`,
`review`, `test`, `worktree`). Two distinct patterns:

**Pattern A — task-bound primers** (`requirements`, `modeling`, `implement`, `test`, `review`):
each is a short wrapper that shells out `sbx sdlc status` and `sbx task show` inline (via `!` bash
substitution in the command markdown itself, so the live task state is injected before the model
even starts reasoning), sometimes pre-loads one relevant knowledge doc (`sbx knowledge get
coding-standard`/`modeling-standard`/`bdd-scenario-structure`), then says "follow the `<skill>`
skill workflow" with a condensed numbered restatement of that skill's steps. These commands add
*nothing* procedurally beyond the matching skill — they exist purely to pre-fetch live task context
into the prompt before the skill's reasoning starts.

**Pattern B — commands that ARE the skill** (`decision`, `deploy`, `merge`): for these three, the
skill file's entire body is `!`cat .claude/commands/{name}.md`` — there is exactly one authored
document, and the "skill" is a re-export of the "command" (or vice versa; the command is the
canonical file). `deploy.md` and `merge.md` are the two most operationally hardened documents in
the whole `.claude/` tree — see the mechanism excerpts below.

**`reconcile.md`** is a third variant: a 9-line pointer ("Invoke the `reconcile` skill for:
`$ARGUMENTS`. Follow its workflow exactly...") — the skill does all the work; the command exists
only so `/reconcile <target>` is typeable, with no task-context pre-fetch (correct, since reconcile
is explicitly *not* SDLC-task-bound).

**`worktree.md`** is unrelated to the FDD chain — it's a dev-environment setup command:
`sbx ports worktree --project <name>` allocates unique `WEB_PORT`/`API_PORT` for the worktree
(offset from the project's base ports) and writes `.sbx/.worktree`, which `sbx run` then reads to
auto-resolve the right port. Refuses to run on `main` (main uses Docker via `sbx infra run` instead).

**Deploy mechanism (from `deploy.md`), concretely:**
Web apps deploy via a **mirror**: the monorepo is never a Dokploy source; `sbx external sync
<project>` vendors/scrubs/pushes to an external GitHub repo, and Dokploy builds compose-from-git
against *that* repo's `main`. The command's reason for existing is a named recurring failure mode:
"deploys kept going in circles" because a newly introduced env var (their example: a Google Maps
key) never made it into the app's `.env.example`, so it silently never got pushed to Dokploy. The
"hard gate" phase (Phase 2) diffs env vars actually referenced in source code against
`.env.example`, then probes every `op://` vault ref against the *target* environment's vault
specifically (documented known bug: prod resolution can string-replace a dev ref rather than using
`vault_ref_prod`, so it must never be assumed correct). Health verification (Phase 6) explicitly
distrusts a single `status: done` read — it requires proving a *new* build is live via some marker
only the new release has (a real story is recorded: a stale build's `done` status was
mistaken for the new one until favicons were compared).

**Merge mechanism (from `merge.md`), concretely:**
A strict, numbered pre-flight audit (uncommitted changes, `.gitignore`-blocked tracked files, active
SDLC task count, other unmerged branches, `sbx decision scan --lint --strict` as a blocking gate,
`scripts/checks/check-context-integrity.sh`) before touching anything, followed by commit → push →
PR-create → **PR-merge with a hard-coded number-read-back rule**: `sbx git pr merge <n>` is
documented to *ignore* its own positional argument and merge whatever PR belongs to the current
branch (a named footgun, `feedback_sbx_git_pr_number_gotcha`), so the actual merge path goes through
a Makefile target (`pr-merge NUMBER=<n>`) with a mandatory `pr-view NUMBER=<n>` title/branch
confirmation immediately before it. Two real incidents are cited by PR number (`#568`, `#640` — the
latter merged the *wrong* PR once because a number was assumed rather than read back) as the reason
these two steps must never be batched in the same turn.

**Decision mechanism (from `decision.md`):** a "PRE-GATE" (§1.5, mandatory) runs *before* any new
research — `sbx decision related "<terms>"` (active rulings) and the same with `--deprecated`
(rejected/parked rulings) — specifically to stop re-deciding a live ruling or re-proposing something
already rejected (named as "the #0119 class the audit found"). A "POST-GATE" in the same turn marks
any conflicting old decision `superseded`/`deprecated` — "never leave a known conflict unmarked."

---

## 5. `.claude/hooks/` — the automated guardrails (and what got turned off)

**Currently wired into `.claude/settings.json`** (four hooks, all active):

| Event | Hook | What it does |
|---|---|---|
| `SessionStart` | `sdlc-on-session-start.sh` | Runs `sbx sdlc status`; if a task is active, injects its status + a `sbx sdlc next --peek --header-only` dispatch hint into context. If none, injects the `sbx sdlc init` usage line. Explicitly documented to emit *only* dynamic state, not governance prose (a prior version apparently duplicated content already in `CLAUDE.md`, per its own comment). |
| `PreToolUse` (`Write`\|`Edit`) | `sdlc-sandbox.sh` | The real enforcement mechanism behind "every change goes through `sbx sdlc`." Blocks any Write/Edit unless: the path is under `.claude/`, `docs/`, `memory/`, or is a root-level file (always allowed); OR the current git branch has an active-task pointer file at `.sbx/.runtime/active/<branch>` whose `task.yml` declares a `scope:` prefix matching the edited path; OR the task's `current-step-scope.json` extends that scope; OR the task's own `artifacts/` dir. No active task on a scoped branch → hard block with the exact `sbx sdlc init` command to run. |
| `Stop` + `SubagentStop` | `inductive-trace-reminder.sh` | Non-blocking (always exit 0). If any file changed this turn, prints a reminder to run the 5-step "inductive trace" from `CLAUDE.md` (git status/diff → does each change serve the current topic → cross-check drift → knowledge/guideline check → active-task-scope check) before claiming "done." |

**Documented but never wired: `capability-validation.yml`.** This file is explicitly labeled
`status: spec-only` / prerequisite CLI command `status: not-started`, targeting a future `sbx
validate capabilities` command that would gate `Stop`/`SubagentStop` on critical capability
descriptors (modeling/architecture/implementation/testing) being complete. **Confirmed dead**: a
grep of the actual Go source (`projects/sbx/apps/sbx/go`, `projects/sbx/packages/core/go`) for
`validate capabilities` / `ValidateCapabilities` returns zero hits, and the settings.json hooks list
has no entry for it. It's a fully-designed spec (complete with a `critical`/`warning` capability
classification and an `override_mechanism` for promoting a capability to critical via project
requirements) that was never implemented or activated.

**Disabled: `.claude/hooks/off/*` (three PreToolUse guard hooks, all moved together in one commit).**

- `git-no-mutation-on-main.sh` — blocked destructive git ops (`commit`, `cherry-pick`, `revert`,
  `rebase`, `merge`, `reset --hard/--keep/--merge`, any push targeting `main`, any `--force` push)
  whenever `HEAD` was on `main`, regardless of whether the call came from the orchestrator, a
  subagent, or a human.
- `pr-base-guard.sh` — blocked raw `gh pr create`/`merge`/`list`, forcing everything through
  `sbx git pr *` (which enforces `--base main` and merge-commit-only strategy).
- `skip-permissions-deny-revalidate.sh` (header names itself "Bash Deny Re-Validate," referencing
  a *different* filename, `bash-deny-revalidate.sh`, inside its own comments — a small naming
  inconsistency) — re-enforced the `settings.json` deny list (`pnpm`, `npm`, `npx`, `node`, `go`,
  `make`, `docker`, `psql`, `mysql`, `redis-cli`, `alembic`, `pytest`) at the **hook** level, because
  `settings.json` deny rules are documented to be bypassable when Claude Code is launched with
  `--dangerously-skip-permissions`, and PreToolUse hooks are not.

  All three were moved to `off/` in a single commit, **`ac7d97a2` ("chore: add Makefile for sbx
  build/install, simplify session hook", 2026-05-05)**. The commit message only explains an
  unrelated change (removing auto-build logic from the session-start hook); it gives no stated
  reason for disabling the three guard hooks. `git log --follow` on each shows no later commit
  re-enabling or explaining them — they still sit in `off/` today. Meanwhile
  `.claude/settings.json`'s declarative `permissions.deny` list still contains the equivalent raw
  git/npm/go/docker denials it was designed to double up — meaning the *only* enforcement left for
  "never mutate main directly" and "never call raw `gh pr`" is the declarative permission system,
  which is exactly the layer the disabled hooks existed to backstop against
  `--dangerously-skip-permissions`. This reads as an undocumented regression in enforcement
  strength, not a deliberate, recorded decision.

---

## 6. Root docs

- **`CLAUDE.md`** (148 lines) — the governance contract. Opens with a deliberate "drift tripwire":
  every response must address the user by name ("Andrei"); omitting it is defined as a signal that
  session context/governance has been lost, triggering a reload. The core is a single **"HARD
  INVARIANTS"** table (16 rows: WHEN / DO / NEVER / CHECK) covering implementation-must-go-through-
  SDLC, push/PR/merge gating, the mirror-only deploy model, `sbx` wrapper enforcement, secrets via
  `sbx vault`, "claiming done" verification discipline, generated-file handling, decision pre/post
  gates, and scope discipline — stated as the single rule table that wins over any other doc if
  they conflict. Below it, a **striking system**: violations get logged with a date and a memory-key
  pointer (currently 1 entry, a shipped-fake-feature incident), counted toward "10 → model
  quarantine," removable only by the user. Then session-startup commands, the SDLC
  plan→init→walk loop (mirrors §1's step engine exactly), a "WHERE TO FIND" index, workflow gates
  (the same inductive trace the Stop hook nudges for, plus context-window budgets: ~450k tokens →
  prepare handoff, 500k → hard stop), and a closing "architect identity" directive written in the
  first person as a standing self-instruction for every session.
- **`DELEGATE.md`** (70 lines) — the subagent delegation protocol referenced from `CLAUDE.md`.
  Core rule: the *main* agent owns SDLC/git/integration; subagents implement + verify + report only
  (never commit/push/PR/switch branches). Subagents start with zero shared context and must be
  briefed fully. Parallel subagents must touch disjoint files. A `reviewer` pass is **mandatory**
  before any commit — explicitly called out as "no commit until reviewer-approved (SHIP-READY) and
  build/test-green," with an anti-pattern list including "a reviewer that rubber-stamps."
- **`README.md`** (roughly 260 lines) — the human-facing companion to `CLAUDE.md` ("why" vs "how").
  Five golden rules (sbx-wrappers-only, everything through `sbx sdlc`, YAML-first, never hand-edit
  generated files, vault is service-account-only) followed by copy-paste command blocks for the
  full lifecycle: discover → SDLC loop → build/run → test → add-a-secret (with an ASCII diagram of
  `project.yml`/`application.yml` → `sbx project bootstrap --target ...` → the generated
  `.env*`/compose/Dockerfiles) → migrations/backups → deploy → git. Two lines worth flagging as
  concrete operational scars: "Never co-commit `.sbx/ports/registry.yml` (it's machine-local)" and
  the repeated PR-number-read-back warning also present in `merge.md`.
- **`sdlc-artifacts.csv`** (58 rows, 9 columns: `#, FDD Step, Phase, Step Name, Capability Ref,
  Artifact(s), Pattern / Filename, Scope, Notes`) — a flat index cross-referencing every FDD
  sub-step (including sub-steps not all present as `.framework/lifecycle/sdlc/` directories, e.g.
  `FDD1.US.ACTOR`/`FDD1.US.NARRATIVE` appear as rows here but not as their own step folders,
  suggesting some steps were later merged/consolidated) against what artifact it should produce,
  the file-naming pattern, whether the artifact is durable or "Temporary" (several early rows are
  marked `Temporary` / "no file written," e.g. FDD1.G's goal is only captured inline in
  `execution-state.yml`'s context field, not its own file) — effectively a specification-vs-reality
  register for the step engine's output contracts.
- **`cliff.toml`** — [git-cliff](https://git-cliff.org/) config (referenced as backing Decision
  #0074) that generates a conventional-commits changelog grouped by `feat`/`fix`/`refactor`/`perf`/
  `doc`/`style`/`test`/`chore`/`ci`/`build`, skipping `chore(release)`/`chore(deps)`. Confirmed in
  use: `.sbx/articles/devlog/devlog-2026-01-13.md`'s content is exactly this template's output
  (grouped `## Features` / `## Bug Fixes` / `## Refactoring` / `## Maintenance` with short hashes) —
  the devlog folder is this tool's daily output, not hand-written journal entries.

---

## 7. `.claude/worktrees/` — pattern, not contents

`git worktree list` from the main checkout shows 5 live worktrees plus 2 **prunable ghosts**
(`contabo-inspect`, `sbx-booking-price` — directories deleted without `git worktree prune`, so git's
internal registry still references them) — plus a `cv/` directory on disk that is *not* a registered
git worktree at all (no `.git` file/dir inside it, just an empty leftover `clients/` stub) and a
`.claude/` subfolder holding a stray `settings.local.json` (not itself a worktree, likely an
accidental nested copy).

Naming shows two eras: newer worktrees are named after their SDLC task
(`7tree-continue` → branch `feature/7tree-2607-138-po-review-tier1`, `rbac` → branch
`feature/sbx-framework-2607-136`, `br-md` → branch `feature/cms-2607-123-rich-content-editor`);
older ones use an ad hoc `worktree-<name>` branch with no task-id correlation (`br-ai` → branch
`worktree-br-ai`, `br-docs-about` → branch `worktree-br-docs-about`). This matches the `worktree`
command/skill described in §4: worktrees are a parallel-dev-session mechanism (isolated checkout +
per-worktree port allocation via `.sbx/.worktree`), and the naming drift over time tracks the
broader shift toward task-id-anchored branch names once the FDD engine's task ids became the
organizing unit for everything else in the repo.

---

## 8. The `sbx` binary and `go.work`

A 38MB compiled Mach-O x86_64 executable sits at the repo root (`./sbx`, not run here per
instructions). A second copy of the same binary lives at `.sbx/.runtime/bin/sbx` (symlinked to
`~/.local/bin/sbx` per `README.md`'s "Rebuild the `sbx` binary itself" instructions) — the root-level
one may simply be an older/duplicate build artifact rather than the one actually invoked day to day.

`go.work` (Go 1.26) joins six modules into one workspace:
```
./clients/andrei/projects/shredbx/apps/shredbx-site/go
./clients/bestie/projects/bestays/apps/dashboard/go
./clients/bestie/projects/bestierealestate/apps/api-chi
./clients/egor/projects/myFamilyTree/apps/api/go
./projects/sbx/apps/sbx/go        <- the sbx CLI's cmd/ entrypoint
./projects/sbx/packages/core/go   <- the sbx CLI's shared core packages
```
So the CLI that mediates literally every SDLC action in this repo (`sbx sdlc`, `sbx task`, `sbx
knowledge`, `sbx decision`, `sbx deploy`, `sbx git`, `sbx vault`, `sbx build/test/check`…) is itself
built from `projects/sbx/apps/sbx/go` + `projects/sbx/packages/core/go`, joined via `go.work` into
the same workspace as four unrelated client-project Go API modules — presumably so the CLI's own
build/test/check tooling can operate uniformly across the CLI and the apps it governs without a
separate toolchain setup per module.

---

## Notable patterns worth studying further

- **Template-rendered, dispatch-annotated steps.** `step.md` (frontmatter: gate requires/produces,
  agent/role/personality/skill/command, required/available knowledge tags, output contract, plain-
  English gates) + a Markdown prompt body, rendered per-task into an archived XML prompt — a clean
  separation between "the reusable step definition" and "the live, task-specific, timestamped
  instance of having asked for it." Worth comparing directly against how process-os processes/
  actions render and record runs.
- **Adaptive step-skipping via a small live classifier.** `scope_answers` at init +
  `reclassifications[]` mid-task (each with a stated reason and timestamp) — one step catalogue
  serves both a one-file static-page task and a full persistence+integration feature without the
  small task manually opting out of irrelevant steps.
- **The `reconcile` skill's gap/drift framing** ("requirements are truth; code that disagrees is
  drift to flag, not adopt") plus `sbx decision scan --lint --strict` as an actual blocking merge
  gate — two different strengths of "governance checking" (LLM-driven audit vs. deterministic CLI
  gate) cleanly separated and both real, unlike the never-built capability-validation hook.
  Worth studying as a template for where process-os should put hard-mechanical checks vs.
  LLM-mediated ones.
- **The three-map (entities/attributes/references) vs. the four-layer (PD/MD/SI/UI) split are two
  independent axes**, not one model — worth keeping that distinction sharp if borrowing either.
- **Filesystem-scoped edit sandboxing tied to a live task pointer per git branch**
  (`sdlc-sandbox.sh` + `.sbx/.runtime/active/<branch>`) as a cheap, general mechanism for "no edits
  outside declared task scope," independent of any LLM discipline.
- **Decision records that are born-ratified with a mandatory documented wrong-pattern and a
  pre/post gate against re-deciding or leaving conflicts unmarked** — a stronger ADR discipline than
  most hand-rolled systems bother with.

## Things that look like dead ends / organic mess, not worth copying

- **`capability-validation.yml`** — a fully speculative hook spec (complete with a critical/warning
  capability taxonomy and a settings.json snippet ready to paste in) whose prerequisite CLI command
  was never implemented and is not wired into any hook today. Purely aspirational documentation
  masquerading as a spec.
- **Three PreToolUse guard hooks silently disabled together, undocumented.** `git-no-mutation-on-
  main.sh`, `pr-base-guard.sh`, and the bash-deny-revalidate hook were all moved to `off/` in one
  commit whose message doesn't mention them, and never revisited. The declarative permission system
  that remains is exactly what those hooks were built to backstop against bypass mode.
- **Two generations of the task-record system left both in the tree.** `.sbx/tasks/` (8 sparse
  ETVX-style records from March 2026) was abandoned in favor of `.sbx/.runtime/tasks/` (458 records,
  the FDD step-engine format) — the old one wasn't deleted, just stopped being written to.
- **Triplicated per-step output.** The same structured content (review scores, page model, build
  results…) is persisted inline in `task.yml`, again standalone in `artifacts/*.yml`, and again
  embedded inside the relevant `steps/*.xml` snapshot — functional but a lot of duplicated bytes
  across 458 tasks.
- **Inconsistent status enum values** (`completed` vs. a lone `complete`) and **stub task folders**
  (empty `task.yml` with only an `artifacts/` dir) scattered through `.runtime/tasks/`.
- **Stale worktree bookkeeping**: two "prunable" ghost worktrees in git's own registry, plus a
  non-worktree leftover directory (`cv/`) sitting inside `.claude/worktrees/` that isn't a worktree
  at all — nobody runs `git worktree prune` as part of the workflow.
- **Root-level clutter unrelated to the SDLC system itself** (dozens of loose `.png` screenshots,
  a zero-byte `backup-*.sql`, two copies of the `sbx` binary) — noted per the task brief as
  out-of-scope for this pass, but visible evidence that housekeeping is not itself a governed step
  in this system.
