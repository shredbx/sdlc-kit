# sbx.framework inventory — "sbx v2," the paused predecessor to process-os

**Last updated:** 2026-09-23
tags: sbx-framework, prior-art, research

> Scope note: read-only survey of `/Users/solo/Projects/workspaces/sbx.framework`, per the user's
> own framing — "sbx v2," the attempt between shredbx ("sbx v1") and the current sdlc-kit/process-os
> effort ("v3"). The user stopped this repo specifically to build process-os itself instead. This
> document describes what actually exists there: how far it got, what it was trying to become, and
> two things asked for by name — an AI-assistant implementation, and a "13 capabilities" definition.
> Nothing in sbx.framework was written, edited, or deleted to produce this report.

---

## 1. Overview — what sbx.framework was trying to be

sbx.framework's own `README.md` states its purpose in one line: *"SBX — an AI assistant
configuration system: schemas rule, tools produce, records result. Runs locally; bootstraps other
repositories."* Its `CLAUDE.md` opens the same way, more pointedly: *"An AI assistant configuration
system. The consumer is an AI agent, not a human at a prompt."* This is a specific, deliberate
framing — the repo is not "a tool a human uses," it's a set of schemas (rules), tools/actions
(callables), and records (results) meant to be operated primarily by an AI coding agent (Claude),
with a human (Andrei) validating gates and rulings along the way. That framing is the direct
conceptual ancestor of how process-os presents itself for MCP/agent consumption — same idea, built
a season earlier, inside a much bigger and more opinionated repo.

The repo describes its own scope with a four-stage pipeline (verbatim from `README.md`):

```
  packages   → similar-interface building blocks, easy to integrate
  framework  → glues packages together — configuration, templates, runtime mechanisms
  project    → picks platforms + apps; a GENERIC whitelabelable product, no client baked in
  delivery   → OUT OF SCOPE — a separate repository per client, deployed however that client needs
```

The stated long-term goal: "a personal, multi-year hub for every reusable piece of software
development Andrei has ever built or will build... hundreds of packages are expected, across many
platforms and languages." Client-specific work is explicitly and permanently out of scope for this
repo — it belongs in a separate repo per client (planned as a git submodule pulling in what this
repo produces).

Architecturally, sbx.framework is a **self-hosting, schema/process-driven engine**, built and
governed by its own rules from day one. Its organizing primitives (`framework/core/{schema, record
[as `type`+`file`], template, action, process}`) map almost one-to-one onto process-os's own
`type · schema · action · template · process` split described in sdlc-kit's `CLAUDE.md` — this is
the single strongest piece of evidence that process-os grew directly out of this repo's engine
design, not from a clean slate. The engine is invoked as `uv run sbx <verb>` (Python, `uv`-managed);
core verbs are `put · get · list · find · rev` plus `apply` and `walk` (process traversal), plus
governance verbs `check` (what's missing) and `gate` (pass/fail against a position's required
protocols).

---

## 2. Directory structure

```
sbx.framework/
├── README.md, CLAUDE.md, sbx.yml (12.6 KB — the store/schema config), repo.yml, pyproject.toml
├── framework/core/{action, file, process, schema, template, type}   — the engine's own vocabulary
├── packages/                                                        — the reusable plugins
│   ├── core/python            (22 files)  — the seed engine: store, render, walk, check
│   ├── sdlc/python             (600 files) — the SDLC plugin: process/schema/action rows, by far
│   │                                          the largest package; also carries `records/` for
│   │                                          non-runtime plugin data such as the `ai-agent` actor
│   ├── web-ui/svelte           (393 files) — a Svelte component library (brand/theme/component rows)
│   ├── workbench/svelte         (30 files) — "Solo IDE" admin shell (renamed from `ide` at Milestone V)
│   └── content/svelte           (14 files) — an entry/category/FAQ content plugin
├── apps/
│   ├── cli/python                          — the `sbx` CLI app shell (pyproject + app.yml + tests)
│   └── web/svelte              (127 files) — the SvelteKit "Solo IDE" web app (schema-driven admin UI)
├── projects/{bos, ide}                     — two whitelabelable product instances (config + records)
├── prototypes/{process-canvas, solo-ide}   — earlier design spikes
├── sandbox/                                — untracked, gitignored-by-effect; three recent (2026-09-19)
│                                              spikes, see §3
├── docs/
│   ├── research/    — ~20 dated research passes (incl. a direct read of shredbx's UI protocols)
│   ├── plans/       — dated milestone plans (A through V so far), a `processos-bodies/` folder of
│   │                  ready-to-put row bodies for the never-executed Milestone T
│   ├── governance/, checklists/, reviews/, validation/, wiki/, reports/
│   ├── ACTIVE-MILESTONE.md   — archived 2026-09-15; superseded by a live "board" of rows
│   └── MILESTONE-SELF-DESCRIBING-SYSTEM.md
├── scripts/, .githooks/, .github/workflows/
└── .claude/{agents, commands, hooks}        — session-start hook auto-launches the Solo IDE
```

Vendored/build directories present but not enumerated: `.venv/` (per-project, several), `out/`
(git-ignored build/instance output), `.playwright-mcp/`, `.pytest_cache/`, `node_modules/` nested
under `.dirty-repo-satte/` (an untracked, separately-named directory that appears to hold an older
or parallel copy of blueprints/docs/products and its own `node_modules` — not investigated further,
out of scope for this pass and clearly not part of the maintained tree).

**Scale, by rough file count:** `packages/sdlc/python` (600 files) and `packages/web-ui/svelte`
(393 files) dominate — this is a large, working build, not a stub. At the last measured point in
the repo's own plans (2026-09-18, Milestone V), the test suites stood at **sdlc 615 · core 219 ·
cli 23 · web 101 — 958 tests, all green**, `sbx check` reporting 0 problems across 1,008 rows, 129
templates, 132 artifacts.

**Timeline:** 856 commits from `93c6188` "Initial commit" (2026-07-27) to `f0574e3` (2026-09-18,
14:40 +07) — about 7.5 weeks of near-daily, disciplined commits (`feat(V-1): …`, `refactor(V-1): …`,
`docs(V-1): …` — every commit tagged to its milestone letter). No commits since 2026-09-18; sdlc-kit
(v3) has commits from 2026-09-21 onward, a 3-day gap.

---

## 3. The AI-assistant finding

The user's framing was: *"We have implementation of such agent in go or python in shredbx and in
sbx.framework some part as well."* Three distinct things exist here, at three different levels:

### 3a. The whole repo as "AI-assistant configuration" (meta level)

As covered in §1, the entire framework bills itself as infrastructure *for* an AI agent to operate
— schemas as rules the agent reads, actions as tools it calls, records as the results it leaves
behind. This isn't a chatbot; it's the same "agent-first tooling" philosophy process-os now embodies.
One concrete artifact of this: `packages/sdlc/python/records/sdlc/actor/ai-agent.yml` models the AI
coding agent itself as a first-class SDLC actor —

```yaml
kind: actor
id: ai-agent
conforms_to: framework/sdlc/schema/actor
purpose: Follows walk answers and runs offered actions without guessing — the primary
  consumer of the framework's prompts.
group: system
capabilities:
- walk-processes
- run-actions
```

### 3b. A planned AI-assistant *product* — Milestone T, "ProcessOS v0.1" (the real find)

This is the important one. `docs/plans/2026-09-15-milestone-t-processos-v0-1.md` (1,083 lines) is
an extraordinarily detailed implementation plan for exactly the kind of AI-assistant product the
user described — and its own internal name is **"ProcessOS v0.1."** Supporting documents:
`docs/research/2026-09-15-processos-technical-task.md` (the requirements) and
`docs/research/2026-09-15-ai-assistant-mapping-pack.md` (a decision/options pack for the "T-0 mapping
session," each open question with options, tradeoffs, prior-art citations from shredbx, and a
recommendation).

The plan specs, in real detail (including runnable Python inlined directly in the plan document):

- a `processos` Python plugin package owning 8 new kinds — `org · project · provider · connector ·
  agent · chat · conversation · cache`
- a `conversing` process with actions `chat-turn` (a tool-calling loop over an OpenAI-compatible
  `/chat/completions` API, streaming, per-session conversation persistence, `max_tool_rounds`
  bounding) and `publish-chat` (issues a bearer token, stores its hash)
- a generic `http-request` action — explicitly modeled on "the n8n HTTP Request node pattern as
  rows: intelligence in the row, plumbing here" — the one script any connector-calling action borrows
- a FastAPI app (`apps/processos/python`, `surface: api`) exposing the store's admin seams plus the
  chat/completions route, containerized (Dockerfile + compose) for a Dokploy deploy chain
- a Solo IDE (SvelteKit) chat panel and a `connection` kind so the local dev IDE can drive a remote,
  deployed instance over HTTP

This is a real, thought-through design for an AI-assistant backend — provider-agnostic chat,
tool-calling bound to whatever actions a process position offers, multi-tenant via org/project,
deployable. But it was **never built**. The plan's own header states this outright: *"Nothing of
the ai-assistant project is built here (D-S-15) — this is the pack the mapping session opens
with."* Confirming this, the later Milestone V plan (2026-09-17/18, the repo's last work) records:
*"the unbuilt milestone-T bodies follow decisions 11 and 13 — no retired coordinate, package or app
left in a draft someone would build from."* A direct search confirms it: no `packages/processos` or
`apps/processos` directory exists anywhere in the tree; `git ls-files | grep -i processos` returns
only the plan document and its `docs/plans/processos-bodies/*.yml` — ready-to-put row bodies that
were prepared but never executed.

**This is the strongest single finding of this survey.** sbx.framework's own roadmap literally named
its next AI-assistant milestone "ProcessOS v0.1" — i.e., process-os was first conceived *as a
milestone inside sbx.framework*, before being pulled out to become its own standalone repository
(the real process-os, `github.com/shredbx/process-os`). That extraction is almost certainly the
concrete moment behind the user's "I stopped to build processos and start all over" account — not a
vague pivot, but a specific, plan-level decision visible in this repo's own history: the engine got
generalized and shipped as its own product before the AI-assistant milestone that would have proven
it out ever got built.

### 3c. Three small, already-abandoned prototype spikes (untracked)

`sandbox/` holds three folders, each named with a `[26-09-19]` date prefix — **one day after** the
repo's last tracked commit (2026-09-18). None are committed to git (their contents fall entirely
under `.gitignore` rules for `.venv/`, `.pytest_cache/`, `out/`, `__pycache__/`, so nothing about
them shows in `git status`). In all three, **the actual `.py` source files have since been deleted**
— only compiled `__pycache__/*.pyc` bytecode (and, for one, a generated JSON artifact) survive. This
was verified by extracting identifier/string tables from the bytecode:

- **`echo-chat-assistant`** — a minimal FastAPI app. Strings recovered: `FastAPI`, `ChatRequest`,
  `ChatReply`, a POST endpoint, title `"Echo chat assistant"`. `.venv` has `fastapi`, `httpx`,
  `uvicorn`, `pytest` installed. This looks like a bare pipeline-validation stub (confirm FastAPI +
  uv + pytest work together), not a working LLM-backed assistant — no LLM/provider library is
  installed in its venv.
- **`n8n-chat-agent-generator`** — a Python code generator (`generate.py`) for n8n workflow JSON,
  using Pydantic models (`Workflow`, `Block`, `Chat`, `Agent`, `Credential`, `Config`, deterministic
  `uuid5` node ids). One surviving output, `out/workflow.json`, is a generated n8n "Support Bot"
  workflow wiring n8n's own LangChain nodes (`chatTrigger` → `agent` → `lmChatOpenAi`, model
  `gpt-4o-mini`). This is an experiment in generating AI-agent workflows for an external orchestration
  platform (n8n) rather than building the chat agent directly in Python.
- **`process-kit-schema`** — the most substantial of the three, and *not* chat-related: a JSON-Schema
  validation engine and CLI (`cli.py` with argparse subcommands, `engine.py` — recovered docstring:
  *"The engine: checks documents against schemas and protocols"* — plus `config.py`, `errors.py`,
  `ids.py`, `loader.py`, `model.py`, `pointer.py`, `profile.py`, `resolvers.py`). By name and shape
  this looks like a direct, minimal precursor experiment for process-cli's own schema-checking
  engine, done as a clean-room spike outside sbx.framework's much larger `packages/core/python`.

All three read as quick, disposable spikes taken right at the pivot point — already partly cleaned
out (source deleted, bytecode orphaned) by the time of this survey.

**Language/stack summary:** everything AI-assistant-related found in sbx.framework is **Python**
(FastAPI, httpx, uvicorn, pytest, uv) — no Go AI-assistant code exists in this repo. (Per the user,
Go/Python assistant work also exists in shredbx; that is out of scope for this pass by design.)

---

## 4. The "13 capabilities" finding

**Found — verbatim, with exact source — but as a citation of shredbx (v1) inside sbx.framework's
own research notes, not as an active definition sbx.framework itself adopted. sbx.framework's own
design explicitly parked and superseded the capability-as-organizing-principle.** Read the nuance
below before reusing this list for v3.

The definitive quote, `docs/research/2026-08-06-old-sbx-ui-protocols.md:346` (a table row from a
direct read of shredbx's UI/protocol schemas):

> `requirement` | `core/requirement.yml` | `properties, definitions, cascade, examples, rules,
> applicability_rules` | **capability enum (line 255), 13 values**: `modeling, architecture,
> implementation, infrastructure, configuration, testing, security, documentation, automation,
> deployment, observability, operations, continuity`; `constraint.type [guideline, pattern,
> standard, technology, regulatory, capability]`; `enforcement [must, should, may]`

That is exactly 13 values: **modeling, architecture, implementation, infrastructure, configuration,
testing, security, documentation, automation, deployment, observability, operations, continuity.**

This is corroborated in two more places inside sbx.framework:

- `CLAUDE.md:483` — *"**The corpus capability enum is 13** (`requirement.yml:255`), not 5 — the 5
  files under `.sbx/workspace/requirements/capabilities/` are requirement *records*, not the
  vocabulary."*
- `docs/research/2026-08-02-wave2-lane-design.md:217-218` — *"capabilities / 13 descriptors —
  deprecated by the corpus (#0057: '1000 capability files … unnecessary brainstorming matrix')."*

**The nuance that matters for v3:** every one of these citations is sbx.framework's own research
*reading shredbx's* `core/requirement.yml` (line 255) — it is v1's vocabulary, documented as prior
art, never redefined as a live schema/enum inside sbx.framework's own `framework/` or `packages/`
trees (confirmed — no 13-item capability enum exists anywhere in the live, buildable part of this
repo; the only other `capabilities` fields found are unrelated free-text lists on `actor` rows,
describing what an actor/role can functionally do, explicitly *not* a namespace). More than that,
sbx.framework's own architecture **ruled the concept out** on 2026-08-01. From `sbx.yml:106-108`:

> "NAMESPACES — who owns a kind's rows. The old capability namespaces are removed (process ruling
> 2026-08-01: what a capability was becomes a subprocess); new namespaces arrive with the plugin
> that owns them (D14), never upfront."

And `CLAUDE.md`'s own vocabulary table: *"`capability` | **parked — out of scope until modeled**
(wiki §parked). What a capability was becomes a subprocess (process ruling 2026-08-01)."* The v1
plan — *"capabilities as namespaces, conformance cross-checks, NFR slices"* — is explicitly called
**SUPERSEDED**, replaced by a five-primitive model (`schema · record · template · action · process`)
where a process's own decision tree (subprocess → switch → case lanes) does the organizing work a
capability namespace used to do.

**Practical implication for sdlc-kit:** the exact 13-value list the user remembers is real and
verbatim, but reviving "13 capability scopes" for v3 would be a deliberate *reversal* of a
considered ruling the v2 attempt made and stuck to for its entire 7.5-week build — not a fresh
idea. Worth surfacing to Andrei directly before modeling it: either the 2026-08-01 reasoning (flat
capability namespaces don't compose, corpus decision #0057's "unnecessary brainstorming matrix") no
longer applies at v3's different scale/shape, or the "13 capabilities" the user wants for v3 are
meant as a different kind of thing (e.g., scope *names*, not row-owning namespaces) than what v2
rejected — that distinction is worth pinning down before it's modeled.

---

## 5. Maturity / completeness assessment

This was not a toy or an early spike — it is a **rigorously engineered, substantially complete
platform-engineering effort** that ran for 7.5 weeks under real TDD discipline (red-first tests,
one commit per task, evidence pasted rather than claimed, an `agent-guard` hook that eventually
hard-blocked any raw shell command outside `uv run sbx …`). At its last measured point (2026-09-18):
1,008 rows, 129 templates, 132 artifacts, 958 passing tests across four suites, `sbx check` and
`reindex --verify` both clean.

**What got built and works:**
- A self-hosting Python engine (store, render, walk, check, gate) — the direct design ancestor of
  process-cli.
- A large SDLC plugin (`packages/sdlc/python`, 600 files) with dozens of processes/schemas/actions
  covering modeling, testing, implementation, delivery (PR-based merge, changelog derivation,
  battery-gated CI).
- A working SvelteKit "Solo IDE" admin web app (`apps/web/svelte`, 127 files) that renders every row
  live, plus a substantial reusable UI component library (`packages/web-ui/svelte`, 393 files) with
  its own contract-testing/fixture-derivation chain.
- Real governance rigor: PR-based delivery, a merge tool that refuses on failed gates, a session-start
  hook that auto-launches the dev IDE, provisional-ruling tracking with named overturn conditions.

**What got built and then retired:** a Go API app, an `rbac` package, and "the Go lane" generally
were built and then explicitly removed at Milestone V (2026-09-17/18) — so sbx.framework did, at
one point, have a Go component, but it was cut before the repo paused, not left half-built. A
`bos-saas` project and an auth/sign-in flow were also built then retired the same milestone.

**What was designed in full but never executed:** the actual reason the framework says it exists —
an AI-assistant/chat product — was planned to the level of runnable inline code (Milestone T,
§3b above) and then abandoned in favor of extracting the underlying engine into a new, standalone
repo (process-os). Three quick, source-already-deleted spikes the day after the last commit
(§3c) suggest that pivot happened right around 2026-09-18/19.

**Overall read:** sbx.framework stopped not because it stalled or broke, but at a natural
inflection point — right when it was about to build the product (the AI assistant) that would have
proven its engine out, the team apparently judged the engine itself was the more valuable,
more general thing to ship, and split it out. What's left here is a large, working, but now-paused
monorepo whose most valuable asset — the schema/process engine design — already has a life of its
own in process-os.

---

## 6. How this compares — to sbx v1 (shredbx) and to process-os

**vs. sbx v1 (shredbx):** v1 is a monolithic, subject-aware Go CLI (~140 Go packages) with a
hand-built FDD-based SDLC layer (`.sbx/`: Claude Code skills, role agents, hooks) bolted on top —
one codebase, one language, governance and product logic intermixed. sbx.framework (v2) is
architecturally the opposite: a small, generic, subject-agnostic engine (five primitives) meant to
be reused across "hundreds of packages," with v1 explicitly used as the *source corpus* to port and
refactor from rather than reference architecture ("search shredbx before writing any definition" is
a standing rule throughout sbx.framework's `CLAUDE.md`). Where v1 baked its 13-capability taxonomy
directly into its `requirement.yml` vocabulary, v2 read that taxonomy, considered it, and explicitly
declined to carry it forward as a namespace — see §4.

**vs. process-os (v3's engine):** these are close cousins, not strangers. Same five-primitive
vocabulary and near-identical philosophy (process = decision tree; schema = rule; action = dumb
callable; template = generator; engine owns plumbing, never behavior), the same self-hosting/
dogfooding discipline, the same "search before writing" ethos. The difference is packaging and
scope: sbx.framework bundled its engine *inside* one large, opinionated monorepo — a fixed
packages→framework→project→delivery layering, a bespoke `uv run sbx …` CLI, and a large amount of
product-specific weight accumulated along the way (a full admin IDE, a UI component library, a
content plugin, a since-retired Go/auth stack). process-os reads as that engine *extracted and
generalized* — stripped down to just the `framework/core` primitives and shipped as an installable
tool (`process-cli`) that any workspace, including this new sdlc-kit repo, can adopt from scratch.
sdlc-kit is then positioned to rebuild the product layer — packages, projects, and (per the open
question in §4) possibly a capability-scoped organization and an AI-assistant milestone of its own —
on top of that extracted engine, informed by what did and didn't get finished here, rather than
inside one all-in-one repo again.
