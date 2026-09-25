# bestierealestate decomposition — recon findings

Last updated: 2026-09-25
tags: bestierealestate, bos, go, svelte, recon, research

> Scope note: four read-only recon passes run on 2026-09-25 to answer one question — *what would it take
> to turn the production Go + SvelteKit app into a `bos` system plus a thin consumer?* Numbers come from
> source (`wc`, `grep`, reading files). LOC splits marked "≈" are estimates within about 10%. Items marked
> *unverified* were not checked. Nothing was run against production and nothing was modified. The design
> that follows from this is `docs/proposals/bos-system-design.md`; the earlier, broader inventory this
> builds on is `docs/research/shredbx-bestierealestate-and-capabilities.md` and, in `sbx.framework`,
> `docs/research/2026-08-06-br-package-inventory.md`.

Source app: `shredbx/clients/bestie/projects/bestierealestate` (5 apps: `api-chi`, `web-svelte`,
`assistant`, `backup`, `watermark-worker`). Shared libraries: `shredbx/projects/sbx/packages`.

---

## 1. Svelte app (`apps/web-svelte`)

**Size.** 353 `.svelte` (93.8k LOC) + 516 `.ts` (51.9k LOC, of which 180 files / 22.7k LOC are tests).
Non-test total 123.7k across 689 files.

| Dir | Files | LOC |
|---|---|---|
| `lib/components` (admin 102/30.2k · media 32/9.6k · public 44/6.7k · contact 24/5.1k · inputs 29/3.3k · sections 16/2.9k · top-level 35/6.9k) | 290 | 66,595 |
| `lib/server` | 63 | 6,904 |
| `lib/canvas` · `lib/site` · `lib` top-level | 27 · 22 · 128 | 3,178 · 3,363 · 16,546 |
| `routes/(admin)` | 229 | 34,854 |
| `routes/(public)` | 61 | 10,182 |
| `routes/(minimal)` · root SEO routes | 5 · 7 | 672 · 537 |

`clients/` (empty directories) and `externals/` (11 pnpm shims) at the app root are stray tool output, not
vendored code. Whether they are git-tracked is *unverified*.

**Packages consumed.** 9 declared `workspace:*` deps (`core-ui`, `canvas-kit`, `canvas-ui`, `ui-calendar`,
`ui-contact`, `ui-image`, `ui-map`, `ui-seo`, `units`), 12 in the transitive closure. `core-ui` is imported by
134 files through 9 subpaths (`components/primitives` 84 files, `components/layouts` 33). There are no aliases in
`svelte.config.js` or `vite.config.ts`: `node_modules/@sbx/*` are symlinks to the monorepo's packages, which
ship raw source (exports point at `src/lib/*.ts`, no build). `core-ui`'s `PageShell` and `Modal` are never
imported; the app carries its own `AdminModal` (39 files) and a local input kit (29 files, 3.3k LOC).

**Config, theme, SEO, i18n.**
- Brand tokens are static in `app.css` (189 custom properties, 88 in the app's own namespace) and bridged to
  `core-ui` contract tokens. 302 files use the app namespace (11,039 uses); 557 hex literals remain in
  component styles. Fonts are linked in `app.html`.
- Site config is a DB singleton served by `GET /api/site/config` (wordmark, logo, header nav, footer sections,
  content sources, animation, hero defaults) and edited at runtime in the admin "Appearance" screens.
  The root, public and admin layouts each re-fetch it, so a request fetches it two or three times.
- Chrome is app-owned: the public layout is 1,079 lines, the admin layout 967, driven by a hardcoded
  `NAV_TREE` (23.7 KB).
- SEO uses `SeoHead` in 16 files; the site name is hardcoded 12×, and 79 of 94 `<title>` tags hardcode the
  brand. Brand strings appear in 178 files. i18n is effectively absent: a `lib/i18n` with two 20-line files
  has no importers; localization is per-field data (`LocalizedField`/`LanguageTabs`) and `<html lang>` is static.

**Routes and repetition.** `(admin)` has 99 `+page.server.ts` (78 are ≤ 60 LOC), 88 `+page.svelte`, and 10
layouts (2,995 LOC). `TabbedPageShell` appears in 33 route files and `buildEntityTabs` (Listing, Design,
Settings, Analytics, SEO) drives 5 hubs. The repetition is parametric: guides vs services layouts are 181 vs 180
lines with 43 differing diff-lines; their SEO tab pages differ by 14 of 31 lines (label and slug only); header
vs footer appearance SEO/analytics pages differ by 2 lines. Overall there are 7 per-entity SEO pages, 8
analytics tab pages, 3 design pages, 5 `new` routes. Not duplicates: guides/services public detail pages, and
the bespoke `pages/appearance/{theme 1,738 · footer 838 · header 541}` editors.

**API client.** No shared contract (no OpenAPI, no codegen). `lib/api.ts` (≈66 KB) plus `lib/*-api.ts` and
`lib/api/content.ts` (3,835 LOC in total) hand-mirror the Go DTOs in 155 exported types. 52 files each declare
`API_BASE_URL`; 584 `/api/` literals (365 under `/api/manage`). `hooks.server.ts` passes `/api/*` through,
replays once after a silent refresh on 401, and calls `/api/auth/me` on each navigation. Cookies are
HttpOnly/SameSite=Strict with TTLs hardcoded to mirror Go; 78 files set `X-Requested-With` by hand;
`requireAuth`/`requireRole` have 83 call sites.

**Domain vs generic** (non-test LOC, name-and-path heuristic): ≈35.7k (29%) strictly property-specific
(public catalogue, `manage/properties` + `content/properties`, transactions, property components, listing
modules); inquiries (2.7k) and watermark/marketing (1k) are borderline; the rest is generic — admin non-property
routes 19.4k, admin shell components 19.4k, non-property media 8.1k, public non-property routes 5.0k, contacts
4.6k, sections 2.5k. The home page (777 lines) mixes both.

**Tests.** Playwright: 66 specs + fixtures (7.4k LOC), ≈245 `test(` calls of which 49 are skipped/fixme; needs
the live Go API, Postgres and seeded users; no screenshot assertions. Vitest: 180 files, ≈1,583 `it(`.
Vitest is portable; Playwright covers admin flows and public status codes but has no visual or DOM baseline.

**Hazards for "identical when run".** Generated files (`analytics.yml` → `config.generated.ts`, brand CSS,
Dockerfile/Makefile/entrypoint from a bootstrap tool); env (`PUBLIC_API_URL`, `PUBLIC_MEDIA_BASE`,
`PUBLIC_FIREBASE_*`, maps key, `APP_VERSION`); brand/locale hardcoded across the source; `Asia/Bangkok` in 5
files; THB/satang handling in 61; `static/` assets; `core-ui` consumed as live source via symlinks, so a
`core-ui` change alters the app with no app diff; cookie TTLs and default content sources duplicated from Go.
Design-time specs exist in shredbx (`project.yml` 889 lines, `brand.yml`, `application.yml`, pages) but
nothing generates the app from them.

## 2. Go API (`apps/api-chi`)

**Verdict: not thin in code terms.** 37.5k non-test LOC of its own (+37.1k test); roughly 57% generic
HTTP/Postgres adapter logic, 34% real-estate domain, 9% wiring. The shared-package domain logic itself is thin;
the app-side adapter layer is large.

- **Entry points.** `main.go` 3,860 LOC (203 KB): lines 1–1594 hold `Config`, `PropertyHandler`/`ImageHandler`,
  response DTOs; `main()` runs 1595–3860 (≈2,266 lines). Root files `property_related.go` (339),
  `property_similar.go` (200), `smart_collection.go` (50). `cmd/` has 6 binaries (638 LOC): `bootstrap-admin`,
  `migrate`, `refresh-feeds`, `scheduler-run`, `sqlexec`, and a 14-line `scheduler` shim over `schedcli.Main`.
- **Wiring.** Manual DI: handlers are nilable pointers assigned in `if db != nil`; cross-package hooks via
  setters/closures; no container. Middleware order: RequestID, Logger, `JSONRecoverer`, `SecurityHeaders`, CORS,
  `AuthExtract` (only if the auth service exists), then per `/api` group `CSRFCheck`, per route `RequireAuth`,
  `RequirePermission` (115 sites), `MutationRateLimit` (112), `BodySizeLimit`. `http.Server` has no timeouts.
- **`internal/` by LOC (non-test/test) and class.** `handler` 17.8k/18.1k (≈11.3k generic, ≈6.5k real-estate);
  `repository` 5.9k/8.1k (≈4.7k generic, ≈1.2k real-estate); `watermark` 916 generic; `deal` 712 RE;
  `canvas` 681 generic; `siteconfig` 672 generic; `propertyfilter` 645 RE; `page` 593 mixed (355 generic dictionary
  view-models); `media` 575 generic; `document` 476 generic; `localization` 474 generic (coupled to
  `property.Translations`); `aiassistant` 470 generic; `autotag` 367 RE; `registry` 319 thin; `agent` 285 RE;
  `imageimport` 258 generic; `assistantchat` 240; `areageo` 229 RE; `assistantcapabilities` 225 (hand-mirrored
  descriptors); `export` 196; small: `feedcache`, `newsfeed`, `config`, `qr`.
- **Routes.** No route table: about 258 unique routes (±5; 176 under `/api/manage`) registered imperatively in
  `main()`, with 13 duplicated `if rateLimiter != nil` branches and 27 hand-written 503 fallback stubs. By
  module: property 58, media/canvas/watermark 33, contact/inquiry 22, auth/users/audit 21, CMS/site/entries 20,
  news 18, documents 12, calendar 11, FAQ 11, dictionaries/tags 11, transactions 10, agents 8, AI assistant 8,
  insights/track 6, scheduler 4. No OpenAPI or generated client; the declared route list in shredbx's
  `application.yml` is stale.
- **Persistence.** pgx v5 pool; simple CRUD through the shared generic `PostgresStore[T]` + per-entity mapper;
  everything else hand-written SQL (35 non-test files touch the pool, 6 of them handlers). The schema name is
  hardcoded 60× (54 in `main.go`).
- **Migrations.** 310 files = 156 logical migrations (154 up/down pairs + 2 single files), applied by the shared
  `pkg/database.Migrate` (advisory lock + `schema_migrations`). Roughly 75–80 real-estate (property 37,
  amenity 12, market areas, agents, tags, collections, units) and 70–75 shared-package-shaped (auth, images/
  watermarks/canvas/videos ≈22, cms/faq ≈17, site_config ≈17, contacts 10, news 5, transactions 4, …). The
  shared packages ship no SQL, so all their DDL lives here.
- **Config.** Hand-rolled `Config` struct: `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `BASE_URL`, `PORT`,
  `ENVIRONMENT`, `ASSISTANT_CHAT_URL`, `FEED_SOURCES_PATH` plus 13 project-prefixed variables (R2 ×5, CORS,
  canvas image origins, Google Fonts/Maps keys, YouTube, Telegram ×2, chat token salt, search threshold).
  Roles are declared twice (constants and a hardcoded `ValidRoles`). There is no runtime "enabled modules"
  notion; effective toggles are resource presence (DB nil → 503 stubs, R2 unset → upload routes vanish, …).
- **Tests.** 152 files, 37.1k LOC, 925 `Test*` functions; ≈115 unit files with `httptest` and in-memory stores,
  ≈37 `//go:build integration` files (≈34 with testcontainers postgres, needing Docker; they replay the
  migrations and are path-coupled to the repo layout). The handler tests build their own `chi.NewRouter()`, so
  the real route table, middleware order and per-route RBAC are untested; `response_*` tests are struct-level
  JSON assertions, not golden files.
- **Byte-identity hazards.** `PropertyResponse`/`PropertyDetail` depend on `encoding/json` dominance rules
  (same-name shallow `omitempty` fields shadow promoted fields; an embedded pointer is flattened; field order is
  declaration order); `json:"-"` data minimization plus `stripPrivate*` zeroing; localization per request
  (`?lang`, then `Accept-Language`, then default, reading a dictionary from the DB); ordering rules with SQL/Go
  twins that must change together; 20 `uuid.New*` and 32 `time.Now()` sites to normalize; duplicated
  `writeJSON`/`writeError` with different signatures and mixed upper/lower-case error codes. Do not "fix"
  during a split: CORS methods omit PATCH; mixed rate-limit key functions; route registration order; per-handler
  `Cache-Control`.

## 3. Shared libraries

**Go — 34 packages of `sbx-core`** (the 33 the app imports plus `personname`, pulled in by `contact`): 164
source files, 30,221 LOC; 128 test files, 25,257 LOC; ≈34% of `pkg/`. None imports `sbx-core/internal/*`.
Layers (dependencies within the closure): **L0** (none) database, geocoordinate, image, language, money,
notify, personname, phonenumber, rbac, repository, rss, seo, socialnetwork, video; **L1** address, collection,
dictionary, feed, httputil, repository/postgres, user; **L2** auth, calendar, cms, contact, faq, inquiry,
property, scheduler, transaction, visitoractivity; **L3** calendar/ical, contact/vcard, scheduler/schedcli.
Hubs: `repository` is imported by 10 closure packages, `repository/postgres` by 9, `seo` and `socialnetwork`
by 4. Third-party: pgx in 12 packages; aws-sdk + webp only in `image`; go-redis in `auth` and
`visitoractivity`; jwt + bcrypt only in `auth`; validator in `httputil`; slug in `property`; yaml in 5;
`x/text` in `language`. 17 packages are stdlib-only. sbx-core as a whole is one module of 846 files that drags
71 unused `pkg/` packages and 75 `internal/` packages onto every consumer.

**Svelte/TS — 12 packages**, ≈395 files, 85,312 src LOC + 11,103 test LOC: `core-ui` 44.7k (nine
sub-systems under one name), `canvas-ui` 14.3k (zero tests), `ui-map` 8.0k, `canvas-kit` 4.5k (best coverage),
`ui-image` 4.0k, `ui-calendar` 3.9k, `animations` 1.8k (no tests), `ui-seo` 1.3k, `ui-contact` 1.2k,
`ui-source-picker` 1.1k (no tests), `units` 0.4k, `text-template` 0.1k. Not in BR's closure: `ui-code`,
`ui-diagram`, `ui-video`, `i18n-svelte`, and the games packages.
Drift of note: `ui-contact` and `ui-source-picker` are styled entirely from the consumer's own token namespace
(21 and 15 variables); `ui-image` leaks the same namespace (12 uses); `ui-map` and `ui-calendar` declare an unused
`core-ui` dependency; `canvas-ui` deep-imports `core-ui`'s `Modal.svelte` 8× through a wildcard export and
consumes 43 tokens it never declares; 13 `core-ui` files import SvelteKit virtuals despite being a plain library.
Sixteen conventions diverge across the twelve (type naming, props typing, adapter vocabulary, export shape,
token namespace, config mechanism, callback casing, file-stem case, import extensions, test presence, …), and
they cluster by era of writing.

**Domain coupling of the Go closure.** Real-estate by design: `property` (4.9k LOC), `transaction`,
`collection`, `inquiry`, `visitoractivity`. Self-declared consumer-local: `cms`. Generic with small leaks:
`image`, `rss`, `dictionary`, `faq`, `address`, `contact`, `calendar`, `httputil`. Clean: `auth`, `user`, `rbac`,
`database`, `repository`(+postgres), `seo`, `notify`, `scheduler`, `language`, `money`, `phonenumber`,
`personname`, `geocoordinate`, `socialnetwork`, `video`, `feed`, `ical`, `vcard`.

**Manifest layer in shredbx** (`.sbx/workspace/packages`: 112 `package.yml`, 222 files). No `provides:`/`requires:`
keys; hand-written `dependencies:` that have drifted (a planned package that lists a non-existent dependency;
entities naming libraries absent from `go.mod`). Real wiring lives in `fabric/modules/*.yml`. Of 30 top-level
closure packages, 16 have a `package.yml`, 4 only an `entity.yml`, 10 neither. The app's `project.yml` has a
`configuration:` block (modules enabled/deferred, templates, bindings, dictionaries) that its own comments say
is ignored by the Go parsers; it disagrees with the code (calendar/contacts/documents "deferred" but mounted).

**How the current tooling wires the app.** Go: a workspace file plus a relative `replace` to the shared module.
Svelte: pnpm workspace symlinks. Deploy is mirror-only: stage → vendor (copy the shared module in, rewrite the
`replace`, `go mod vendor`; `pnpm deploy --filter` for Svelte) → scrub → push to a mirror repo that the host
builds. Path-coupling points a port must change: the external registry entry, the root pnpm workspace file, the
root Go workspace file, the app's `go.mod` `replace`, the `path:` fields in manifests, and `composes:` in the
project spec.

## 4. The earlier BOS attempt (`sbx.framework/projects/bos`)

An earlier attempt at exactly this idea, generated from records: 105 record files (14 sections, 19 content
rows, 6 pages, 6 config concerns, 1 brand, 38 FAQ items, 7 FAQ steps, …), a generated `run` script, and a
whitelabel Svelte app (127 files, ≈9.5k LOC) whose packages ship rows and templates and "no runnable code at
all".

**What it proved.** A data model that works: theme ← brand ← config-concern overlay by `subject`, a derived
resolved record ("lockfile-as-row"), fail-open defaults, a token taxonomy, pages/sections/content rows feeding
one generic route, and a two-flavor falsifiability test (exact look values asserted, no shared identity tokens).
This is the easy slice: ≈6 public pages, generic admin, FAQ/guide content.

**What it never touched.** The Go API (modeled once as a "lane", then retired on 2026-09-17), the property
domain (85 tables, filters, privacy projections, media), and BR's bespoke surfaces.

**What hurt.** Flavor written into one shared app tree, so a consumer run left its flavor in the working tree
and the check stayed green (the "rest state" bug recurred four times); placement lists that silently kept the
previous flavor's files; components stored as opaque template text (90% pure copies, three bookkeeping
artifacts per source file, no package-level type checking); claims without readers (14 of 16 component rows
declared props that disagreed with the code); and awk in a generated shell script. No automated visual or
behavioral equivalence to the real app existed — evidence was hand-taken screenshots.

**Reuse / avoid.** Reuse the data model and the second-flavor test. Avoid mutating a shared app tree per
flavor (each consumer owns or generates its own tree), avoid components-as-text (use real packages), avoid
unread claims. Use the real app's own tests plus route and screenshot diffs as the equivalence oracle.

## 5. Earlier package inventory (2026-08-06) — extracts

Priced the port of the 45 BR packages (12 Svelte/TS, 33 Go) at about 195 half-day slices (≈98 focused days)
under a template-row method, excluding the app itself (≈44 more mechanical slices). Grouped the ports into waves
by structural sameness: Wave 1 pure TS (`units`, `text-template`, `canvas-kit`); Wave 2 adapter-bearing Svelte
(`ui-map`, `ui-calendar`, `ui-image`, `ui-contact`); Wave 3 small flat packages (`ui-seo`,
`ui-source-picker`, `animations`); Wave 4 `canvas-ui` then `core-ui`; Wave G the Go side in dependency order.
Its gap list (no type vocabulary beyond 7 primitives, a template engine that cannot iterate, no adapter kind,
no test-fixture bridge for foreign suites) belongs to the old engine; the ordering and the structural findings
carry over.

## 6. Unknowns

- Whether `clients/` and `externals/` under `web-svelte` are git-tracked.
- Test runtimes (Go integration suite, Playwright) — not measured.
- Size and shape of the Python assistant service the `assistant` app deploys (lives outside the app tree).
- Purpose of `notify` was unverified at recon time; it is a Telegram + SMTP notifier (`telegram.go`,
  `smtp.go`), found when it was ported.
- Where the equivalence oracle's "before" should run (original in place vs a copy) — decided when it is built.
