# bos — turning bestierealestate into a system plus a thin consumer

Last updated: 2026-09-25
tags: bos, bestierealestate, bestays, go, svelte, decomposition, proposal

Status: **plan approved; Scope 1 and Scope 2 (package taxonomy and regroup) approved and done on
2026-09-25** (logs: `docs/plans/2026-09-25-bos-scope-1-workspaces-and-leaf-ports.md`,
`docs/plans/2026-09-25-bos-scope-2-package-taxonomy-and-regroup.md`). Per `CLAUDE.md`, approval is per
scope: this document fixes the *direction* and the *first scopes*; every later scope gets its own
before/after file tree and its own confirmation before anything is written.

Client-specific values (brand, copy, contacts, hosts) stay in the client repo. This document names
projects (`bestierealestate`, `bestays`) because the repo already does, and nothing more.

---

## 1. The assignment — the end state

When the work is finished, the Go + SvelteKit product that runs in production today as
`bestierealestate` runs **identically** (same routes, same API bytes, same rendered pages, same
database schema), but its source is no longer an app:

- The code lives **once**, in `sdlc-kit/platform/`, as **packages** (libraries), **kits** (one
  domain area across Go and Svelte: handlers, repositories, migrations, routes, components, admin
  pages) and two **frameworks** (`bos-go`, `bos-svelte`) that assemble kits into a running app.
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
| **package** | One job, one language, depends only downward. `money`, `repository`, `ui-seo`. Lives in a *group* (section 3.1). | Reusable with no knowledge of any other domain area. |
| **kit** | One *domain area* across both stacks: Go handlers + repositories + register + migrations, Svelte routes + components + admin pages, and the spec keys it reads. `faq`, `identity`, `property-catalog`. | Carries domain vocabulary (property, booking, faq). |
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

### 3.1 Package taxonomy and placement rules

One taxonomy for both stacks: `platform/<lang>/packages/<group>/<package>`.

- **`values`** — pure value types or logic: no I/O, no product vocabulary.
- **`foundation`** — technical plumbing (Go) or UI foundation (Svelte): no product vocabulary.
- **Domain areas** — anything with product vocabulary, in the area it serves: `identity`, `content`,
  `media`, `engagement`, `analytics`, `real-estate`. A domain folder is created when its first
  package lands, never pre-scaffolded.

**Placement test, in order:**
1. Pure value type or logic, no I/O, no product vocabulary → `values`.
2. Technical plumbing or UI foundation, no product vocabulary → `foundation`.
3. Product vocabulary → the domain area it serves.
4. Used by two or more domain areas → it moves *down* (`values` / `foundation`), never sideways.

**Dependency direction:** `values` ← `foundation` ← domain areas. A domain area may import another
only through a declared edge (an architecture test enforces this later). Known violation today:
`scheduler` (foundation) imports `feed` (content) — to be inverted so jobs register themselves.

**Directory placement is independent of import identity.** Groups are directories only; module paths
and npm names stay verbatim until M5 (D7). Kits are finer-grained than groups: a kit draws packages
from several groups.

```
platform/go/packages/
├── values/        money · phonenumber · language · geocoordinate · personname · socialnetwork · seo · address
├── foundation/    database · repository (+postgres) · httputil · notify · scheduler (+schedcli)
├── identity/      user · rbac · auth
├── content/       dictionary · rss · feed · cms · faq
├── media/         image · video
├── engagement/    contact (+vcard) · inquiry · calendar (+ical)
├── analytics/     visitoractivity
└── real-estate/   collection · property · transaction            → 34 modules (30 top-level + 4 nested)

platform/svelte/packages/
├── values/        units · text-template
├── foundation/    animations · core-ui (decomposed in M1)
├── content/       ui-seo
├── media/         canvas-kit · ui-image · canvas-ui · ui-source-picker
├── engagement/    ui-calendar · ui-contact
└── real-estate/   ui-map                                          → 12 packages
```

`units` sits in `values` because `core-ui` and `ui-map` depend on it, even though its Thai land units
are real-estate flavoured: the dependency direction wins. Python (`platform/python`) is untouched — it
is one family (process-kit) belonging to another track.

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
| D6 | Go: **one module per package** under one `go.work` (34 modules, 4 nested as sub-packages). Reason: `sbx-core` is one module that drags aws-sdk/webp/jwt/redis onto every consumer; per-package modules make "enabled kits" true at the dependency level | Proposed |
| D7 | Names stay **verbatim** through M0 (`github.com/shredbx/sbx-core/pkg/<name>` module paths — nested module paths are legal, longest prefix wins — and `@sbx/*` npm names); final names decided at M5 via a mechanical codemod. Directories are grouped from Scope 2 (D13): placement and import identity are independent | Proposed |
| D8 | Oracle runs on deterministic seed + synthetic users only, never on production backups | Proposed |
| D9 | Consumer routes are committed one-line re-export shims (generated from the spec, drift-checked by `conform`) **or** a build-time composed directory; decided by the M1 spike, lean = shims (an override is then just a real file at that path) | Open — spike decides |
| D10 | Kit join manifests and presets are process-os **records** (`records/sbx-sdlc-kit/<capability>/{kit,preset}/`), with schemas modeled per-definition after the shape is proven; namespace chosen capability-first (the 13 SDLC capabilities) at that time | Proposed; nothing modeled yet |
| D11 | Framework reads all env-specific settings from config; neutral env names with a consumer-side mapping so production's existing secrets keep working until cutover; the schema name becomes one config value | Proposed; scheme settled in M2 |
| D12 | "Selling" must be switchable by configuration, not by deleting code: the real-estate kit splits into `property-catalog` and `transactions`, with property offerings (`for_sale`/`for_lease`) a spec key | Proposed; inference from recon, confirmed against code in M4 |
| D13 | Package taxonomy and placement rules (section 3.1): `platform/<lang>/packages/<group>/<package>`, groups `values`, `foundation` + domain areas created lazily; the same taxonomy on both stacks | Accepted 2026-09-25 |
| D14 | "capability" is reserved for the 13 SDLC capabilities; product areas are "domain areas" | Accepted with D13 |

## 6. Final structure

### 6.1 sdlc-kit (the framework side)

```
sdlc-kit/
├── platform/
│   ├── CLAUDE.md                          placement test · dependency direction · group map (auto-loaded here)
│   ├── python/ …                          (unchanged: packages/×7, frameworks/{process-framework, agent-framework})
│   ├── go/
│   │   ├── go.work
│   │   ├── packages/<group>/<package>     34 modules, verbatim ports of sbx-core pkg/*  (section 3.1, table 6.3)
│   │   ├── kits/                          Go half of each domain area: handlers · repositories · register · migrations/
│   │   │   ├── identity/  content/  seo/  media/  dictionary/  faq/  contact/  calendar/
│   │   │   ├── news/  scheduler/  analytics/  documents/
│   │   │   ├── property-catalog/  transactions/       (the real-estate kits, split so selling can be switched off)
│   │   │   └── assistant/                              (thin proxy to the Python service; held in consumer first)
│   │   └── frameworks/bos-go/             Module interface · buildRouter · config · migration composer · middleware chain
│   ├── svelte/
│   │   ├── pnpm-workspace.yaml · package.json · pnpm-lock.yaml
│   │   ├── packages/<group>/<package>     12 packages (section 3.1, table 6.4); core-ui decomposed into ui-* packages (provisional)
│   │   ├── kits/                          Svelte half, same domain names: routes · components · admin pages
│   │   └── frameworks/bos-svelte/         app shells (public / admin / minimal) · module registry · site-config · theme · API client · auth hooks
│   └── (later) bookings kit for bestays, in both halves
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

### 6.3 Go packages — where each of the 34 goes (verbatim, `platform/go/packages/<group>/<package>`)

Layer = dependency layer within the closure (L0 = no in-repo imports). Src/test LOC from source.
✔ = ported. Nested modules sit inside their parent's folder.

| Group | Package (src / test LOC) | Layer | Serves kit | Status |
|---|---|---|---|---|
| values | money (843/911) | L0 | property-catalog, transactions | ✔ |
| values | phonenumber (56/111) | L0 | contact | ✔ |
| values | language (60/28) | L0 | content, contact | ✔ |
| values | geocoordinate (65/85) | L0 | property-catalog | ✔ |
| values | personname (56/187) | L0 | contact | ✔ |
| values | socialnetwork (109/187) | L0 | content, contact | ✔ |
| values | seo (137/118) | L0 | seo | ✔ |
| values | address (409/761) | L1 | property-catalog, contact, content | |
| foundation | notify (272/122) — Telegram + SMTP notifier (`telegram.go`, `smtp.go`) | L0 | kit assigned when read (likely contact/inquiry alerts) | ✔ |
| foundation | database (965/588) | L0 | bos-go (migrations, pool) | |
| foundation | repository (411/557) | L0 | bos-go, all kits | |
| foundation | repository/postgres (1,149/660) — nested | L1 | bos-go | |
| foundation | httputil (405/429) | L1 | bos-go | |
| foundation | scheduler (955/915) — imports `content/feed` today (edge to invert) | L2 | scheduler | |
| foundation | scheduler/schedcli (232/0) — nested | L3 | scheduler | |
| identity | rbac (374/362) | L0 | identity | |
| identity | user (412/112) | L1 | identity | |
| identity | auth (2,631/3,189) | L2 | identity | |
| content | rss (1,937/2,037) | L0 | news | |
| content | dictionary (909/638) | L1 | dictionary | |
| content | feed (343/221) | L1 | news | |
| content | cms (1,561/1,497) — self-declared BR-local; genericize or keep, decided in the content kit | L2 | content | |
| content | faq (580/482) | L2 | faq | |
| media | image (2,265/1,541) — coupled to Postgres and S3 | L0 | media | |
| media | video (855/513) | L0 | media | |
| engagement | contact (1,216/417) | L2 | contact | |
| engagement | contact/vcard (677/499) — nested | L3 | contact | |
| engagement | inquiry (451/279) | L2 | contact | |
| engagement | calendar (797/413) | L2 | calendar | |
| engagement | calendar/ical (893/878) — nested | L3 | calendar | |
| analytics | visitoractivity (1,675/1,945) | L2 | analytics | |
| real-estate | collection (292/246) | L1 | property-catalog | |
| real-estate | property (4,939/3,256) | L2 | property-catalog | |
| real-estate | transaction (1,290/1,073) | L2 | transactions | |

### 6.4 Svelte/TS packages — where each of the 12 goes (`platform/svelte/packages/<group>/<package>`)

| Group | Package (src LOC) | Notes | Status |
|---|---|---|---|
| values | units (402) | pure TS; depended on by `core-ui` and `ui-map` (hence `values`) | ✔ |
| values | text-template (115) | pure TS | ✔ |
| foundation | animations (1,821) | pure TS; no tests (author some); needed by `core-ui` | |
| foundation | core-ui (44,748) | ported as ONE package first (needs `animations` + `units`); decomposed in M1 into `ui-primitives`, `ui-layouts`, `ui-navigation`, `ui-sections`, `ui-viz`, `ui-blocks`, `ui-theme`, `ui-analytics`, `ui-language` (provisional — only BR-imported subpaths) | |
| content | ui-seo (1,285) | imports `core-ui` (2 files); `$app/state` coupling in `SeoHead` only | |
| media | canvas-kit (4,474) | pure TS, LOW drift | ✔ |
| media | ui-image (4,013) | imports `core-ui` (8 files) | |
| media | ui-source-picker (1,137) | zero tests | |
| media | canvas-ui (14,299) | zero tests, 8 deep imports into `core-ui`'s `Modal.svelte`, token contract with no declarations | |
| engagement | ui-calendar (3,851) | declares `core-ui`, imports it 0 times | |
| engagement | ui-contact (1,172) | imports `core-ui` (2 files); hardcodes consumer tokens | |
| real-estate | ui-map (7,995) | declares `core-ui`, imports it 0 times; parcel/plot code | |

`workspace:*` breaks `pnpm install` for a package whose declared workspace dependency is missing, so
the Svelte port order is dependency-driven: `animations` → `core-ui` → the `ui-*` packages.

### 6.5 App-owned code → kits (provisional; each kit's boundary is proven only when it lands)

| Kit | Go half (from `api-chi`) | Svelte half (from `web-svelte`) |
|---|---|---|
| identity | `internal/handler` auth/users/audit/reset, roles → spec | `(minimal)` login/callback, `admin/users`, `admin/audit`, `admin/reset-requests`, `manage/account` |
| content | `internal/siteconfig`, `internal/page` (generic part), `internal/localization`, cms handlers | `manage/pages/**` (appearance header/footer/theme), sections renderer, public about/terms/privacy/cookies/services/guides |
| seo | seo endpoints, sitemap/robots/llms | `manage/seo`, SEO tab pages, `robots.txt`/`sitemap.xml`/`llms.txt` routes |
| media | `internal/watermark`, `canvas`, `media`, `imageimport` | `lib/components/media`, `manage/tools/media-canvas`, `manage/marketing/watermarks` |
| faq | `RegisterPublic/ManageFaqRoutes` | `manage/faq/**`, public `/faq` |
| contact | contact/inquiry handlers, `internal/qr` | `lib/components/contact`, `manage/contacts/**`, `manage/inquiries/**`, public `/contact` |
| calendar · news · scheduler · analytics · documents · dictionary | their handlers/repositories (`internal/newsfeed`, `feedcache`, `document`, `registry`…) | `manage/calendar`, `manage/news`, `admin/schedules`+`admin/backup`, `lib/analytics`+`manage/analytics`, `manage/documents`, `manage/content/dictionaries`+`manage/tags` |
| property-catalog | `internal/propertyfilter`, `autotag`, `areageo`, `agent`, collections, similar/related (extension seam → consumer) | public `/properties/**`, `/search`, `/p/[id]`, `manage/properties/**`, `manage/content/properties/**`, `manage/agents` |
| transactions | `internal/deal`, transaction handlers | `manage/transactions/**`, public `/sell` |
| assistant | `aiassistant`, `assistantchat`, `assistantcapabilities` — **held in the consumer**; integrates later with the main session's agent-framework | `manage/ai-assistant/**` — held in the consumer |
| (framework) | route assembly, middleware order, config, DI | app shells, `NAV_TREE` → derived from kit manifests, API client, auth hooks, site-config loader |

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

## 8. Ladder

| Rung | What | Gate |
|---|---|---|
| **M0 Baseline** (scopes 1–7 below) | Port the 34 Go + 12 Svelte libraries verbatim with their own tests green; import bestierealestate verbatim so it runs from the new layout; capture the oracle | tests green, oracle captured |
| M1 `bos-svelte` | Shell + module registry; package conventions settled; brand-token codemod; one API client; spec v0; decompose core-ui | pixel + DOM diff = 0 |
| M2 `bos-go` | `Module` interface, `buildRouter`, config, single roles list, migration composer; proven on one vertical slice: **FAQ** (small, generic, public + admin, already `Register*Routes`-shaped) | route-table diff = 0 |
| M3 kits | identity → content → seo → media → faq/contact → calendar → news → scheduler/analytics/documents | oracle slice green per kit |
| M4 real-estate kits | `property-catalog`, `transactions`; consumer keeps only product decisions via extension seams | full oracle green |
| M4b second consumer | `bestays` on the same kits + `bookings` − selling; extract the preset; the falsifiability test of every kit boundary | bestays needs config + `bookings` only |
| M5 Thin-out + regroup | consumer footprint audit; final names; move to `clients/bestie/bestierealestate` | full oracle green |
| M6 Deploy cutover | its own plan: bundle spec and/or the mirror/vendor pipeline redone for a multi-repo layout; parallel run; nothing touches production without separate approval | separate approval |

M0 scopes (provisional order; each gets its own before/after tree and approval). The Svelte order is
dependency-driven: `ui-image`, `ui-contact` and `ui-seo` import `core-ui`, and `ui-map`/`ui-calendar`
declare it, so `core-ui` — which needs `animations` and `units` — comes first.
1. **Scope 1 (done):** workspaces, 8 Go leaf packages, 3 pure-TS packages.
2. **Scope 2 (done):** package taxonomy and regroup (no new code).
3. **Scope 3:** the remaining Go L0/L1 packages (13 modules): address, database, repository (+postgres),
   httputil, user, rbac, dictionary, rss, feed, video, image (the unused `watermark/` sub-package stays
   behind), collection.
4. **Scope 4:** Svelte foundation: `animations`, then `core-ui` (one package, verbatim).
5. **Scope 5:** Svelte `ui-seo`, `ui-image`, `ui-contact`, `ui-map`, `ui-calendar`, then
   `ui-source-picker` and `canvas-ui`.
6. **Scope 6:** Go L2/L3 domain packages: auth, cms, faq, contact (+vcard), inquiry, calendar (+ical),
   scheduler (+schedcli), visitoractivity, property, transaction.
7. **Scope 7:** baseline import of the bestierealestate apps + oracle capture.

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
- 2026-09-25 — Finding while ordering the Svelte ports: `ui-image` (8 files), `ui-contact` (2) and
  `ui-seo` (2) import `core-ui`; `ui-map`/`ui-calendar` declare it unused; `core-ui` needs `animations` +
  `units`. Svelte order is dependency-driven (section 8); the earlier "adapters first, core-ui last" order
  was wrong.
