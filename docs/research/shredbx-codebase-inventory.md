# shredbx codebase inventory

**Last updated:** 2026-09-23
tags: shredbx, codebase-inventory, research

Read-only structural inventory of `/Users/solo/Projects/workspaces/shredbx` — a real, ~1-year-old,
messy monorepo covering the user's personal "SBX" framework, several client projects, and a large
docs/ archive. Purpose: give the sdlc-kit / process-os planning conversation a map of what exists,
organized by platform, so concrete porting candidates can be picked later. This is a map, not a
code review — implementation logic was not inspected beyond file/folder names, package
manifests, and a handful of README/CLAUDE.md/NOTES.md files.

**Headline finding:** shredbx already contains a full, home-grown "SDLC factory" tool — a Go CLI
called `sbx` (built from `projects/sbx/apps/sbx/go`, module `github.com/shredbx/sbx-api`, backed by
a ~140-package `projects/sbx/packages/core/go` library). It does SDLC task walking, knowledge/decision
governance, vault secrets, deployment (Dokploy), external-mirror sync, entity/schema modeling, CMS,
and more — i.e. conceptually the same territory the new `sdlc-kit`/`process-cli` workspace is
building, but implemented a year earlier, monolithically, and (by the root `CLAUDE.md`'s own
"STRIKING SYSTEM" and drift tripwires) under real strain. `docs/plans/2026-01-24-sdlc-factory-vision.md`
is the original design doc for this — worth reading directly rather than reinventing, as prior art
for what worked and what didn't. Per the sdlc-kit CLAUDE.md's own instruction, this is prior art to
learn from, not to copy wholesale.

---

## 1. By platform / language

### Go

- **`projects/sbx/packages/core/go`** — the SBX "Fabric" framework itself. ~140 `internal/` and
  `pkg/` packages: SDLC (`pkg/sdlc`, `internal/sdlc`), knowledge/decision governance
  (`internal/knowledge`, `internal/decision`), vault/secrets (`pkg/vault`), deployment
  (`internal/deploy`, `pkg/deployment`, `pkg/dokploy`), entity/schema modeling
  (`internal/entityschema`, `internal/schema`, `pkg/entityschema`), CMS (`pkg/cms`), external-mirror
  sync (`internal/external`), plus many **generic domain packages** reused across client projects:
  `pkg/property`, `pkg/auth`, `pkg/rbac`, `pkg/person`, `pkg/address`, `pkg/money`, `pkg/i18n`,
  `pkg/media`/`pkg/image`/`pkg/video`, `pkg/faq`, `pkg/inquiry`, `pkg/calendar`, `pkg/lease`,
  `pkg/csvimport`, `pkg/seo`, `pkg/scheduler`, `pkg/feed`/`pkg/rss`, `pkg/storage`, `pkg/r2`,
  `pkg/authz`, `pkg/completeness`, `pkg/socialnetwork`, and more. **This is the single largest,
  most mature reusable Go asset in the repo.**
- **`projects/sbx/apps/sbx/go`** — the `sbx` CLI/API binary itself (`sbx-api` module), source of the
  compiled `sbx` binary sitting at repo root (confirmed: same build, `github.com/shredbx/sbx-api`).
- **Client Go APIs**, all built as thin Chi-router consumers of `packages/core/go` (per
  bestierealestate's CLAUDE.md: "thin Chi consumer (99% rule) — domain logic lives in shared
  packages"):
  - `clients/andrei/projects/shredbx/apps/shredbx-site/go` — the personal shredbx.com site API.
  - `clients/bestie/projects/bestays/apps/dashboard/go`
  - `clients/bestie/projects/bestierealestate/apps/api-chi` — largest client Go app (property
    search/similar/related endpoints, smart collections, response localization — has real Go test
    files alongside the handlers).
  - `clients/egor/projects/myFamilyTree/apps/api/go` — Chi + PostgreSQL API for the genealogy app.
  - Small worker apps: `bestierealestate/apps/backup` (Go backup workflows), `shredbx/apps/backup`.
- All of the above are wired into the **root `go.work`** (see §3), confirming they share one Go
  module graph rooted at `packages/core/go`.

### Python

- **`projects/assistant-kit`** — an AI-assistant "studio/builder" product. `apps/builder/python`
  (backend, uv/pyproject-managed) + `packages/chat/python` and `packages/prompt/python` (shared
  Python libraries — chat/prompt abstractions used by the builder). `NOTES.md` shows active,
  detailed engineering (streaming LLM responses, template/section "LINKED block" resolution,
  snapshot/fork semantics for guest vs. admin) — this is a real, evolving LLM application, not a
  stub. Has its own `docker-compose.yml` and Makefile.
- **`projects/sbx-next`** — a `pyproject.toml`/`uv.lock`-based Python project with `CLAUDE.md`,
  `LAYOUT.md`, `IMPORT-BATCH-1.md`, `sbx.yml`, plus `src/` and `tests/`. Reads as an in-progress
  **next-generation rewrite/successor of the `sbx` tool in Python** (name literally "sbx-next"),
  still early — no deep package tree yet, mostly planning/layout docs. Worth reading `IMPORT-BATCH-1.md`
  and `LAYOUT.md` directly if a Python-based successor to `sbx`/`process-cli` is ever on the table.
- **`clients/bestie/projects/bestierealestate`** and its mirror in `externals/` also carry a
  `pyproject.toml`/`uv.lock` at the project root (alongside the Go API) — likely tooling/automation
  scripts, not a separate app; not deep-inspected.

### TypeScript / Svelte / Next.js

- **SvelteKit 2 / Svelte 5** is the dominant web stack across nearly every client, all built on a
  shared **`projects/sbx/packages/core/svelte`** component/design-system library plus a large family
  of small, focused UI packages under `projects/sbx/packages/`: `ui-image`, `ui-video`, `ui-code`,
  `ui-diagram`, `ui-map`, `ui-calendar`, `units`, `text-template`, `canvas-kit`, `canvas-ui`,
  `ui-source-picker`, `ui-contact`, `ui-seo`, and a TS `core/ts/animations` package. **These are the
  clearest candidates for a genuinely reusable Svelte component/package base** — each is a small,
  independently versioned pnpm workspace package (`package.json` + `package.yml` governance file).
  - App consumers of this base: `projects/sbx/apps/sbx/svelte` (the sbx web app itself),
    `projects/assistant-kit/apps/studio/svelte` (chat/prompt studio UI),
    `clients/andrei/projects/land-canvas/apps/web/svelte`,
    `clients/andrei/projects/shredbx/apps/shredbx-site/svelte` (personal site, + its own
    `packages/shredbx-ui` design-system extension),
    `clients/bestie/projects/bestays/apps/dashboard/svelte`,
    `clients/bestie/projects/bestierealestate/apps/web-svelte` (largest client web app — has its own
    `externals/`, extensive Playwright `e2e/` suite, many dated e2e run logs),
    `clients/egor/projects/myFamilyTree/apps/web/svelte`,
    `clients/wanflo/projects/wanflo/apps/web/svelte` (bilingual EN/RU marketing site, fully
    prerendered, deployed standalone on Vercel).
  - **Games package family**: `projects/sbx/packages/games/{vendor-pixi, engine-renderer,
    engine-input, game-preset-arcade, game-svelte}` is a reusable Pixi.js-based 2D game
    engine/renderer/input layer + Svelte wrapper + an "arcade preset". Concrete games built on top
    live under `clients/andrei/projects/shredbx/packages/games/{game-snake, game-pacman,
    game-tetris, game-2048, game-minesweeper}` — a clean engine/product split, i.e. an actual
    reusable framework with multiple products built on it, not a one-off.
- **Legacy Next.js / Turborepo stack**: `clients/bestie/projects/bestie-workspace` — an older
  "ReactBook" Next.js + Turborepo monorepo (turbo.json, jest configs, `src/apps/{bestays-web,
  bestierealestate-web}`, `src/packages/{web-ui, web-services, property2, playground}`,
  `src/configs/*`). Strong legacy signals: `CLAUDE-old-do-not-read.md`, `.old.claude/`,
  `.pb-old-and-outdated/`, all dated Jan 18 2026 vs. July 2026 for the current
  Go+Svelte `bestierealestate`/`bestays` projects it was superseded by. **Treat as superseded —
  interesting only if the Next.js component patterns themselves are wanted**, not as the live
  Bestie product.
- **Cloudflare Worker**: `clients/bestie/projects/bestierealestate/apps/watermark-worker` — small
  TypeScript Cloudflare Worker (wrangler.toml) for image watermarking; standalone, not part of the
  pnpm workspace.
- **`clients/yura/projects/hhh-reactnative`** — a React Native / Expo app (`app/`, `hhh-expo/ios`),
  notably its **own nested git repo** (`.git` present) rather than living inside the shredbx
  workspace conventions — likely an earlier or parallel track to the Kotlin native app below. Has
  its own docs, a `nativewind-fix-plan.md`, and task notes on Android build/SQLite migration
  fixes — reads as active but rougher than the Kotlin app.
- Root **`pnpm-workspace.yaml`** lists ~23 package globs (see §3) confirming the above as one pnpm
  workspace.

### Swift / SwiftUI (rare, high-value for platform targets)

- **`clients/egor/projects/myFamilyTree/apps/ios/swift`** — a genuinely mature, production-oriented
  SwiftUI iOS app ("SevenTree"), confirmed and more substantial than the prompt's baseline note:
  - Xcode project (`SevenTree.xcodeproj`) generated via `project.yml` (XcodeGen), not just a raw
    `.xcodeproj`.
  - CloudKit + Core Data (per root README: "SwiftUI, CloudKit, Core Data").
  - **Fastlane** setup (`fastlane/`) and a **Deploy** folder with `ExportOptions.plist` — real
    App Store deployment tooling, not a prototype.
  - Source organized conventionally: `Sources/{App, Managers, Models, Screens, Services,
    Generated}`.
  - Its own docs: `PRODUCT_DOCUMENTATION.md`, `CLOUDKIT_WORKFLOW.md`, `NOTES.md`, `docs/`.
  - `ios-screen-feature-status.csv` and `governance-table.md` at the project root suggest tracked,
    disciplined feature/QA status — an unusually well-governed client project.
  - **This is the strongest, and only substantial, Swift/SwiftUI asset in the whole repo** — directly
    relevant to the Swift platform target. Even if myFamilyTree-as-a-product isn't ported, its
    CloudKit sync patterns, XcodeGen setup, and fastlane/Deploy scaffolding are strong candidates for
    an iOS app-starter/bootstrap template.
- **`projects/voice-to-text/apps/menubar-macos`** — a **Swift Package Manager** macOS menu-bar app
  (`Package.swift`, SwiftLint config, `Tests/`, `Sources/`) for voice-to-text dictation. Smaller than
  myFamilyTree's iOS app but a second, independent Swift codebase with its own `docs/research`,
  `docs/plans`, and `docs/decisions` (e.g. "state-as-object and DI container" decision doc) —
  reads as a deliberately-designed small app, not a throwaway. Paired with 20 (!) standalone HTML
  prototype files (`prototypes/whisperbar-v01..v20-*.html`) exploring UI concepts before the Swift
  build — an interesting rapid-prototyping pattern (HTML mockups → SwiftUI) worth noting for process
  design even though the HTML files themselves aren't code to port.
- No other Swift/iOS work found anywhere else in `clients/` or `projects/`.

### Kotlin / Android (rare, high-value for platform targets)

- **`clients/yura/projects/hhh-android`** — a real, mature native Kotlin Android app ("HHH"):
  - Root `README.md` literally states "Kotlin Android application"; has `build.gradle.kts`,
    `settings.gradle.kts`, `gradle/`, `.gradle/`, `.kotlin/`, a release keystore
    (`app/hhh-release-key.jks`), `proguard-rules.pro`, `detekt-baseline.xml`/`lint-baseline.xml`
    (static analysis baselines — a sign of an enforced code-quality gate), and Firebase config
    (`app/google-services.json`).
  - Structured as `app/` (main Android application module, `src/main` + `src/test`) plus
    `apps/mobile/kotlin` and a separate **reusable library module**:
    `packages/configuration-module` — a standalone Gradle module (own `settings.gradle.kts`,
    `gradlew`) implementing a "Provider-Cache-Schema" typed configuration system (per its
    `package.yml`: "reusable, provider-agnostic typed configuration module that manages the full
    config lifecycle: define (schema) → cache (DataStore) → fetch (remote) → validate → observe
    (Flow-reactive state)"). Explicitly governed (`conforms_to: [entity]`, tied to "Decision #0144"),
    i.e. modeled the same way the Go/Svelte packages are — **a genuinely reusable Kotlin library**,
    the only one of its kind found in the repo.
  - `ROADMAP.md` shows an active gameplay/quiz-style app under iterative design (session screens,
    per-level answer modes) referencing its own prototype-alignment plan doc.
  - **This, alongside `hhh-reactnative` (see TS section) as a possible earlier/parallel track for the
    same product, is the strongest Kotlin/Android asset in the repo** — directly relevant to the
    Kotlin platform target, both as a potential app-starter and for the `configuration-module`
    pattern as a portable library idea (typed-config lifecycle) that could generalize beyond Kotlin.

### Other / Infra

- Deployment target: **Dokploy** (self-hosted PaaS), driven entirely through the `sbx` CLI
  ("MIRROR-ONLY" deploy model — see §4/§6, `infrastructure/`).
- Secrets: 1Password (`op`/`op-sa` scripts under `scripts/infra/`), read via `sbx vault`.
- Server bootstrap: a numbered hardening/setup script sequence (SSH hardening, UFW, Docker install,
  fail2ban, unattended-upgrades, sysctl hardening, egress firewall, Lynis audit, restic backups,
  Dokploy install) — a genuinely reusable "provision a fresh Ubuntu box for Dokploy + Postgres +
  Redis" runbook (see §5).
- Backups: `scripts/infra/db-backup.sh` / `db-restore.sh` / `db-provision*.sh`, plus per-project Go
  `apps/backup` workers — a repeated backup pattern across at least 3 projects (shredbx,
  bestierealestate, and a shared workflow file), i.e. reusable.

---

## 2. `projects/` — internal/shared products

| Project | Stack | What it is | Maturity |
|---|---|---|---|
| `assistant-kit` | Python (builder API, chat/prompt libs) + SvelteKit (studio UI) | An AI-assistant / prompt-template "studio & builder" — compose documents with LINKED/reference blocks, run them through LLM providers (incl. streaming), admin vs. guest resolution model. Has its own `docker-compose.yml`, active `NOTES.md` engineering log. | Real, actively developed app; substantial (own apps + 2 shared Python packages). |
| `sbx` | Go (CLI/API + core framework) + SvelteKit (web UI) | **The framework** — `sbx` CLI/binary (source of root `sbx` binary, confirmed), `packages/core/go` (~140 packages), `packages/core/svelte`, plus ~15 focused UI packages and the games engine family. | Very mature — this is the backbone the whole repo is built on. |
| `sbx-next` | Python (uv/pyproject) | Early-stage, appears to be a **Python-based next-generation successor** to the Go `sbx` tool (`CLAUDE.md`, `LAYOUT.md`, `IMPORT-BATCH-1.md`, `sbx.yml`). | Early/planning stage — few files, mostly docs/layout, `src/` + `tests/` present but thin. |
| `voice-to-text` | Swift (SwiftPM macOS menu-bar app) + HTML prototypes | Voice dictation menu-bar utility for macOS, with a deliberate HTML-prototype-first design process (20 mockups) and its own decisions/plans/research docs. | Small but deliberately engineered; real Swift app, not a stub. |

## 3. `clients/` — per-client inventory

| Client | Project(s) | Stack | Product (best guess) |
|---|---|---|---|
| **andrei** | `land-canvas` | SvelteKit web app (`apps/web/svelte`), plus `source-assets` (logo/cover images) | Small/early — only a logo, cover image, and a web app skeleton found; low maturity. |
| **andrei** | `shredbx` | Go API + SvelteKit (`apps/shredbx-site`), `apps/backup` worker, `packages/shredbx-ui` (site-specific design-system extension), `packages/games/*` (snake/pacman/tetris/2048/minesweeper, built on the sbx games engine) | Andrei's own personal site/portfolio (shredbx.com) — real, deployed product (has migrations, seed-data, sql, docker-compose for infra, analytics.yml). |
| **bestie** | `bestays` | Go (`apps/dashboard/go`) + SvelteKit (`apps/dashboard/svelte`) | A "dashboard" product for Bestie — real but smaller than bestierealestate; has its own `go.work`, migrations, `sync/`. |
| **bestie** | `bestierealestate` | Go Chi API (`api-chi`, with real `_test.go` files), SvelteKit web (`web-svelte`, extensive Playwright e2e suite), Python `assistant` app, Go `backup` worker, TS Cloudflare `watermark-worker` | **Largest, most mature client product in the repo** — a real-estate platform (property search/similar/related, smart collections, i18n response localization, CMS, FAQ, media/watermarking). Has `CLAUDE.md`, `ROADMAP.md`, `DEPLOY.md`, `HANDOFF.md` — clearly the most actively engineered client project. |
| **bestie** | `bestie-workspace` | Next.js + Turborepo (legacy) | **Superseded/legacy** — "ReactBook" monorepo predating the Go+Svelte rebuild; explicit `-old`/`-outdated` markers throughout, dated Jan 2026 vs. July 2026 for its replacements. Low reuse value except as a reference for what NOT to repeat, or if specific React component patterns are wanted. |
| **egor** | `myFamilyTree` | Go API + SvelteKit web + **SwiftUI iOS app** | Privacy-first genealogy app with visual family-tree builder and iCloud/CloudKit sync (per its README, all three apps are real, deployed products). **Confirmed**: the iOS app (`apps/ios/swift`, "SevenTree") is a mature, fastlane-deployable SwiftUI + CloudKit + Core Data app — the strongest Swift asset in the repo (see §1). |
| **internal** | `assistant-kit` | (mirror/staging copy) | A lighter-weight copy of `projects/assistant-kit` (`apps/assistant-kit-web`, `apps/assistant-kit-api`) staged under `clients/internal` — looks like the internal-facing deployment target for the assistant-kit product rather than a separate codebase. Not deep-inspected; low independent value, it's the same product as `projects/assistant-kit`. |
| **wanflo** | `wanflo` | SvelteKit (bilingual EN/RU, prerendered, Vercel) | "Composable, white-label business systems" landing site/pitch for a product (listings, content, marketing, SEO, analytics, deals engine) that doesn't appear to exist yet beyond this landing page — currently just the marketing site. |
| **yura** | `hhh-android` | **Kotlin** (Gradle multi-module: `app/` + `packages/configuration-module`) | Native Android app ("HHH" — a session/quiz-style app per its `ROADMAP.md`), with a genuinely reusable typed-configuration library module. Strongest Kotlin asset in the repo (see §1). |
| **yura** | `hhh-reactnative` | React Native / Expo (own nested `.git` repo) | Appears to be an earlier or parallel cross-platform attempt at the same "HHH" product before/alongside the native Kotlin rewrite — has active build-fix and SQLite-migration task notes. Worth comparing timelines with `hhh-android` before assuming one fully supersedes the other. |

Vendored/dependency trees noted but not enumerated per instructions: `.gradle/`, `.kotlin/`,
`build/` under `hhh-android`; `node_modules/`, `.next/`, `.turbo/` under `bestie-workspace`;
`.svelte-kit/`, `build/` under every SvelteKit app; `.venv/` under Python projects; `DerivedData/`
and `.build/` (not found materializing under egor's iOS app in this pass, likely cleaned).

## 4. Root-level workspace manifests

**`go.work`** (Go 1.26) — six modules in one workspace:
```
clients/andrei/projects/shredbx/apps/shredbx-site/go
clients/bestie/projects/bestays/apps/dashboard/go
clients/bestie/projects/bestierealestate/apps/api-chi
clients/egor/projects/myFamilyTree/apps/api/go
projects/sbx/apps/sbx/go
projects/sbx/packages/core/go
```
Every client Go API is a thin consumer of `projects/sbx/packages/core/go` — confirms the "one
shared framework, N thin client apps" architecture described in bestierealestate's CLAUDE.md.

**`pnpm-workspace.yaml`** — ~23 package globs: the `projects/sbx/packages/*` UI/utility library
family, `projects/sbx/apps/sbx/svelte`, `projects/assistant-kit/apps/studio/svelte`, every client's
SvelteKit app (`land-canvas`, `shredbx-site` + its `shredbx-ui` package, `bestays/dashboard`,
`bestierealestate/web-svelte`, `myFamilyTree/web`, `wanflo/web`), plus the games engine family
(`vendor-pixi`, `engine-renderer`, `engine-input`, `game-preset-arcade`, `game-svelte`) and the five
concrete game packages under `shredbx/packages/games/*`.

**`package.json`** (root) — minimal: `"name": "shredbx-workspace"`, pnpm 9.12.0, one override
(`svelte: 5.50.0`). No root-level scripts; build/dev/test orchestration is delegated entirely to
`Makefile` + `Makefile.d/`.

**`Makefile` + `Makefile.d/`** — a two-tier structure:
- `Makefile.d/tools/*.mk` (17 files) — one file per external tool wrapped for `pkg/makerun`
  invocation (per root `CLAUDE.md`: "call a Makefile.d target via `pkg/makerun`, never raw `exec.Command`"):
  `docker`, `dokploy`, `ffmpeg`, `ffprobe`, `git`, `go`, `gradlew`, `magick` (ImageMagick), `op`
  (1Password), `pnpm`, `process` (process-cli, interestingly — see note below), `sbx`, `sips`, `ssh`,
  `swift`, `vault`, `wrangler` (Cloudflare).
- `Makefile.d/workflows/*.mk` (22 files) — one file per repeatable dev/ops workflow:
  `appstore`, `backup-run`/`backup`, `build`, `chat`, `check`, `client`, `data`, `deploy`,
  `external`, `generate`, `infra`, `migrate`, `project`, `qa`, `restore`, `run`, `sbxtest`, `seed`,
  `sqlexec`, `watermark-worker`/`watermark`. This is exactly the "per-service dev/build/deploy/test
  target pattern" the research prompt asked about — it exists, and is fairly comprehensive.
  Top-level root `Makefile` targets include `build`, `dev`, `run`, `up`/`down`, `logs`, `status`,
  `test-core`, `watch-app`/`watch-core`/`watch-project`, `bump-major/minor/patch`, `stage-up/down/status/setup`,
  plus per-client shortcuts (`bestie`, `internal`, `shredbx`, `sbx`).
- Note: `Makefile.d/tools/process.mk` wraps a `process` tool — worth a quick follow-up check
  (not done in this pass) on whether this is unrelated, or an early/independent brush with
  process-cli-like tooling already inside shredbx.

## 5. `infrastructure/`

- **`infrastructure/docker/`** — the real deployment surface: root `docker-compose.yml` +
  `docker-compose.dev.yml`/`.prod.yml`/`.stage.yml`, per-project compose files under
  `projects/{sbx,shredbx}/compose.yml`, per-client `.env.example.*` files (`bestays`, `landcanvas`,
  `shredbx`, `infra`, `staging-infra`), a `plugins/` folder (`claude-runner` Dockerfile, `postgres`
  compose template), and `_archive`/`.archive` folders holding retired compose/Caddy configs
  (old Caddyfiles, an old nextjs Dockerfile) — i.e. there was an earlier Caddy-based deployment
  approach that was retired in favor of the current Dokploy-driven, mirror-only model documented in
  the root CLAUDE.md.
- **`infrastructure/docker/contabo-singapore/`** — the staging server config: its own
  docker-compose, Postgres/Redis staging volumes, migration init scripts, redis-namespace notes.
  Names the actual hosting provider (Contabo, Singapore region).
- **`infrastructure/stage/setup-dokploy.sh`** — single script to bootstrap Dokploy on a fresh stage
  server.
- Overall shape: **one shared Postgres+Redis infra, N Dokploy-deployed app containers, staging vs.
  prod separated by env files and compose overlays**, all driven by the `sbx deploy` / Makefile
  workflow layer rather than manual `docker` commands (which the root CLAUDE.md explicitly forbids).

## 6. `scripts/`

- **`scripts/checks/check-context-integrity.sh`** — a single governance/context-integrity check
  script (likely validates the CLAUDE.md/knowledge-graph consistency described in the root
  CLAUDE.md).
- **`scripts/infra/`** — the server-provisioning and secrets automation layer:
  - `bootstrap/` — a **numbered, ordered hardening script sequence** for a fresh Ubuntu host:
    `00-create-user` → `01-ssh-hardening` → `02-openssh-patch` → `03-ufw-base` → `04-docker-install`
    → `05-ufw-docker` → `06-fail2ban` → `07-unattended-upgrades` → `08-sysctl-hardening` →
    `09-audit-lynis` → `10-dokploy` → `11-server-secrets` → `12-backups-restic` →
    `13-verify` → `14-egress-firewall`, plus a shared `lib.sh` and `bootstrap.sh` driver. This is a
    clean, reusable "provision a production box" runbook — a strong infra-automation candidate.
  - `op/` / `op-sa/` — 1Password (regular + service-account) provisioning and a
    `capability-selftest.sh` to verify vault access works.
  - `db-provision.sh` / `db-provision-tunnel.sh` / `db-backup.sh` / `db-restore.sh` /
    `db-migrate.sh` / `service-restart.sh` / `config-push.sh` / `stage-up.sh` — generic
    Postgres/service lifecycle scripts, reused across projects rather than per-client copies.
- **`scripts/playwright/`** — `artifacts.mjs`/`.d.mts` (typed Playwright artifact helpers) and
  `clean-strays.sh` (cleans stray Playwright processes/artifacts) — small e2e-testing utility
  scripts, reused by the several projects that run Playwright (bestierealestate, sbx app, wanflo,
  myFamilyTree).
- A handful of loose top-level Python/shell scripts (`portfolio-strip-placeholders.py`,
  `portfolio-projects-audit.py`, `portfolio-projects-reconcile.py`, `portfolio-migrate-p11.sh`) —
  one-off portfolio/content migration scripts for the shredbx personal site; low reuse value.

## 7. `externals/`

Three subfolders implementing the "deployment mirror" pattern the root CLAUDE.md describes
(MIRROR-ONLY deploys — Dokploy never pulls the monorepo, only a scrubbed external mirror repo):

- **`externals/registry/`** — one YAML per externally-mirrored project (`assistant-kit.yml`,
  `hhh-android.yml`, `bestierealestate.yml`, `wanflo.yml`, `shredbx.yml`, `myFamilyTree.yml`) —
  presumably declares the mirror repo URL, sync/scrub rules, and which local project it tracks.
- **`externals/staging/`** — a scratch/staging checkout of each project's scrubbed content before
  it's pushed to the mirror (mirrors `myFamilyTree`, `shredbx`, `wanflo`, `assistant-kit`,
  `bestierealestate` — notably no `hhh-android` staging folder despite having a registry entry).
- **`externals/workspaces/`** — actual local git clones of the external mirror repos themselves
  (each has its own `.git/`) for the same five projects — i.e. `staging` is "about to be pushed",
  `workspaces` is "the real external repo, checked out locally" for verification/diffing.

Not deep-inspected per instructions (these are working copies of what's already been inventoried
above as source).

## 8. `docs/`

Large, topical, and uneven in density — 27 top-level subfolders. High-level pass only:

| Folder | Guessed purpose | Notes from a peek |
|---|---|---|
| `ai-assistant` | Product docs for assistant-kit / an AI-assistant pitch | Numbered docs (`01-current-state` → `04-technical-task`) plus a Claude design guide and three pitch decks (internal/partner/client) — looks like assistant-kit was/is also being pitched externally. |
| `audits` | Point-in-time code/endpoint audits | e.g. `2026-05-24-br-endpoint-validation-trace.md` — bestierealestate endpoint audit. |
| `blog` | Blog drafts/research/templates for a personal or product blog | Not read in depth. |
| `books` | Long-form writing | `a-practical-guide-to-feature-driven-development` — the user has been writing an FDD book; directly relevant to sdlc-kit's FDD framing. |
| `cv` | Andrei's CV/resume material | `cv_latest.md`, `experience.md`. |
| `discussions` | Design-discussion transcripts/notes | Includes `schema-system-architecture`, `mvp-0.1-schema-tool-development`, `capability-cross-cutting-analysis` — likely close conceptual ancestors of process-os's schema work. |
| `domain-example` | Worked domain-modeling examples | `fdd-templates`, `property-filter`, `property-sbx` — concrete FDD/entity-modeling worked examples, potentially directly reusable as process-os template inspiration. |
| `handoffs` | Dated session handoff notes | e.g. `2026-04-07-sdlc-evolution-session.md`, `2026-04-27-br-import-pipeline-listing-handoff.md` — session-to-session continuity notes, same genre as sdlc-kit's own eventual handoffs. |
| `jobseeking` | Job-search materials | `10-research` → `40-positions`, unrelated to the SDLC-kit effort. |
| `plans` | **632 files** — the dominant, by far largest subfolder | Dated plan/design docs going back to at least Jan 2026, e.g. `2026-01-24-sdlc-factory-vision.md` (the direct ancestor of this whole effort — worth reading), `2026-01-23-visual-entity-modeling-design.md`, `2026-01-30-unified-types-system.md`, `active-milestones.md` (the live tracker, still being pruned/updated as of Jul 2026). Far too large to catalog; treat as a searchable archive, not a folder to read end-to-end. |
| `prompt-engineering` | Reference material on prompting/context engineering | Includes copies of Anthropic's own best-practices docs. |
| `prototypes` | Standalone HTML/design prototypes | `authz-console`, `footer-signoff`, `snake-carousel`. |
| `reference` | Empty (as of this inventory) | No files present. |
| `research` | Internal research notes | `2026-07-08-sdlc-audit/`, `2026-07-08-sdlc-efficiency-audit.md`, `2026-07-08-context-optimization.md` — **direct prior art for the "keep a running retrospective on process-os efficiency" instruction in sdlc-kit's own CLAUDE.md** — worth reading before or alongside that retrospective. |
| `reviews` | Session/code reviews | `2026-05-18-br-session-review.md`. |
| `tasks` | Mostly archived | Only `.archive/2601-010` present — task tracking has apparently moved elsewhere (the `sbx sdlc` tool tracks tasks directly, per root CLAUDE.md). |
| `test-plans` | Manual verification checklists | e.g. `2026-05-18-br-manual-verification.csv`. |
| `texts` | Misc long-form text | `annotations-online-brainstorming.md`. |
| `user-input` | Raw user-provided source material | Includes `SBX-Technical-Architecture-v0.0.4.md` — likely the original architecture spec for the whole sbx framework, worth reading as prior art. |
| `validation` | Screenshot-based validation evidence | Two PNGs (`knowledge-hub-v-post-fix.png` etc.) — visual proof-of-fix artifacts, not documents. |

Two loose top-level files worth noting: `docs/shredbx-goals.md` and `docs/standards-reference.csv`
(a large CSV, possibly a coding-standards catalog) — not opened in this pass but named clearly
enough to flag as likely relevant to sdlc-kit's own eventual standards work.

---

## Strongest porting candidates (best-effort judgment call)

- **`projects/sbx/packages/core/go`** (and the `sbx` CLI built from it) — the single richest piece of
  prior art in the repo for exactly what sdlc-kit/process-os is building: SDLC task-walking,
  knowledge/decision governance, vault, deployment, entity/schema modeling. Read `docs/plans/2026-01-24-sdlc-factory-vision.md`
  and `docs/user-input/SBX-Technical-Architecture-v0.0.4.md` before designing new process-os
  entities that overlap this territory — don't rediscover lessons this system already paid for.
- **`projects/sbx/packages/*` Svelte/TS UI library family** (`core/svelte`, `ui-image`, `ui-video`,
  `ui-code`, `ui-diagram`, `ui-map`, `ui-calendar`, `units`, `text-template`, `canvas-kit`,
  `canvas-ui`, `ui-source-picker`, `ui-contact`, `ui-seo`, `core/ts/animations`) — small, focused,
  independently-versioned packages, each reused by 2+ apps; a natural seed for a Svelte package base.
- **Games engine family** (`sbx/packages/games/{vendor-pixi, engine-renderer, engine-input,
  game-preset-arcade, game-svelte}` + the 5 concrete games) — a clean, proven engine/product split;
  good template for "framework package + thin product" pattern generally, not just games.
- **`clients/egor/projects/myFamilyTree/apps/ios/swift`** — the strongest Swift/SwiftUI asset found:
  mature SwiftUI + CloudKit + Core Data app with XcodeGen (`project.yml`), fastlane, and a Deploy/
  ExportOptions setup — a strong seed for a Swift/iOS app-starter/bootstrap template, independent of
  whether myFamilyTree itself is ported.
- **`clients/yura/projects/hhh-android/packages/configuration-module`** — the strongest Kotlin
  asset: a standalone, explicitly-modeled ("Decision #0144", `conforms_to: [entity]`) typed
  provider-cache-schema configuration library — both a candidate Kotlin package and a pattern
  (typed config lifecycle) worth generalizing across platforms.
- **`scripts/infra/bootstrap/*`** (the numbered 00–14 server-hardening sequence) — a clean, reusable,
  already-ordered runbook for provisioning a Dokploy-ready Ubuntu host; low risk, high reuse value,
  not tied to any one client's business logic.
- **`clients/bestie/projects/bestierealestate`** (Go API + Svelte web) — not "reusable" in the
  library sense, but the single most mature, most actively engineered client *product* in the repo
  (real tests, CLAUDE.md, ROADMAP.md, DEPLOY.md, HANDOFF.md, e2e suite) — the best example to study
  for "what does a fully-built product on this stack look like end to end."
- **`docs/domain-example/`** and **`docs/books/a-practical-guide-to-feature-driven-development`** —
  worked FDD/entity-modeling examples and a full-length FDD writeup, directly relevant to sdlc-kit's
  own FDD framing; cheap to read, likely to save re-derivation of modeling conventions.

## Client-specific / low-reuse-value (best-effort judgment call)

- **`clients/andrei/projects/land-canvas`** — very early (just a logo, a cover image, and an app
  skeleton); not enough built yet to port anything concrete from.
- **`clients/bestie/projects/bestie-workspace`** — explicitly superseded legacy Next.js/Turborepo
  monorepo (self-marked `-old`/`-outdated`); the Go+Svelte `bestays`/`bestierealestate` rebuilds
  replaced it. Skip unless specifically mining old React component patterns.
- **`clients/wanflo/projects/wanflo`** — currently just a prerendered marketing/landing site for a
  product ("composable white-label business systems") that doesn't appear to be built yet; the
  underlying product concept may be worth discussing, but there's no product code to port.
  yet.
- **`clients/yura/projects/hhh-reactnative`** — likely an earlier/parallel attempt at the same
  product now being built natively in Kotlin (`hhh-android`); probably safe to treat as historical
  unless the native rewrite is confirmed to be a full replacement.
- **`projects/sbx-next`** — too early-stage (mostly planning docs, thin `src/`) to port from yet;
  worth revisiting later if it matures, since it's aimed at the exact same problem as sdlc-kit.
- **Root-level loose portfolio scripts** (`scripts/portfolio-*.py`, `portfolio-migrate-p11.sh`) —
  one-off content-migration utilities for Andrei's personal site; no reuse value outside that
  context.
- **`docs/jobseeking/`, `docs/cv/`, `docs/texts/`** — personal/unrelated content, no bearing on the
  porting effort.
