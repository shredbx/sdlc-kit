# bos — roadmap: from an empty app to the first consumer, milestone by milestone

Last updated: 2026-09-26
tags: bos, roadmap, milestones, parity, baseline, bundle, docker, makefile, proposal

Status: **proposal, not yet approved.** Parent: `docs/proposals/bos-system-design.md` (decisions D5, D19 to D26). Companion:
`docs/proposals/bos-constructor.md` (the model and the file structures). This document replaces the earlier sprint draft, which began by
moving slices out of the old app; the direction is now the reverse: **build the new app from nothing and use the old app as the
baseline**. Every scope inside a milestone still gets its own before/after tree and its own approval.

The first consumer's own evidence (its route map, page inventory, wiring facts, brand values, deployment) is in its repo,
`docs/bos-consumer-plan.md`. This document is generic.

---

## 1. The goal, the baseline, and how "similar" is measured

The goal is the first consumer **reimplemented as a bos consumer**. The old app already works, so it is the **baseline**: each milestone
ports one part of it and must produce an output **comparable to that part's output in the old app**. Nothing is built that the old app
does not do, and nothing is imported that it does not use. **Code that already runs is moved and configured, not rewritten** (section 2, D25).

### 1.1 Parity gates

Two different implementations cannot have identical markup, so "similar" is defined in five measurable levels. Each milestone names the
levels it must pass.

| Level | Compares | Passes when |
|---|---|---|
| **P-HTTP** | status, header names and normalized values, the shape of a JSON body | equal for the same request; values that come from configuration are compared after mapping |
| **P-TOK** | design token values and the computed style of named elements | every color, font family, radius, shadow and size equal to the old brand's |
| **P-TXT** | visible text, the heading outline, links (target and text), images (file name and alt), meta tags, structured data | equal after normalizing whitespace and generated ids |
| **P-PIX** | screenshots at three viewports | the changed-pixel share is within the page's recorded budget (starts at 2%, tightened as the library matures) |
| **P-API** | JSON bytes of an API response | equal after scrubbing ids and timestamps (dynamic milestones) |

### 1.2 Capturing the baseline, incrementally

"Capture what you are about to port." The old app is run **in place, read-only**, from the original project (never edited, never
deployed, no production access), on loopback only. A small harness in the consumer's `tests/oracle/` starts it, makes the requests,
normalizes the results and stores **golden files** for the pages or endpoints the next milestone ports. A milestone's parity gate compares
its output with those golden files. The old app is not imported into the consumer; the parts of it that are framework code are extracted into
the frameworks (section 2, D25).

## 2. How we work

- **A milestone is one to four approved scopes** and ends with a demo of the new app and a green parity gate against the baseline.
- **Definition of done**, for every milestone:
  1. it runs and builds from a clean checkout: `make build`, `make check`, and `make bundle-up` where it applies (`make test` once tests exist, D26);
  2. its parity gate is green against the golden files;
  3. `bos check` is clean and the offline gates pass (build and vet with `GOPROXY=off`); no test code is written until the core is validated (D26);
  4. new frameworks and components have unit records and rendered READMEs (the unit model), written as they are built;
  5. committed locally; nothing pushed; nothing touches the original project or production.
- **Order is decided by dependency.** Static before dynamic. The model (pages, blocks, sources) is proven with files before anything needs a
  database.
- **Reuse before writing (D25).** In this order: (1) a ported package is imported as it is; (2) code that runs in the old app is moved into
  the framework and given configuration where it had constants, never rewritten from a description of it; (3) new code only where nothing runs
  today (pages, presets, tokens, assets, the registry, the CLI). The golden files are the evidence for (2). A dependency such as a cache or a
  database is a service record and a bundle through the existing infrastructure process, not code.
- **Names are in this repository (D24).** Module and package paths are `github.com/shredbx/sdlc-kit/...` and `@sbx/*`; only clients have their
  own repositories; a consumer links the frameworks by local directory (`go.work` and `replace`, the pnpm workspace) and nothing is fetched.
- **Tests come after the core (D26): no test code is written while the core is being built; a milestone's gates are the offline build and vet, the golden-file comparison and `bos check`; tests are added, as scopes of their own, once the core is built and validated.**
- **The unit-model track continues** for the ported libraries alongside, at lower priority; the ported libraries enter the new app only
  when a milestone needs one.

## 3. The milestones

Scope counts are ranges. The static phase (B0 to M4) is about 21 scopes; the whole plan to the real-estate kit is about 46, close to the
design's earlier estimate of about 39 ±30%.

### B0 — The baseline harness (about 2 scopes, alongside M0)

| Part | What | Gate |
|---|---|---|
| **B0a** | run the old API read-only in development mode with no database and no cache (it starts, and reports the database as disconnected); capture the P-HTTP golden files M0 needs: the health check, `HEAD /health`, the root info, a CORS preflight, the security headers, a `POST` without the CSRF header, an unknown route | golden files reproducible; the original project untouched; only loopback used |
| **B0b** | run the old web and API with a local Postgres built from the original's migrations and seeds; the capture tool (a headless browser on loopback; animations off) and the normalizers for P-TXT and P-PIX. **Fonts:** the old app loads its fonts from a font host, so the tool answers those requests from local font files (an intercept on the browser, nothing leaves the machine); the files are acquired once, as an explicit and approved setup step. It also captures every custom property on `:root` with its computed value, and the computed style of each element of the pages it captures | one page captured twice gives identical golden files |

### M0 — The empty app: both frameworks attached, linked, built and run as a bundle (about 3 scopes, plus the path rename if approved)

The very first thing that exists. It has one page that says the app works and shows the API's status, which proves the link.

**Deliverables**

1. **`bos-go`** (`platform/go/frameworks/bos-go`, module `github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go`), **extracted from the
   running API, not rewritten** (D25): the configuration loader, the **fixed middleware chain in the old app's order** (request id, logger,
   JSON recoverer, security headers, CORS from configuration, CSRF on the `/api` group), `GET` and `HEAD /health` with the old response keys,
   the root info route and graceful shutdown are the old code moved into the framework, with what was hard-coded (names, ports, origins,
   environment, timeouts) read from configuration and the product-prefixed environment names left to the consumer. Security headers, CSRF and
   authentication are the **ported `identity/auth`, imported not copied**, and the recoverer's error body is the ported `httputil`. The only
   new code is the `bos.yaml` reader and `cmd/bos` (`version`, `env`, `dev`, `build`, `check`). The old code has no tests of its own for these
   pieces (measured), so the golden files are the evidence, and no test is written now (D26).
2. **`bos-svelte`** (`platform/svelte/frameworks/bos-svelte`): the package, the server hooks (the same-origin `/api` pass-through with the
   forwarded client address, security headers), a root layout, and the catch-all page route rendering the placeholder home, whose server
   `load` asks the API for its health and shows "API: healthy". The workspace file gains `frameworks/*`.
3. **The app skeleton** (in the consumer repo), by hand this time and by `bos new` from M4: `bos.yaml`, `apps/api/main.go` (about 15 lines),
   `apps/web`, `site/site.yaml`, and the `Makefile`.
4. **Attachment, by local directory (D24).** Go: the consumer's `go.work` `use`s the framework module, and its `go.mod` `replace`s the framework
   **and its whole in-repo closure** (the ported `identity/auth`, `httputil` and what they require) to relative paths; the paths depend on the
   mount depth (four levels below the sdlc-kit root). Svelte: the consumer's `pnpm-workspace.yaml` includes `@sbx/bos-svelte`. Nothing is
   fetched; the gates run offline.
5. **The link, from one file.** `bos.yaml` holds the API port and allowed origins and the web port and the API address; `bos env` writes
   both halves' environment from it, so the two cannot disagree.

**Scopes.** M0.0 the module-path rename, only if approved (the ported modules move from `github.com/shredbx/sbx-core/pkg/*` to their sdlc-kit
paths, mechanically, before `bos-go` records them). M0.1 `bos-go` (the extraction, unit records and README; no tests, D26). M0.2 the
app skeleton in the consumer and the parity run against the golden files. M0.3 `bos-svelte`, the link, the bundle, the action and the Makefile.

**The bundle, the action and the Makefile** (section 4) are part of M0.

**Baseline and gate.** P-HTTP against the B0a golden files: `/health` (same five keys; the database reads "disconnected" when none is
configured, as the old app reports with none), `HEAD /health`, the root info keys, the CORS preflight, the security headers (HSTS only in
production), the CSRF refusal on a `POST` to `/api/...` without `X-Requested-With`, the unknown-route response, **including the quirks the golden files record** (a CSRF refusal is JSON sent as `text/plain`; a preflight for PATCH or from another origin gets 200 and no CORS headers; a not-found is plain text; there is **no** request-id response header). In addition: a clean
checkout builds; `make bundle-up` passes `verify-link`; changing a port in `bos.yaml` moves both halves.

**Demo.** `make bundle-up`, open the web page: "bos: it works. API: healthy."

### M1 — Branding: colors and logo, assets and static delivery, fonts (about 4 scopes)

Branding follows the kind-folder convention (constructor document, section 4.6): **one folder, `site/brand/`**, with the config, the token
sources, presets, assets and plugins. Delivering the files (favicons, logos, fonts, images) is its own design, section 4.7 of the same
document: sources in `site/`, everything delivered generated into `.bos/build/static/`, nothing copied by hand. M1 creates only what it needs.

#### M1a — colors and the logo (about 2 scopes)

| Where | What |
|---|---|
| **Framework** `bos-svelte/src/lib/theme/` | `pipeline/` (read the YAML sources, normalize to the W3C structure, resolve aliases and modes, build) · `transforms/` and `formats/` (built-ins) · `validators/` (schema, alias cycles, undefined or unused roles, contrast) · `presets/` (a neutral `theme/default`) · `defaults/` (fail-open tokens) · `schema/` · `index.ts` (`definePlugin`, the loader) |
| **Framework** | the `Logo` atom and the logo resolvers (one for the URL, one for the absolute URL); the development-only brand page (`/_bos/brand`: swatches, logo variants); `bin/bos-web` gains `tokens build` and the brand part of `check` |
| **App** `site/brand/` | `brand.yaml` · the imported tokens (`tokens/primitive/`, `tokens/semantic/`, and `tokens/component/` for the deferred contracts) · `tokens/_import/mapping.yaml` · `assets/logo/…` |
| **App**, generated | `.bos/build/tokens/{tokens.css, tokens.ts, tokens.json}` (git-ignored, never committed) |

Steps: (1) **the engine spike**: can Style Dictionary reproduce the baseline brand's computed output from W3C-structured YAML sources? Adopt it if
so; otherwise a small pipeline of our own with the same file formats and plugin interface. (2) The pipeline and its built-ins. (3) **Import all the old tokens** (the consumer's plan, section 17.3b): a one-shot importer in the consumer's repo reads the old stylesheet's custom
properties and its brand file, resolves them, classifies them into the tiers, and writes `site/brand/tokens/**` and a mapping file with a row per old
token; gates T1 to T3 (the mapping is complete, every old token's computed value equals its mapped token's, and the tokens the first pages use are covered). (4) The `Logo` atom and the header. (5) The brand page and the check.

#### M1b — assets and static delivery: favicons, the head, the cache policy (about 1 scope)

| Where | What |
|---|---|
| **Framework** `bos-svelte/src/lib/assets/` | `collect/` · `derive/` (favicon set, apple-touch icon, manifest icons, social card, tinted logos) · `headers/` (the stable and the hashed cache policies) · `head/` (the link and meta generator) · `map/` (the asset-map types and checks); the engine spike for image processing |
| **App** `site/brand/assets/` | `favicon/` only if the app supplies its own; otherwise everything is derived from `logo.mark` |
| **Generated** | `.bos/build/static/{favicon.ico, icon.png, icon.svg, apple-touch-icon.png, manifest.webmanifest, og-default.png, icons/}` and `.bos/build/assets.json` |

#### M1c — fonts and typography (about 1 scope)

`site/brand/tokens/primitive/typography.tokens.yaml` and `semantic/typography.tokens.yaml`; the font source (local files or a font-package id in a
typography preset) with **its license file**; the `@font-face` generation, the preload, the fallback stacks; `assets/fonts/` created with the
first family. The `bos-web check` license rule.

**Baseline** (B0b): the old brand file's values, **every custom property on `:root` with its computed value**, the **computed styles of the running old app** (the two can differ), the logo and wordmark as
rendered in the header, the header region, the old head's icon links, the old icon files (formats, sizes, alpha), and the fonts the old page
actually loads (families and weights).

**Gate.**
- **P-TOK**: every color, font family, radius and shadow the milestone covers equals the old app's computed value.
- **Token import (T1 to T3, consumer plan section 17.3b)**: every old token is mapped; every old token's computed value equals its mapped token's; the tokens the first pages use are covered.
- The declared contrast pairs pass at the old brand file's level; **P-PIX** on the header region within budget.
- **Assets** (from the requirements in section 4.7): `/favicon.ico` at the root returns an icon with 16 and 32 pixel images; the apple-touch icon is
  180 × 180 and has no alpha; the manifest icons are 192 and 512; the head's icon links equal the old head's set (`rel`, `sizes`, `type`); the
  derived apple-touch icon matches the old file within a small pixel budget; **no static file shadows a route**; hashed files carry the immutable
  cache header and stable files the short one.
- **Fonts**: the same families and weights are loaded as in the old page, from the same origin, with **zero requests to any non-loopback host**
  (a test on the new app; the old app's Google Fonts requests are what the new app removes).
- Changing a color, a token or the logo changes the site with **no component edit**; **deleting `.bos/` and rebuilding reproduces identical output**,
  and nothing generated is committed (both are tests); an **empty `site/brand/`** still builds a working site from the framework defaults.

**Demo.** Edit two colors, swap the logo and change a font; the empty app re-skins, the browser tab, the home-screen icon and the social card all
follow, and the brand page shows the result.

### M2 — The atomic library v0, layouts, and the first pages (about 4 scopes)

- **Atoms** (`Text`, `Heading`, `Button`, `Link`, `Image` with the responsive image pipeline of section 4.7, `Icon` from a bundled set, `Logo`, `Badge`), **molecules** (`NavItem`, `Breadcrumb`, `Card`),
  **organisms** (`Header` and `Footer` driven by `site.yaml` and `nav.yaml`, `Hero`, `Prose`), **layouts** (`default`, `legal`, `minimal`).
- The **renderer registry** with `meta.ts`, the **compatibility check**, the **page loader and catch-all route**, `_layout.yaml`, the Markdown
  pipeline with the old app's sanitizing rules, `bos check` for pages, and the first generators (`bos g page`, `bos g renderer`).
- **Pages.** `terms` and `privacy`, with the old app's Markdown as their body.
- **Baseline.** The two pages (P-TXT and P-PIX at three viewports), the header and the footer.
- **Gate.** P-TXT equal, P-PIX within budget; **token gates T4 to T6** (no raw values in any component, element-level computed-style parity, no token used that the old page did not use; consumer plan, section 17.3b); adding a page by adding a file works; `bos check` refuses a block in a slot that does not
  allow it and a block whose content does not fit its renderer.
- **Demo.** `bos g page about`, write Markdown, see it; the two legal pages match the old site.

### M3 — Content types, collections, sources, and list and detail pages (about 4 scopes)

- The **content-type format**; the **`file` source** in both stacks and the in-memory fake; organisms `Accordion`, `CardGrid`, `FeatureCards`,
  `Steps`; layouts `listing` and `article`; `[slug]` pages; generated `sitemap.xml`, `robots.txt` and `llms.txt`; `redirects`.
- In `bos-go`: `/api/content/{type}` served from its own file source, and the web's **`api` source** adapter.
- **Pages.** The FAQ, the guides list and detail, the services list and detail.
- **Baseline.** Those URLs, and the old sitemap, robots and `llms.txt`.
- **Gate.** P-TXT and P-PIX per page; the sitemap URL set, robots rules and `llms.txt` equal for the pages that exist in both; **switching a
  collection from the `file` source to the `api` source changes no page's output** (a test that renders both and compares).
- **Demo.** The same FAQ block reads files, then the API, by changing one line of `site.yaml`.

### M4 — The constructor experience, and the rest of the static pages (about 4 scopes)

- **Presets** at all four levels, each beside its kind (`site/brand/presets/`, `site/blocks/presets/`, `site/pages/_presets/`, `site/presets/`); `bos g collection` and `bos g preset`; the real `bos new`; a complete `bos check`; a **component gallery**
  in development (`/_bos/components`) that renders every renderer with its presets and a fake source.
- **Pages.** The cookies page (with a table block: the old sanitizer strips tables, so the baseline records exactly what is expected), the
  about page (hero and a contact block), the page built from sections, the home page (an ordinary page with a configurable block order; an
  override only where a block cannot express it), the redirects, the footer's legal links.
- **Gate.** P-TXT and P-PIX for each page; every static page of the old site is ported; `bos check` is clean; the number of app-side files
  per page is recorded (a page is one file here, against several rows or files before).
- **Demo.** A new site from `bos new`, three pages by generators, a brand, in minutes.

### Phase D — dynamic, milestone by milestone, each with a baseline and a parity gate

| Milestone | What | Parity | Scopes |
|---|---|---|---|
| **M5** Database and runtime site configuration | a Postgres service (already a record and a bundle; a `redis` record and bundle when the first kit needs the cache), the `db` source, site configuration stored and seeded from `site.yaml`, `bos migrate` and `bos seed`, health with the database | P-HTTP and P-API for the site-configuration endpoint's shape | 3 |
| **M6** Identity and the admin shell | authentication, roles, the admin shell and its navigation, the page editor (pages become `db` entries; files become seed); **the authentication defect is fixed in the original first** | P-HTTP on the auth endpoints and the cookie attributes; the admin's routes and permissions | 4 |
| **M7** SEO and media | generated SEO surfaces from data, storage driver, uploads, canvas, watermarks | P-API, P-TXT | 3 |
| **M8** The other kits | contacts and inquiries with the notifier, calendar, news with the job runner, analytics, documents, reference data | P-API, P-TXT per kit | 7 |
| **M9** Real-estate | properties (the largest surface), transactions, collections, agents; the app keeps only its own decisions | P-API and P-TXT on the catalog and the property page | 4 |
| **M10** Production bundle and cutover | the production profile, the parallel run | its own plan and approval | — |

The **hard parts are scheduled, not deferred**: the property domain and the admin are named milestones with their own baselines, so the
easy pages cannot make the project look finished early.

## 4. The bundle, the action and the Makefile (M0)

### 4.1 The bundle

Docker Compose, composed by the existing `infrastructure` machinery (records `service` and `service-bundle`, the `bundle-compose`
template, the `render-bundle` and `run-bundle` actions, the `bootstrap-bundle` process), extended.

| Service | Kind | Built from | Notes |
|---|---|---|---|
| `bos-api` | application | the framework's Go Dockerfile template and the app's `apps/api` | health check on `/health`; the port and origins from `bos.yaml` |
| `bos-web` | application | the framework's Node Dockerfile template and the app's `apps/web` | depends on `bos-api` being healthy; `PUBLIC_API_URL` from `bos.yaml` |
| database, admin tool | datastore, tool | existing records (postgres, pgAdmin) | **added only by the milestone whose kits need them (M5)**, never in the empty app |
| cache | datastore | a **new `redis` service record** and its own bundle, by the same process that made postgres and pgAdmin (no code) | added when the first kit needs it (the old app requires a cache in production only, so not before M6) |

Two profiles with the same images: **dev** (the app halves run natively with hot reload through `make dev`, or all in containers with
`make bundle-up`) and **prod** (M10). Secrets never appear in tracked files: compose holds `${VAR}` references, `.env.example` holds names.

**Base images are pulled once, deliberately, before any gate** (Docker on this machine is slow and stalls); gates build with the cached
images and never pull.

### 4.2 The action and the process (definitions to be proposed one by one at M0)

| Definition | Kind | What it is |
|---|---|---|
| `service` schema extension | schema | adds `build` (context and Dockerfile), `depends_on` (with a health condition) and `healthcheck`, the first fields the app services need |
| `bundle-compose` template extension | template | renders `build:` instead of `image:` for application services, the health checks and the dependencies |
| `verify-link` | action (shell) | waits for `bos-api` to be healthy, then requests the web page and checks that it shows the API's status: it proves the two halves are linked, not just running |
| `bootstrap-app-bundle` | process | `render-bundle` → `run-bundle` (with `--build`) → `verify-link` |

Each is proposed with its own tree and approved individually, as the workspace rule requires.

### 4.3 The Makefile (in the app)

| Target | Does |
|---|---|
| `make help` | lists the targets |
| `make env` | `bos env`: writes both halves' environment from `bos.yaml` |
| `make dev` | starts the API and the web natively with hot reload |
| `make build` | builds the Go binary and the Svelte app |
| `make test` | the app's tests and, from the workspace, the frameworks' |
| `make check` | `bos check` |
| `make bundle-up` / `bundle-down` / `bundle-logs` | run the `bootstrap-app-bundle` process, and `docker compose down` and `logs` |
| `make clean` | removes build output |

**Two levels, with examples in the consumer repo** (marked *EXAMPLE, not formalized*: `Makefile`, `apps/api/Makefile`, `apps/web/Makefile`). The
product Makefile is the one entry point: `make api-<target>` and `make web-<target>` run that app's own Makefile, `make dev` runs both at once,
`make build`, `test`, `check` and `clean` loop over the apps, `make links` checks that the local framework directories exist, `make env` writes
both apps' settings from one place into `.bos/env/`, and `make bundle-up`, `bundle-down`, `bundle-logs`, `bundle-status` and `bundle-config`
drive the docker bundle once it exists. Each app's Makefile also works alone (`make -C apps/api dev`). In the examples the settings are variables
in the product Makefile; formalized, `bos env` writes them from `bos.yaml`.

The Makefile is thin: each target calls `bos` or `process-cli`, so the logic lives in one tested place and the Makefile stays the same
size as the app grows.

## 5. Risks

| Risk | Mitigation |
|---|---|
| Different implementations never match byte for byte | the five parity levels replace "identical"; budgets are recorded per page and tightened over time |
| Pixel comparisons are flaky (fonts, animation, timing) | the capture tool disables animation, pins fonts to local files, fixes the viewport and waits for network idle on loopback; a page captured twice must give identical golden files (the B0b gate) |
| Building from empty repeats the earlier "the easy part looks done" trap | the dynamic and hard milestones (M5 to M9) exist with their own baselines from the start |
| Components drift from their declarations (the earlier bos) | one folder, one `meta.ts`, and the props type derived from it; `bos check` reads the same object |
| A generator or resolver becomes a monolith | sources, renderers and layouts are registries; nothing switches on a role name |
| The old app is hard to run locally | B0a needs no database; B0b needs a local Postgres and the original's seeds; both are read-only |
| Docker is slow or stalls here | base images cached once; gates never pull; dev mode runs natively |
| Two sources for one setting (token lifetimes) | one `bos.yaml` key read by both halves, checked by the auth milestone's parity gate |
| Scope creep into dynamic features early | static first; no database before M5 |
| The empty API compiles in the ported auth package's dependencies (a JWT library, a Postgres driver, a Redis client) because the middleware is imported as it is | accepted: nothing is rewritten (D25); the cost is compile size only, since the services are not contacted; revisit when kits may become the Go modules (design D6) |

## 6. Decisions proposed

- **D5 (revised).** Approach: **spine first, baseline parity.** The new app is built from an empty app on the frameworks; the old app, run
  read-only, is the oracle; each milestone ports one part and must match its baseline. This replaces "strangler in place" (importing the old
  app and moving slices out). The earlier risk of a greenfield build is answered by scheduling the hard parts as named milestones.
- **D19 (revised).** The bundle is composed by the existing infrastructure machinery, extended; **the first bundle is the empty app's, in M0**.
  A new datastore such as `redis` is a service record and a bundle by the same process, not code.
- **D20 (proposed).** The **baseline and parity levels** of section 1: the old app run in place is the oracle; five parity levels; the
  baseline is captured incrementally, only for what the next milestone ports.
- **D21 (proposed).** The constructor model and the rule "code declares, data selects" (`docs/proposals/bos-constructor.md`).
- **D22 (proposed).** Brand and tokens: one folder, `site/brand/`, with the config, W3C-structured YAML token sources in three tiers plus modes,
  presets, assets and plugins; generated output never committed; every kind of the constructor follows the same one-folder convention.
- **D23 (proposed).** Assets and static delivery (`docs/proposals/bos-constructor.md`, section 4.7).
- **D24 (proposed).** Every module and package path is inside the sdlc-kit repository (`github.com/shredbx/sdlc-kit/platform/<lang>/...`, `@sbx/*`);
  only clients have repositories of their own; consumers link by local directory and nothing is fetched.
- **D25 (proposed).** Reuse before writing: ported packages are imported as they are; running code is moved into the frameworks and configured,
  never rewritten; new code only where nothing runs today; the golden files are the evidence for a move.
- **D26 (proposed).** Tests come after the core: none are written while the core is built; the gates are the offline build and vet, the
  golden-file comparison and `bos check`; tests are added, as scopes of their own, once the core is built and validated.
