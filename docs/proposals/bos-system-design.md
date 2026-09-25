# bos — turning bestierealestate into a system plus a thin consumer

Last updated: 2026-09-26
tags: bos, bestierealestate, bestays, go, svelte, decomposition, proposal

Status: **plan approved; Scopes 1–3 (leaf ports, package taxonomy and regroup, remaining Go L0/L1 ports),
3b (group renames) and 3c (regroup into kits by functionality) approved and done on 2026-09-25** (logs:
`docs/plans/2026-09-25-bos-scope-1-workspaces-and-leaf-ports.md`,
`docs/plans/2026-09-25-bos-scope-2-package-taxonomy-and-regroup.md`,
`docs/plans/2026-09-25-bos-scope-3-go-l0-l1-ports.md`,
`docs/plans/2026-09-25-bos-scope-3b-group-renames.md`,
`docs/plans/2026-09-25-bos-scope-3c-kits-by-functionality.md`; Scope 4, Svelte foundation, approved and done on
2026-09-26: `docs/plans/2026-09-25-bos-scope-4-svelte-foundation.md`; Scope 5, the remaining Svelte packages,
the same day: `docs/plans/2026-09-25-bos-scope-5-svelte-ui-packages.md`; Scope 6a, the first seven Go feature
packages, Scope 6b, auth, visitor activity and the job scheduler, and Scope 6c, property and transaction,
the same day:
`docs/plans/2026-09-25-bos-scope-6a-go-feature-packages.md`, `docs/plans/2026-09-25-bos-scope-6b-go-identity-analytics-jobs.md`,
`docs/plans/2026-09-25-bos-scope-6c-go-real-estate.md`). Per `CLAUDE.md`, approval is per
scope: this document fixes the *direction* and the *first scopes*; every later scope gets its own
before/after file tree and its own confirmation before anything is written.

Client-specific values (brand, copy, contacts, hosts) stay in the client repo. This document names
projects (`bestierealestate`, `bestays`) because the repo already does, and nothing more.

---

## 1. The assignment — the end state

When the work is finished, the Go + SvelteKit product that runs in production today as
`bestierealestate` runs **identically** (same routes, same API bytes, same rendered pages, same
database schema), but its source is no longer an app:

- The code lives **once**, in `sdlc-kit/platform/`, as **packages** (libraries) grouped into **kits** (one
  functionality's family: `seo`, `calendar`, `contacts`, …) and two **frameworks** (`bos-go`,
  `bos-svelte`) that assemble kits into a running app. Where a kit's app-side wiring (handlers,
  repositories, migrations, routes, admin pages) lives is open decision D15.
- `bestierealestate` is a **consumer**: a configuration spec, branding, seed content, and a minimum of
  genuinely product-specific code. Its own repo, its own process-os workspace.
- `bestays` (a Next.js app today) becomes the **second consumer** of the same system: same kits, plus
  a new `bookings` kit, minus selling, with its own branding.
- Each consumer deploys as **its own bundle** (own Postgres + pgAdmin, own Svelte web, own Go API,
  own Python assistant). Code only ever receives *connection configuration*.

"Config spec" has three tiers, and only the first two belong to the consumer repo:

| Tier | Examples | Lives in |
|---|---|---|
| Build-time spec | enabled kits, offerings, role names, brand tokens, nav skeleton, env-name mapping | consumer `spec/` |
| Seed content | dictionaries, FAQ items, CMS pages, guides, services | consumer `seed/` |
| Runtime data | anything an admin edits (Appearance, content, listings) | the consumer's own database |

## 2. Evidence (recon, 2026-09-25; read-only, numbers are from source)

| Area | Finding |
|---|---|
| Go API (`api-chi`) | 37.5k non-test LOC (+37.1k test). `main.go` is 3,860 LOC with a 2,266-line `main()`; ~258 routes registered inline (176 under `/api/manage`); no route table, no OpenAPI. Roughly 57% generic HTTP/Postgres adapter code, 34% real-estate, 9% wiring (±10%). `internal/handler` 17.8k LOC (≈11.3k generic), `internal/repository` 5.9k (≈4.7k generic). Roles declared twice; schema name hardcoded 60×; no runtime "enabled modules". |
| Migrations | 310 files = 156 logical migrations, applied by `pkg/database.Migrate` (advisory lock + `schema_migrations`). ≈75–80 real-estate, ≈70–75 shared-package-shaped. |
| Svelte app (`web-svelte`) | 353 `.svelte` + 516 `.ts`, 123.7k non-test LOC. ≈29% property-specific. 229 admin route files with parametric repetition (guides vs services differ by ~13 lines per page). 155 hand-mirrored API types; 52 files each declare `API_BASE_URL`; 302 files use the app's own brand tokens (11k uses); brand strings hardcoded in 178 files; i18n effectively absent. |
| Shared Go libs | 34 packages of `sbx-core` (30.2k src LOC, 25.3k test LOC), 4 of them sub-packages. All stdlib + pgx/redis/aws/jwt etc. per package; `image` alone drags aws-sdk + webp; `auth` alone drags jwt/bcrypt/redis. |
| Shared Svelte libs | 12 packages, 85.3k src LOC + 11.1k test LOC. `core-ui` (44.7k) is nine packages under one name; two packages hardcode the consumer's own tokens; `cms` is self-declared "BR-local". |
| Prior attempt (`sbx.framework/projects/bos`) | Proved the easy slice (≈6 public pages, generic admin, FAQ/guide content, theme←brand←config overlay). Never touched Go or the property domain. Pain: flavor written into one shared app tree; components stored as opaque `.tmpl` text; declared-but-unread claims. |
| Tests as an equivalence gate | Insufficient alone: Go handler tests build their own routers (real route table untested); Playwright has no visual/DOM baseline and needs the live stack. |
| Manifests in shredbx | `project.yml` `configuration:` block (modules/templates/bindings/dictionaries) is inert; `package.yml` layer has drifted. |

Sources: `docs/research/shredbx-bestierealestate-and-capabilities.md`;
`sbx.framework/docs/research/2026-08-06-br-package-inventory.md` (the earlier full package inventory);
four read-only recon passes on 2026-09-25 (Svelte app, Go API, shared libraries, prior BOS attempt);
`docs/research/bestierealestate-decomposition-recon.md`.

## 3. Vocabulary

| Term | Meaning | Test for "does this belong here?" |
|---|---|---|
| **package** | One library, one job, one language, own module and dependencies. `money`, `repository`, `ui-seo`. Lives in a *kit* (section 3.1). | Reusable with no knowledge of any other functionality. |
| **kit** | One *functionality's* family of packages: `calendar` (core, iCal, later integrations, UI), `contacts`, `seo`. It is the top-level folder under `packages/`. Later it also carries its bos wiring — Go handlers, repositories, migrations, `Register`; Svelte routes, components, admin pages; the spec keys it reads — where that lives is D15. The same idea as Python's `process-kit`. | Its name says what it does to a newcomer, without its parent folder. |
| **framework** | Composes kits into an app; no domain of its own. `bos-go`, `bos-svelte`. | Only assembles, configures, runs. |
| **preset** | *Data*: a named bundle of enabled kits, dictionaries, roles, nav skeleton, defaults. `real-estate`. | A choice, not code. |
| **consumer** | preset + brand + seed + overrides + connection config. `bestierealestate`, later `bestays`. | Would differ between two clients. |

`bos` is the name of the whole system (frameworks + kit catalogue + presets). There is **no
"realestate-platform" tier**: everything a platform tier would hold is either domain code (a kit) or a
bundle of choices (a preset). Promote a preset to a tier only if a third vertical demands shell-level
customization; presets are data and kits are already separate, so promotion stays cheap. The preset is
*extracted* by diffing the first two real consumer specs, not designed before the second exists.

The word **capability** is reserved for the 13 SDLC capabilities (`architecture`, `infrastructure`, …)
that govern this workspace's process-os namespaces. Product areas are called **domain areas**.

### 3.1 Package layout and placement rules

One layout for both stacks: `platform/<lang>/{packages/<kit>/<package>, frameworks/<name>}`. Three levels:

| Level | Names | Why it exists |
|---|---|---|
| role | `packages/` · `frameworks/` (a `tools/` may join later) | dependency direction: frameworks depend on packages, never the reverse (the pair Python already has) |
| kit | `seo`, `calendar`, `contacts`, `persistence` … | groups one functionality's packages; a kit grows without anything moving |
| package | `address`, `ical`, `repository` … | one library, one module, isolated dependencies |

Rules (decided 2026-09-25, Scope 3c, after four review rounds — see the decision log):

1. **The top level of `packages/` is a kit, named by functionality.**
2. **Naming test:** a kit name must tell a newcomer what is inside without its parent folder. Generic nouns
   (`property`, `transaction`, `collection`, `dictionary`, `units`) are package names only, under a kit
   that qualifies them (`real-estate/property`, `reference-data/dictionary`).
3. **A package with the same name as its kit is the kit folder** (`packages/seo`, not `packages/seo/seo`);
   every other package is a subfolder (`packages/calendar/ical`). A kit with no package of its own name is
   just a group (`identity/`, `contacts/`). Nested modules already exist in the tree
   (`persistence/repository/postgres`); Go's own stdlib has both shapes (`net` + `net/http`; `text/` is only a group).
4. **Placement test:** which functionality does this package serve? Put it in that kit; create the kit when
   its first package lands, never pre-scaffold. A package no single feature owns gets a shared kit named for
   what it does (`persistence`, `http`, `money`).
5. Directory placement is independent of import identity: module paths and npm names stay verbatim until
   M5 (D7).
6. **Not bos:** the 45 SDLC/workspace-tooling packages in `sbx-core` (`sdlc`, `vault`, `project`,
   `workspace`, `trace`, `dokploy`, …) never enter the product tree.

**Dependency direction:** shared kits (`persistence`, `http`, `money`, `location`, `localization`,
`formatting`, `notifications`, `jobs`, `ui`, `reference-data`) never import feature kits; no cycles; a
feature kit may import another only through a declared edge (an architecture test enforces this later).
Known violation today: `scheduler` (`jobs`) imports `news/feed` — to be inverted so jobs register themselves.

**Full estate behind this** (read-only inventory, 2026-09-25): `sbx-core/pkg` has 89 Go packages — 30
top-level packages in the app's closure (34 modules with sub-packages), 4 more product packages outside it
(`authz`, `land`, `mapoutline`, `utilityreading`), 8 still to classify (`catalogue`, `csvimport`,
`document`, `i18n`, `identitydocument`, `person`, `persistence`, `storage`), 2 superseded stubs (`lease`,
`leaseparticipant`, superseded by `transaction`) and 45 SDLC/workspace tooling. Svelte: 18 packages under
`sbx/packages`, 12 in the closure; `ui-video` fits `media`; `i18n-svelte`, `ui-code`, `ui-diagram`,
`games`, `receipts`, `review-receipts` were seen by name only. The 4 product packages outside the closure
fit existing kits without a new axis.

End state of the closure (✔ ported · ○ later scope):

```
platform/go/packages/
├── seo ✔ · money ✔ · calendar ✔ (+ical) · faq ✔ · cms ✔      a package named like its kit sits at the kit root
├── real-estate/     collection ✔ · property ✔ · transaction ✔
├── reference-data/  dictionary ✔
├── identity/        user ✔ · rbac ✔ · auth ✔
├── contacts/        personname ✔ · phonenumber ✔ · socialnetwork ✔ · contact ✔ (+vcard) · inquiry ✔
├── location/        geocoordinate ✔ · address ✔
├── media/           image ✔ · video ✔
├── news/            rss ✔ · feed ✔
├── persistence/     database ✔ · repository ✔ (+postgres)
├── localization/language ✔ · notifications/notify ✔ · http/httputil ✔
└── analytics/visitoractivity ✔ · jobs/scheduler ✔ (+schedcli)          → 34 modules (30 top-level + 4 nested)

platform/svelte/packages/
├── ui/              core-ui ✔ · animations ✔
├── formatting/      units ✔ · text-template ✔
├── media/           canvas-kit ✔ · canvas-ui ✔ · ui-image ✔ · ui-source-picker ✔
└── seo/ui-seo ✔ · contacts/ui-contact ✔ · calendar/ui-calendar ✔ · location/ui-map ✔          → 12 packages
```

`core-ui` (in `ui`) depends on `formatting/units`: shared kits are imported by others, never the reverse.
Python (`platform/python`) is untouched — it is one family (`process-kit`: a dist-name prefix over a flat
`packages/`) belonging to another track.

## 4. Approach

| Option | Verdict |
|---|---|
| **A. Strangler in place** — import bestierealestate verbatim so it runs from the new layout, then move one slice at a time out of the consumer into kits; equivalence-checked at every step; stoppable at any step | **Chosen (proposed)** |
| B. Greenfield thin consumer, switch at the end | Rejected: repeats the earlier attempt's mistake (the easy 83% looks done; the hard 17% breaks the promise at the end) |
| C. Generate the whole app from a spec | Rejected: Go is unexplored; ~37k LOC does not survive as template text |

Working method carried over from the Python port: **port verbatim first, prove with the source's own
tests, then refactor**; **build to prove, then model** (no process-os definitions from one example;
the second real instance earns a template/action); scoped test runs per package, full runs only in CI.

## 5. Design decisions

| # | Decision | Status |
|---|---|---|
| D1 | Client repo `shredbx/sdlc-bestie-bestierealestate`, mounted at `consumers/clients/bestie-bestierealestate`; later regrouped to `consumers/clients/bestie/bestierealestate` (a `git mv` of the submodule path + one `libraries:` path edit) | Confirmed by user |
| D2 | Each consumer is its own deployable bundle (Postgres + pgAdmin + web + api + assistant); code takes connection config only; no multi-tenancy | Confirmed by user |
| D3 | `bestays` is consumer #2: same kits + `bookings` kit − selling, own branding | Stated by user |
| D4 | System `bos` = frameworks + kits + presets; no separate realestate-platform tier | Proposed; user's follow-up answers assumed it but did not explicitly confirm |
| D5 | Strangler-in-place approach (section 4) | Proposed |
| D6 | Go: **one module per package** under one `go.work` (34 modules, 4 nested as sub-packages). Reason: `sbx-core` is one module that drags aws-sdk/webp/jwt/redis onto every consumer; per-package modules make "enabled kits" true at the dependency level. Revisit at M5: about a dozen product packages are under 150 LOC, so one module each is heavy — a kit may become the Go module, with packages inside it | Proposed |
| D7 | Names stay **verbatim** through M0 (`github.com/shredbx/sbx-core/pkg/<name>` module paths — nested module paths are legal, longest prefix wins — and `@sbx/*` npm names); final names decided at M5 via a mechanical codemod. Directories are grouped from Scope 2 (D13): placement and import identity are independent | Proposed |
| D8 | Oracle runs on deterministic seed + synthetic users only, never on production backups | Proposed |
| D9 | Consumer routes are committed one-line re-export shims (generated from the spec, drift-checked by `conform`) **or** a build-time composed directory; decided by the M1 spike, lean = shims (an override is then just a real file at that path) | Open — spike decides |
| D10 | Kit join manifests and presets are process-os **records** (`records/sbx-sdlc-kit/<capability>/{kit,preset}/`), with schemas modeled per-definition after the shape is proven; namespace chosen capability-first (the 13 SDLC capabilities) at that time | Proposed; nothing modeled yet |
| D11 | Framework reads all env-specific settings from config; neutral env names with a consumer-side mapping so production's existing secrets keep working until cutover; the schema name becomes one config value | Proposed; scheme settled in M2 |
| D12 | "Selling" must be switchable by configuration, not by deleting code. `transaction` is a package inside the `real-estate` kit next to `property` (not a kit of its own); property offerings (`for_sale`/`for_lease`) are a spec key, and switching `transaction` off is a per-package setting, decided when `bestays` exists and two consumers can be compared | Proposed; revised 2026-09-25 (was: a `property-catalog` / `transactions` kit split); confirmed against code in M4 |
| D13 | Package layout and placement rules (section 3.1): `platform/<lang>/{packages/<kit>/<package>, frameworks/}`; a kit is a functional family named by functionality (naming test); a package named like its kit is the kit folder; the same kit names on both stacks | Accepted 2026-09-25 after four review rounds; supersedes the Scope 2 taxonomy (`datatypes`, `foundation` + domain groups) and the Scope 3b renames |
| D14 | "capability" is reserved for the 13 SDLC capabilities; product areas are "domain areas" | Accepted with D13 |
| D15 | Where a kit's bos wiring (handlers, repositories, migrations, routes, admin pages) lives. Wiring imports bos-go's `Module` contract, so it cannot sit under `packages/` if packages never import frameworks. Candidates: (a) a separate `platform/<lang>/kits/<kit>` layer; (b) inside the kit folder, with the contract extracted as a tiny leaf package | Open — the FAQ slice in M2 decides; nothing in Scopes 3c–7 depends on it |

## 6. Final structure

### 6.1 sdlc-kit (the framework side)

```
sdlc-kit/
├── platform/
│   ├── CLAUDE.md                          layout · naming test · placement · dependency direction (auto-loaded here)
│   ├── python/ …                          (unchanged: packages/×7, frameworks/{process-framework, agent-framework})
│   ├── go/
│   │   ├── go.work
│   │   ├── packages/<kit>/<package>       34 modules, verbatim ports of sbx-core pkg/*  (section 3.1, table 6.3)
│   │   └── frameworks/bos-go/             Module contract · buildRouter · config · migration composer · middleware chain
│   ├── svelte/
│   │   ├── pnpm-workspace.yaml · package.json · pnpm-lock.yaml
│   │   ├── packages/<kit>/<package>       12 packages (section 3.1, table 6.4); core-ui decomposed into ui-* packages (provisional)
│   │   └── frameworks/bos-svelte/         app shells (public / admin / minimal) · module registry · site-config · theme · API client · auth hooks
│   │   kit wiring (Go: handlers · repositories · register · migrations; Svelte: routes · components · admin pages)
│   │   lands in M2/M3; where it lives is open (D15)
│   └── (later) bookings kit for bestays, in both stacks
├── processos-workspace/
│   ├── definitions/sbx-sdlc-kit/<capability>/…    kit / preset / consumer-spec schemas, port + equivalence actions,
│   │                                              go/svelte package templates — each created only after per-definition approval
│   └── records/sbx-sdlc-kit/<capability>/{kit,preset}/…    one join manifest per kit (go half ↔ svelte half ↔ provides ↔ requires); the real-estate preset
├── consumers/clients/
│   ├── bestie/                            existing submodule (main session; assistant + bestays Next.js today)
│   └── bestie-bestierealestate/           new submodule (later: bestie/bestierealestate/)
└── docs/ …
```

### 6.2 The consumer (`bestierealestate`) — thin

```
consumers/clients/bestie-bestierealestate/          (its own repo and process-os workspace)
├── processos.yaml                       libraries: sbx-sdlc-kit (readonly)
├── README.md
├── spec/
│   ├── consumer.yaml                    preset: real-estate · kits on/off · offerings · roles
│   ├── brand.yaml                       tokens · fonts · wordmark · asset refs
│   ├── site.yaml                        seed for header nav · footer · content sources
│   ├── connections.yaml                 env-name mapping (no secrets)
│   └── dictionaries/                    property types · title deeds · amenities · languages
├── seed/                                FAQ · CMS pages · guides · services
├── apps/
│   ├── api/                             main.go (calls bos-go with the spec) + product/ (BR-only decisions: similar/related ranking …)
│   ├── web/                             svelte.config.js · vite.config.ts · src/{app.css (brand), routes/ (shims + overrides, e.g. home), static/}
│   ├── watermark-worker/  backup/       carried verbatim; their own pass later
│   └── assistant/                       the Python service (main session); reached only by URL
├── migrations/                          only what no kit owns (expected near-empty; kit-owned versions keep identical IDs + content)
├── deploy/                              bundle spec: postgres + pgadmin + web + api + assistant (M6)
└── tests/oracle/                        equivalence harness, kept as the regression suite
```

Success measure (audited at M5): hand-written consumer code is a small fraction of today's ~180k LOC;
brand tokens appear only in `brand.yaml`/`app.css`; nothing in `platform/` names a client.

### 6.3 Go packages — where each of the 34 goes (verbatim, `platform/go/packages/<kit>/<package>`)

Layer = dependency layer within the closure (L0 = no in-repo imports). Src/test LOC from source.
✔ = ported. Nested modules sit inside their parent's folder. A package named like its kit is the kit folder.

| Path under `packages/` | Src / test LOC | Layer | Notes | Status |
|---|---|---|---|---|
| `money` | 843/911 | L0 | | ✔ |
| `seo` | 137/118 | L0 | | ✔ |
| `location/geocoordinate` | 65/85 | L0 | | ✔ |
| `location/address` | 409/761 | L1 | | ✔ |
| `contacts/phonenumber` | 56/111 | L0 | | ✔ |
| `contacts/personname` | 56/187 | L0 | | ✔ |
| `contacts/socialnetwork` | 109/187 | L0 | | ✔ |
| `contacts/contact` | 1,216/417 | L2 | | ✔ |
| `contacts/contact/vcard` | 677/499 | L3 | nested | ✔ |
| `contacts/inquiry` | 451/279 | L2 | | ✔ |
| `localization/language` | 60/28 | L0 | | ✔ |
| `notifications/notify` | 272/122 | L0 | Telegram + SMTP notifier (`telegram.go`, `smtp.go`) | ✔ |
| `persistence/database` | 965/588 | L0 | migrations, pool (bos-go) | ✔ |
| `persistence/repository` | 411/557 | L0 | | ✔ |
| `persistence/repository/postgres` | 1,149/660 | L1 | nested | ✔ |
| `http/httputil` | 405/429 | L1 | | ✔ |
| `jobs/scheduler` | 955/915 | L2 | imports `news/feed` today (edge to invert); ported as is | ✔ |
| `jobs/scheduler/schedcli` | 232/0 | L3 | nested | ✔ |
| `identity/rbac` | 374/362 | L0 | | ✔ |
| `identity/user` | 412/112 | L1 | | ✔ |
| `identity/auth` | 2,631/3,189 | L2 | | ✔ |
| `news/rss` | 1,937/2,037 | L0 | | ✔ |
| `news/feed` | 343/221 | L1 | | ✔ |
| `reference-data/dictionary` | 909/638 | L1 | flat code/label lookup lists (property-type, land-size-unit, amenities) | ✔ |
| `cms` | 1,561/1,497 | L2 | self-declared BR-local; ported verbatim, genericize or keep is decided in its slice | ✔ |
| `faq` | 580/482 | L2 | | ✔ |
| `media/image` | 2,265/1,541 | L0 | coupled to Postgres and S3 | ✔ |
| `media/video` | 855/513 | L0 | | ✔ |
| `calendar` | 797/413 | L2 | | ✔ |
| `calendar/ical` | 893/878 | L3 | nested | ✔ |
| `analytics/visitoractivity` | 1,675/1,945 | L2 | 13 Postgres tests need `VISITOR_ACTIVITY_TEST_DSN` | ✔ |
| `real-estate/collection` | 292/246 | L1 | | ✔ |
| `real-estate/property` | 4,939/3,256 | L2 | | ✔ |
| `real-estate/transaction` | 1,290/1,073 | L2 | selling is switchable (D12); 4 skipped tests wait for the app (Scope 7) | ✔ |

### 6.4 Svelte/TS packages — where each of the 12 goes (`platform/svelte/packages/<kit>/<package>`)

| Path under `packages/` | Src LOC | Notes | Status |
|---|---|---|---|
| `formatting/units` | 402 | pure TS (date, number, area, land units); depended on by `core-ui` and `ui-map` | ✔ |
| `formatting/text-template` | 115 | pure TS | ✔ |
| `ui/animations` | 1,821 | pure TS; no tests (author some); needed by `core-ui` | ✔ |
| `ui/core-ui` | 44,748 | ported as ONE package first (needs `animations` + `units`); decomposed in M1 into `ui-primitives`, `ui-layouts`, `ui-navigation`, `ui-sections`, `ui-viz`, `ui-blocks`, `ui-theme`, `ui-analytics`, `ui-language` (provisional — only BR-imported subpaths) | ✔ |
| `seo/ui-seo` | 1,285 | imports `core-ui` (2 files); `$app/state` coupling in `SeoHead` only | ✔ |
| `media/canvas-kit` | 4,474 | pure TS, LOW drift | ✔ |
| `media/ui-image` | 4,013 | imports `core-ui` (8 files) | ✔ |
| `media/ui-source-picker` | 1,137 | zero tests | ✔ |
| `media/canvas-ui` | 14,299 | zero tests, depends on `ui-source-picker`, 8 deep imports into `core-ui`'s `Modal.svelte`, token contract with no declarations | ✔ |
| `calendar/ui-calendar` | 3,851 | declares `core-ui`, imports it 0 times | ✔ |
| `contacts/ui-contact` | 1,172 | imports `core-ui` (2 files); hardcodes consumer tokens | ✔ |
| `location/ui-map` | 7,995 | declares `core-ui`, imports it 0 times; parcel/plot code | ✔ |

`workspace:*` breaks `pnpm install` for a package whose declared workspace dependency is missing, so
the Svelte port order is dependency-driven: `animations` → `core-ui` → the `ui-*` packages.

### 6.5 App-owned code → kits (provisional; each kit's boundary is proven only when it lands)

| Kit | Go half (from `api-chi`) | Svelte half (from `web-svelte`) |
|---|---|---|
| identity | `internal/handler` auth/users/audit/reset, roles → spec | `(minimal)` login/callback, `admin/users`, `admin/audit`, `admin/reset-requests`, `manage/account` |
| cms | `internal/siteconfig`, `internal/page` (generic part), `internal/localization`, cms handlers | `manage/pages/**` (appearance header/footer/theme), sections renderer, public about/terms/privacy/cookies/services/guides |
| seo | seo endpoints, sitemap/robots/llms | `manage/seo`, SEO tab pages, `robots.txt`/`sitemap.xml`/`llms.txt` routes |
| media | `internal/watermark`, `canvas`, `media`, `imageimport` | `lib/components/media`, `manage/tools/media-canvas`, `manage/marketing/watermarks` |
| faq | `RegisterPublic/ManageFaqRoutes` | `manage/faq/**`, public `/faq` |
| contacts | contact/inquiry handlers, `internal/qr` | `lib/components/contact`, `manage/contacts/**`, `manage/inquiries/**`, public `/contact` |
| calendar · news · jobs (scheduler) · analytics · documents · reference-data (dictionary) | their handlers/repositories (`internal/newsfeed`, `feedcache`, `document`, `registry`…) | `manage/calendar`, `manage/news`, `admin/schedules`+`admin/backup`, `lib/analytics`+`manage/analytics`, `manage/documents`, `manage/content/dictionaries`+`manage/tags` |
| real-estate (property) | `internal/propertyfilter`, `autotag`, `areageo`, `agent`, collections, similar/related (extension seam → consumer) | public `/properties/**`, `/search`, `/p/[id]`, `manage/properties/**`, `manage/content/properties/**`, `manage/agents` |
| real-estate (transaction) | `internal/deal`, transaction handlers | `manage/transactions/**`, public `/sell` |
| assistant | `aiassistant`, `assistantchat`, `assistantcapabilities` — **held in the consumer**; integrates later with the main session's agent-framework | `manage/ai-assistant/**` — held in the consumer |
| (framework) | route assembly, middleware order, config, DI | app shells, `NAV_TREE` → derived from kit manifests, API client, auth hooks, site-config loader |

Kit names in this table are the kit folders of section 3.1 (`real-estate` covers both the property and transaction slices). Where a slice's wiring lives is D15.

## 7. Equivalence oracle — how "identical" is proven

Built before any refactor, captured against the **original** running read-only from shredbx, so
"before" is the source of truth (where it runs is decided when the oracle is built).

1. **Schema** — apply all migrations to an empty DB; compare `pg_dump --schema-only`. Kit-owned
   migrations keep identical version IDs and content, because production's `schema_migrations` already
   records them.
2. **API** — replay a request corpus over all routes × roles against a seeded DB; compare raw bytes
   after scrubbing UUIDs/timestamps. Byte order matters: response shaping depends on Go's embedded-struct
   shadowing rules, so **move code, never reshape types** during the split.
3. **Web** — Playwright crawl of public + admin routes with seeded users: normalized SSR HTML and
   screenshots at 3 viewports.
4. **Route table** — dump method, path, middleware, permission per route (requires extracting a
   `buildRouter`); must equal baseline.
5. **Existing suites stay green** — 925 Go tests (integration via Docker), 180 vitest files, 66
   Playwright specs.

Do-not-"fix"-during-the-split list (each is observable behavior): CORS methods omit PATCH; mixed
rate-limit key functions; upper- vs lower-case error codes; route registration order; per-handler
`Cache-Control`.

**Defects found in the originals while porting** are recorded, never fixed inside a port (the port is
byte-identical and the oracle must equal the original). Each needs the user's decision on where it is fixed: in the
original first (then the port is re-synced), or in the kit's slice as a declared behavior change.
1. **`identity/auth`, magic-link onboarding keeps the raw refresh token.** `magic_link_service.go`
   `CompleteOnboarding` creates its session with `tp.RefreshToken` (line 218), while `auth_service.go` stores
   `HashToken(tp.RefreshToken)` (`createSessionAndTokens`, the original's "F1" fix) and `Logout` / `RefreshToken`
   look sessions up by `HashToken(presented)`. Reproduced in a scratch copy (2026-09-26): the stored raw value can
   never match a lookup, so a magic-link session cannot be refreshed (`ErrSessionNotFound`) and `Logout` for it
   silently does nothing (no JTI revocation). It is a functional bug plus an F1 hygiene violation (a raw token in
   the DB and the cache), not an exposed credential. Read only: the magic-link path records no IP or user agent,
   so session pinning never applies to those sessions. The fix is one hunk plus tests.
   Details: `docs/plans/2026-09-25-bos-scope-6b-go-identity-analytics-jobs.md`.
   **Decision (2026-09-26): fix it in the original first.** This track never edits the originals, so the fix
   (`HashToken(...)` on that line, and no token in the cached copy, plus tests) is the user's to make there; `identity/auth` is then re-synced from
   the original in one small scope. Scope 7 captures the oracle from the original as it then stands, so the fix
   should land first; if it has not, magic-link onboarding is a declared difference between the oracle and the
   re-synced port.

## 8. Ladder

| Rung | Approved scopes (estimate) | What | Gate |
|---|---|---|---|
| **M0 Baseline** (scopes below) | ≈12 (10 done) | Port the 34 Go + 12 Svelte libraries verbatim with their own tests green; import bestierealestate verbatim so it runs from the new layout; capture the oracle | tests green, oracle captured |
| M1 `bos-svelte` | ≈5 | Shell + module registry; package conventions settled; brand-token codemod; one API client; spec v0; decompose core-ui | pixel + DOM diff = 0 |
| M2 `bos-go` | ≈4 | `Module` contract, `buildRouter`, config, single roles list, migration composer; proven on one vertical slice: **FAQ** (small, generic, public + admin, already `Register*Routes`-shaped). The slice also settles where kit wiring lives (D15) | route-table diff = 0 |
| M3 kits | ≈8 | identity → cms → seo → media → contacts → calendar → news → jobs/analytics/documents, one slice each | oracle slice green per kit |
| M4 real-estate | ≈3 | the `real-estate` slices (property, transaction); consumer keeps only product decisions via extension seams | full oracle green |
| M4b second consumer | ≈3 | `bestays` on the same kits + `bookings` − selling; extract the preset; the falsifiability test of every kit boundary | bestays needs config + `bookings` only |
| M5 Thin-out + regroup | ≈2 | consumer footprint audit; final names; a kit may become the Go module (D6); move to `clients/bestie/bestierealestate` | full oracle green |
| M6 Deploy cutover | own plan | bundle spec and/or the mirror/vendor pipeline redone for a multi-repo layout; parallel run; nothing touches production without separate approval | separate approval |

About 37 approved scopes before M6 (33 at the start; Scope 6 turned out to be three scopes and Scope 7 is likely
two), ±30%; the FAQ slice gives the first real measurement.

### 8.1 The per-kit loop

| Stage | What happens | Gate |
|---|---|---|
| 1. Port libraries | Verbatim copy into `packages/<kit>/…`, proven by the source's own tests | same pass/skip/fail counts as the original |
| 2. Import the app | `api-chi` and `web-svelte` copied verbatim into the consumer and run from this repo; the oracle is captured against the original | oracle equals the original |
| 3. Slice a kit | Move the kit's Go handlers, repositories, migrations and `Register`, and its Svelte routes, components and admin pages, out of the consumer into the kit. Code moves, it is never reshaped | oracle slice green: route table, API bytes, DOM, schema |
| 4. Spec it | The consumer enables the kit by a config key and its own copy is deleted | oracle green, consumer LOC drops |
| 5. Second consumer | `bestays` needs only config plus `bookings` | boundaries hold, preset extracted |

Interfaces and integrations are not separate stages. *Interfaces* already exist in the source (the
repository interface, the notify provider, image storage, the dictionary backends, the domain/adapter
split); each kit's requires/provides is recorded when its slice lands, not designed ahead. *Integrations*
are adapter modules behind those interfaces (`repository/postgres` already is one); this migration adds
none — Google Calendar and similar are growth after M4b, because the goal is an identical app. Once the
loop has survived three kits it may be proposed as a process-os process (separate approval).

Policy: **port only what the app uses** (34 Go modules, 12 Svelte packages). Other product packages
(`authz`, `land`, `mapoutline`, `utilityreading`, …) wait for a consumer that needs them.

M0 scopes (provisional order; each gets its own before/after tree and approval). The Svelte order is
dependency-driven: `ui-image`, `ui-contact` and `ui-seo` import `core-ui`, and `ui-map`/`ui-calendar`
declare it, so `core-ui` — which needs `animations` and `units` — comes first.
1. **Scope 1 (done):** workspaces, 8 Go leaf packages, 3 pure-TS packages.
2. **Scope 2 (done):** package taxonomy and regroup (no new code); superseded by 3b and 3c.
3. **Scope 3 (done):** the remaining Go L0/L1 packages (13 modules): address, database, repository (+postgres),
   httputil, user, rbac, dictionary, rss, feed, video, image (the unused `watermark/` sub-package stays
   behind), collection.
4. **Scope 3b (done):** group renames for clarity (`values` → `datatypes`, `engagement` → `crm`).
5. **Scope 3c (done):** regroup into kits by functionality (section 3.1); 145 renames, no new code.
6. **Scope 4 (done):** Svelte foundation: `ui/animations`, then `ui/core-ui` (one package, verbatim).
7. **Scope 5 (done):** Svelte `seo/ui-seo`, `media/ui-image`, `contacts/ui-contact`, `location/ui-map`,
   `calendar/ui-calendar`, then `media/ui-source-picker` and `media/canvas-ui` (canvas-ui depends on ui-source-picker). All 12 Svelte
   packages are ported.
8. **Scope 6, three scopes** (split by risk, not by layer: every in-repo dependency of the 13 remaining Go
   modules was already ported, except the nested ones):
   - **6a (done):** `faq`, `cms`, `contacts/inquiry`, `contacts/contact` (+vcard), `calendar` (+ical); 7 modules,
     hermetic tests.
   - **6b (done):** `identity/auth`, `analytics/visitoractivity`, `jobs/scheduler` (+schedcli); jwt, redis, bcrypt and
     env-dependent tests; `scheduler` imports `news/feed` (the known edge to invert).
   - **6c (done):** `real-estate/property`, `real-estate/transaction`; `testdata` fixtures; 4 tests skipped
     unconditionally until the app is imported. All 34 Go modules are now ported.
9. **Scope 7:** baseline import of the bestierealestate apps + oracle capture (likely two scopes). The `auth`
   fix (section 7) should land in the original before the oracle is captured; the re-sync of `identity/auth`
   afterwards would be one more small scope.

Sizing honesty: the earlier inventory priced the library port alone at ≈195 half-day slices under a
template-row method. Real packages drop the row/template overhead, but the app-side extraction
(≈16k generic Go adapter LOC, ≈70k generic Svelte LOC) is on top. This is a multi-month track; every
rung leaves bestierealestate runnable and oracle-green.

## 9. Scope 1 — workspaces and Wave 1 (done)

**Goal:** prove the port loop on both stacks with the safest packages, and stand up the two workspaces
and the client repo. Zero framework code, zero process-os definitions.

**Tasks**
1. Persist the recon (`docs/research/bestierealestate-decomposition-recon.md`) and index it.
2. Bootstrap `shredbx/sdlc-bestie-bestierealestate` (initial commit) and add it as a submodule.
3. `platform/go/`: `go.work` + 8 leaf packages, each with an authored `go.mod`, sources copied verbatim.
4. `platform/svelte/`: pnpm workspace root + 3 pure-TS packages, sources copied verbatim.
5. `.gitignore` (`node_modules/`, `.svelte-kit/`), `go-ci.yml`, `svelte-ci.yml`.
6. Milestone plan log `docs/plans/2026-09-25-bos-scope-1-workspaces-and-leaf-ports.md` (per precedent).

**Exit gates (evidence, not assertion)**
- Ported sources are byte-identical to the originals (`diff -r`, excluding authored `go.mod`).
- Go: `go vet`, `gofmt -l` empty, `go test` per module with **the same test counts and results as the
  original**. TS: `vitest run` per package, same counts (units 81 cases, text-template 10).
- Scoped runs per package; the full battery only in CI.

**Known facts and risks**
- The eight Go packages import nothing in-repo; only `language` has a third-party dependency
  (`golang.org/x/text`); tests use only the standard `testing` package.
- Source declares `go 1.26`; the local toolchain is 1.25.6, so the first `go test` auto-fetches 1.26
  (needs network). `go mod tidy` also needs network for `go.sum`.
- `notify` turned out to be a Telegram + SMTP notifier; its kit is assigned when read.
- Pushing the client repo's initial commit is an outward-facing action and needs explicit confirmation.

## 10. Boundaries

- Production is never touched; the shredbx originals stay read-only source until M6.
- Out of this track: the AI-assistant service (main session's `agent-framework`; the BR surfaces are
  carried in the consumer until integration), `watermark-worker`, `backup`, and the deploy pipeline.
- No process-os definitions from a single example. First candidates, once M1/M2 prove the shape: the
  kit manifest schema and the consumer-spec schema; then a `port-package` action and Go/Svelte package
  templates after the second/third port; then a `check-equivalence` action. Each gets its own approval.
- Coordination: the main session works on `feature/agent-framework` and the `bestie` client repo;
  expect a trivial merge on `.gitmodules`. The chat contract stays framework-neutral
  (`{message} → {reply}`), so a Svelte `bestays` can reuse it (`core-ui` already ships `ChatWidget`).
- Worth checking when `bestays` starts: BR has `migration-packs/2607-001-legacy-supabase-properties`;
  by its name it may already hold the Supabase→Postgres path (unverified).

## 11. Decision log

- 2026-09-25 — D1, D2, D3 recorded from the user (see section 5).
- 2026-09-25 — Layering discussed (bos system; frameworks/kits/presets; no realestate-platform tier);
  D4 recommended.
- 2026-09-25 — User requires: plan saved as a file, final structure shown, first scope and its
  after-structure shown, *then* confirmation, commit, proceed. This document is that saved plan.
- 2026-09-25 — User confirmed the plan, the final structure and Scope 1 ("so far confirmed"),
  including the initial push to the client repo. D4–D8 and D10–D12 were on the table as *Proposed* and
  were not vetoed, so they are treated as accepted for Scope 1 and stay revisitable; D9 stays open until
  the M1 spike.
- 2026-09-25 — Scope 1 done: workspaces, 8 Go and 3 TS packages ported verbatim and proven against the
  originals, CI added, client repo bootstrapped and mounted. See the Scope 1 log for evidence and the three
  things found on the way (a `.gitignore` rule that swallows `src/lib/`, one comment-only gofmt finding,
  and `./...` matching nothing at a `go.work` root).
- 2026-09-25 — User, reading the packages: "see no scopes separations, is it good?" Honest finding: a
  flat `packages/` was a design gap — there was no placement taxonomy, the doc drew `packages/` flat, and
  directory grouping had been wrongly deferred together with import renaming (D7). Taxonomy proposed
  (section 3.1); the word "capability" collides with the 13 SDLC capabilities, so product areas are
  "domain areas" (D13, D14).
- 2026-09-25 — User: "looks better and more clear now", then "commit plan then lets go, working along
  this worktree" → Scope 2 (taxonomy + regroup) approved, executed in this worktree, nothing pushed. The
  earlier "13 Go ports" proposal became Scope 3, placed into the groups.
- 2026-09-25 — Scope 2 done: 11 packages regrouped (101 pure renames, tests unchanged: 211 Go, 487 TS),
  `platform/CLAUDE.md` and the completion-report shape added. See the Scope 2 log.
- 2026-09-25 — Scope 3 done: 13 Go modules ported verbatim into their groups (address; database,
  repository + postgres, httputil; user, rbac; dictionary, rss, feed; image, video; collection). Per-module
  results equal the original exactly: 629 pass, 16 skip, 0 fail. New convention: in-repo Go dependencies as
  `require … v0.0.0` + relative `replace` (verified to coexist with `go.work`). Go L0/L1 is complete
  (21 of 34 modules). See the Scope 3 log.
- 2026-09-25 — Scope 3b: the user, reading `platform/go/packages/`, found `values` unclear and asked for
  the other names to be checked. All eight reviewed: `values` → `datatypes` (the word is used in ~50 other
  tracked files in unrelated senses), `engagement` → `crm` (marketing jargon); `foundation`, `identity`,
  `content`, `media`, `analytics`, `real-estate` kept (`core` and `infra` weighed and rejected for
  `foundation`: they collide with `core-ui`/`sbx-core` and the `infrastructure` SDLC capability). "So far
  this way": the names stay revisable. The Scope 2 and 3 logs keep the names of their time.
- 2026-09-25 — Finding while ordering the Svelte ports: `ui-image` (8 files), `ui-contact` (2) and
  `ui-seo` (2) import `core-ui`; `ui-map`/`ui-calendar` declare it unused; `core-ui` needs `animations` +
  `units`. Svelte order is dependency-driven (section 8); the earlier "adapters first, core-ui last" order
  was wrong.
- 2026-09-25 — Scope 3c review, four rounds. (1) The user, on `datatypes/seo`: "seo is seo, address-book is
  address … impossible to maintain" — a scope should be a **kit**, a family of packages that will grow (a
  calendar gains many integrations), wrapped later by frameworks, like Python's `process-kit`. (2) `kits/` as
  a folder was rejected: Python has `packages/` + `frameworks/` and the kit is the family, not a role folder.
  (3) `seo/seo`, `money/money`: a structural stutter, and the design had been sized from the app's closure
  only; the full estate was inventoried (89 Go / 18 Svelte packages; 45 Go packages are SDLC tooling and
  stay out) → rule "a package named like its kit is the kit folder". (4) `property` rejected as a generic kit
  name → `real-estate`; naming test written down; `dictionary` → `reference-data`; `transaction` moved into
  `real-estate`. D12 revised, D13 replaced, D15 opened. The process (per-kit loop, section 8.1) and the ladder
  were confirmed with "so far yes". Plan: `docs/plans/2026-09-25-bos-scope-3c-kits-by-functionality.md`.
- 2026-09-25 — Scope 3c done: 145 renames (129 Go, 16 TS), byte-identical except one `replace` line in each of
  four `go.mod` files, plus `go.work` and the lockfile importer paths. Go 840 pass / 16 skip / 0 fail equal to the
  baseline per module; TS 81 / 10 / 396. See the 3c log.
- 2026-09-26 — Scope 4 done: `ui/animations` (25 files) and `ui/core-ui` (231 files) copied verbatim — 256 files,
  byte-identical to the source; `animations/package-lock.json` and `node_modules` left behind. `core-ui`: 21 test
  files / 251 tests and `animations`: `tsc --noEmit` exit 0, both equal to the original; the existing three
  packages unchanged (81 / 10 / 396). Lockfile +100 packages. Finding: `core-ui`'s `jsdom` dev dependency changed
  the resolved peer suffix of the existing packages' vitest entry (`vitest@4.1.11(vite@8.3.1)` →
  `…(jsdom@25.0.1)(vite@8.3.1)`); no version changed and their suites are unchanged. Svelte 5.57.1 / vitest 4.1.11
  resolved against the original's 5.50.0 / 4.1.0 (accepted). 5 of 12 Svelte packages ported. See the Scope 4 log.
- 2026-09-26 — Scope 5 done: the remaining seven Svelte packages (`seo/ui-seo`, `contacts/ui-contact`,
  `calendar/ui-calendar`, `location/ui-map`, `media/{ui-image, ui-source-picker, canvas-ui}`) copied verbatim — 164
  files, byte-identical. The five tested packages equal a pre-port baseline measured on a copy of the original
  source (ui-seo 10, ui-contact 12, ui-calendar 55, ui-map 160, ui-image 45 tests; 22 files); the two untested ones
  (48 files) have no runtime gate until Scope 7. Findings: (1) pnpm 11 exits non-zero on unreviewed dependency
  build scripts — `core-js` (via `canvas-ui` → `jspdf` → `canvg`) and `esbuild@0.21.5` (via a vite 5 dev dependency
  of `canvas-ui` and `ui-source-picker`) — so `pnpm-workspace.yaml` now denies both explicitly with `allowBuilds`;
  this was not in the approved tree and was needed for the frozen-install gate. (2) The lockfile gained 64 packages
  and lost one entry (`vitefu@1.1.3` unsuffixed → peer-suffixed); no version and no existing importer entry
  changed. (3) The baseline is a copy under default tool resolution, not the original workspace's own pins.
  All 12 Svelte packages are ported. Client information for `bestierealestate` is deferred to the client library
  (the main session's `clients` scope), to be registered after alignment.
- 2026-09-26 — Scope 6a done: seven Go modules (`faq`, `cms`, `contacts/inquiry`, `contacts/contact` + `vcard`,
  `calendar` + `ical`) copied verbatim — 41 source files, byte-identical — plus 7 authored `go.mod` and 7 `go.sum`;
  `go.work` 21 → 28 paths (28 of 34 modules). Tests per module equal a baseline measured on the original module
  (326 pass / 0 skip / 0 fail; the 21 earlier modules still 840 / 16 / 0, so 1,166 / 16 / 0 in all). Findings:
  (1) `replace` directives do not propagate, so a module replaces its whole in-repo dependency closure, not just its
  direct dependencies — `vcard` requires two in-repo modules and needs nine `replace` lines; the closure was
  computed from the ported `go.mod` files. (2) Direct third-party versions equal the original pins (`uuid` v1.6.0,
  `pgx/v5` v5.9.1), checked after tidy because tidy resolves a missing requirement to the latest version;
  indirect versions come from tidy, as in Scope 3. (3) The same seven files are not gofmt-clean in the originals; they
  are formatted in their own commit (25 insertions / 23 deletions). (4) No shared kit imports a feature kit
  (`scheduler` → `news/feed` remains for 6b); the feature-to-feature imports `faq`/`cms` → `seo` and
  `cms` → `contacts/socialnetwork` are declared as kit edges when kit wiring lands (D15). (5) M0 is re-estimated from 8 to about 12 scopes (Scope 6 became three,
  Scope 7 is likely two) and the ladder from about 33 to about 37; still inside the ±30% band. `cms` is ported
  verbatim even though it is self-declared client-local; whether to genericize it is decided in its slice.
- 2026-09-26 — Scope 6b done: four Go modules (`identity/auth`, `analytics/visitoractivity`, `jobs/scheduler` +
  `schedcli`) copied verbatim — 63 source files, byte-identical — plus 4 authored `go.mod` and 4 `go.sum`; `go.work`
  28 → 32 paths (32 of 34 modules) and two new kits, `analytics` and `jobs`. Tests per module equal a baseline measured
  on the original module with `VISITOR_ACTIVITY_TEST_DSN` unset (332 pass / 13 skip / 0 fail; the 13 skips are
  `visitoractivity`'s Postgres tests); the 28 earlier modules are unchanged, so 1,498 / 29 / 0 in all. Findings:
  (1) three third-party modules enter the workspace at their original pins (`jwt/v5` v5.3.1, `go-redis/v9` v9.18.0,
  `x/crypto` v0.49.0 for bcrypt). (2) Four `auth` files are not gofmt-clean in the original; formatted in their own
  commit (11 insertions / 12 deletions). (3) `jobs/scheduler` still imports `news/feed`, ported as is — the one known
  shared-imports-feature edge. (4) The `replace`-closure rule found in 6a is now written in `platform/CLAUDE.md`.
  (5) The 29 skips at that point, read one by one afterwards: 13 need a DSN (`visitoractivity`), 13 are unconditional
  RED placeholders, 3 are pointers at app-level tests; a live Postgres can enable only the 13.
  (6) A defect in the original `auth` (magic-link onboarding stores the raw refresh token; severity corrected on
  2026-09-26 to a functional bug plus hygiene, not an exposed credential) was found by
  the background review of the port commit and confirmed by reading; recorded in section 7 as a defect found in the
  originals, not fixed in the port, decision left to the user.
- 2026-09-26 — Scope 6c done: two Go modules (`real-estate/property`, `real-estate/transaction`) copied verbatim —
  51 source and `testdata` files, byte-identical — plus 2 authored `go.mod` and 2 `go.sum`; `go.work` 32 → 34 paths, so
  **all 34 Go modules are ported**. Tests per module equal a baseline measured on the original module (292 pass /
  4 skip / 0 fail; the 4 skips are unconditional `t.Skip` calls that point at app-level tests, not Postgres); the 32
  earlier modules are unchanged, so 1,790 / 33 / 0 in all. Findings: (1) the `testdata/` fixtures are copied
  byte-for-byte and read by relative path; the tests pass in the new folders. (2) Five files are not gofmt-clean in the
  original; formatted in their own commit (10 insertions / 10 deletions). (3) `gosimple/slug` and its indirect
  `gosimple/unidecode` enter the workspace at their original pins. (4) `mapper.generated.go` in both packages is
  ported as is, not regenerated; regeneration belongs to the kit slice. (5) `property` requires five in-repo modules
  and replaces seven (`geocoordinate` and `database` arrive transitively).
- 2026-09-26 — Decision on the `identity/auth` defect (section 7, item 1): fix it in the original first. The fix is
  the user's to make; this track does not edit the originals. Sequencing consequence for Scope 7: the oracle is
  captured from the original as it stands, so the fix should land before that capture, then `identity/auth` is
  re-synced in one small scope.
