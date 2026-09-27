# bos — port plan: the real CMS/appearance system, not a rebuild

Last updated: 2026-09-27
tags: bos, bos-demo, cms, port, kit-placement, proposal

Status: **proposal, not yet approved.** Written after confirming a real research gap: `bos-page-designer.md`
and `bos-site-chrome.md` checked `consumers/clients/bestie/bestierealestate` (this repo's own thin-consumer
submodule, `sdlc-bestie-bestierealestate` — its own from-scratch repo, currently just M0's placeholder) and
concluded there was nothing real to port. That conclusion was true of that path and false about the world: the
real, mature production app lives at `/Users/solo/Projects/workspaces/shredbx/clients/bestie/projects/bestierealestate`
(Go `apps/api-chi` + SvelteKit `apps/web-svelte` — confirmed present, per `docs/research/shredbx-codebase-inventory.md`
line 201: "Largest, most mature client product in the repo"). This plan is grounded in files actually read there and
in this repo, not directory listings.

## 1. What the real system actually does

**A CMS page is not what `bos-demo` built.** `apps/api-chi/internal/handler/cmspage.go` (255 lines) and
`internal/repository/cmspage.go` (208 lines) implement a page keyed by a validated kebab-case `Slug`
(`cms.Slug`, regex `^[a-z0-9]+(?:-[a-z0-9]+)*$`), not a UUID id, with:

- `Title`, `BodyMarkdown` — simple fields, both nullable (COALESCE-preserved on partial save).
- `Details` (`cms.PageDetails`) — a large, typed (never `map[string]any`) structured payload: an optional
  `PageContact` block, hero fields (headline/eyebrow/cover images with slideshow controls), closing-band copy,
  homepage section ordering, hero color tints validated against a CSS-injection-safe regex. One JSONB column,
  additive by design — "adding them needs NO migration" (`cmspage.go:86`).
- `DraftContent` / `PublishedContent` (`cms.SectionList`) — **two separate JSONB columns**, not one. Decision
  #0273 split them; Decision #0281 ("AE4 — DIRECT-SAVE") later collapsed the workflow so `Save` writes straight
  to `PublishedContent`, gated live by `Published` (`cmspage.go:500-514`). `bos-demo`'s `pages.go` has no
  draft/live distinction at all — a section edit is instantly live regardless of the page's publish flag.
- `SeoMeta` (`seo.SeoMeta`, a shared value type also used by `property`) and `Version int` (bumped on every
  save, no separate audit table).
- `Validate()` enforces real invariants bos-demo never modeled: body length cap, CSS-color-injection defense on
  every hero tint field, geometry checks on the contact address, per-field length caps on closing-band copy.

**`SectionList` is a closed, typed, self-registering renderer registry — the Go-side twin of what
`bos-svelte/src/lib/renderers/registry.ts` reinvented in TypeScript.** `platform/go/packages/cms/section.go`
(already in *this* repo): each kind is "an ADAPTER: a payload struct in its own `section_<kind>.go` file that
registers a constructor in `init()`" (`section.go:7-9`) — five kinds exist today: `hero`, `prose`, `featurecards`,
`steps`, `faq` (`section.go:34-39`, files `section_hero.go`/`section_prose.go`/`section_featurecards.go`/
`section_steps.go`/`section_faqlist.go`). An unregistered kind is rejected on decode — "the set stays closed"
(`section.go:9`). This is exactly `bos-app-roadmap.md`'s M2/M3 organism list (`Hero`, `Prose`, `FeatureCards`,
`Steps`, an accordion/FAQ) — the roadmap's vocabulary was clearly derived from this real system, not invented
independently.

**"Surface" is a real, richer concept than a raw page** — confirmed by reading
`apps/web-svelte/src/routes/(admin)/manage/pages/[surface]/+layout.server.ts` in full (132 lines). A surface is
a registry entry (`SITE_SURFACES`, resolved via `resolveManageSurface`) mapping an admin route segment to: a
public URL, a label, an optional `cmsSlug` (only surfaces with editable content have one — `home` does not),
and per-surface behavior flags. Critically: **only the `sell` surface has a draft/publish toggle at all** —
"every OTHER content surface saves 'edit = live'... or has no cms row (home)" (`+layout.server.ts:104-107`).
Also present at this layer: RBAC (`requireRole(locals, url, 'manager', 'super-admin')`, line 38 — matching
`cmspage.go`'s own doc comment: "The `agent` role manages properties + landlords but must NOT edit CMS pages"),
view-count insights, and a facet system (dynamically-resolved listing pages, e.g. by property type/location)
that gets "managed as per any other static page" (line 41) — materially bigger than anything asked for yet.

**Appearance (header/footer) is `siteconfig.header_nav`**, confirmed by reading
`apps/web-svelte/src/routes/(admin)/manage/pages/appearance/header/+page.server.ts` (27 lines): "The header
navigation editor over `siteconfig.header_nav`... inherited from `(admin)/+layout.server.ts`." This is the
exact shape `bos-site-chrome.md` §4 already cited (via `bos-consumer-plan.md` §13) before this plan existed —
that earlier citation was accurate; only its own conclusion ("nothing to port as a file") was wrong. The real
editor (`+page.svelte`, 541 lines) composes its link-destination picker from three families — static
`SITE_SURFACES`, live indexable facets, and property collections — none of which `bos-demo`'s ~100-line
hand-built header editor has any equivalent of.

## 2. Kit/package placement

**The entity layer is already ported — correctly, and unused.** `platform/go/packages/cms/` exists in *this*
repo right now: `cmspage.go`, `entry.go`, `section.go`, `section_hero.go`, `section_prose.go`,
`section_featurecards.go`, `section_steps.go`, `section_faqlist.go`, plus their test files — a byte-for-byte
match against the real source's `cmspage.go` I read. Its `go.mod` already wires correct local `replace`
directives to sibling ported kits: `persistence/repository`, `persistence/database`, `seo`, `location/address`,
`contacts/socialnetwork`. This satisfies the placement test in `platform/CLAUDE.md` already — `cms` is its own
kit (a package named for its kit, `packages/cms`), which fits rule 3 ("a package named like its kit is the kit
folder"). **Nothing needs to move or be renamed here.** The gap is one layer up:

- **`CmsPageRepository` and `CmsPageHandler` were never ported. Correction (2026-09-27, user-caught): this
  plan's first draft put them in `bos-demo/apps/api/internal/cmspages/`, reasoning by analogy to the original
  app's own layering ("CmsPage is BR-project-scoped... lives in the app's `internal/` tree", `cmspage.go`'s doc
  comment). That analogy doesn't hold here — the original app never had a shared framework to place things in,
  so everything sat in one `internal/` by default, not by a placement decision. `platform/CLAUDE.md:16`
  already names this exact question as **open decision D15**: a kit "later also carries its bos wiring
  (handlers, migrations, routes, admin pages) — where that lives is open." Line 17 of the same file already
  answers half of it: a **framework** "composes kits; no domain of its own" — so CMS handlers cannot live in
  `bos-go` without giving it a domain. And the consumer must stay thin (standing goal of the whole `bos`
  track). That leaves one place: **the `cms` kit itself.**
  Confirmed directly against the real source (`internal/repository/cmspage.go:36-45`): `CmsPageRepository` is
  nothing but `postgres.PostgresStore[cms.CmsPage]` (the already-shared, generic engine — same one
  `pkg/property` and `internal/agent` use) plus two slug-keyed methods (`GetBySlug`, `UpsertBySlug`) the
  generic store can't express. There is no logic here that is specific to *bos-demo the app* — it is all
  specific to *CmsPage the entity*, which is precisely what a kit owns. `CmsPageHandler` is the same shape one
  layer up (HTTP glue over that repository). **Resolution: `platform/go/packages/cms/` grows a
  `postgres.go` (repository) and an `http.go` (chi `RegisterAdmin`/`RegisterPublic`, the same two-function
  shape `bos-demo`'s own `settings` package already uses), plus a `migrations/` folder for the `cms_pages`
  table.** `bos-demo/main.go` only ever constructs the kit's repository and calls its two `Register*`
  functions — no new `internal/` package, and `internal/pages` is deleted rather than renamed. This is offered
  as the resolution to D15 for kit-level bos wiring generally, not just for `cms` — worth confirming with the
  user once, since it then applies to `settings`/chrome's own eventual port too (§3, §6).
  **Open mechanical question, not resolved here:** `database.Migrate(ctx, db, "migrations")` in `main.go`
  currently reads one directory. Once a kit ships its own `migrations/`, the runner needs to combine the kit's
  migration source with the consumer's own — needs a small design decision before P1 lands, not guessed at.
- The Svelte side has **no equivalent port yet at all** — checked `platform/svelte/packages/` structure is out
  of scope for this pass (not read); the appearance/header editor alone is 541 lines, over 5x anything hand-built
  in `bos-svelte/src/lib/chrome/` so far. This needs its own reconnaissance pass before a kit-placement call can
  be made honestly — flagged in §6, not decided here.

## 3. Reconciliation — what this session already built

| Built this session | Recommendation | Why |
|---|---|---|
| `bos-demo/apps/api/internal/pages/{pages.go,handlers.go}` (flat `Page`/`Section`, one `published` bool, opaque `json.RawMessage` content) | **Replace** | `cms.CmsPage` is a strict superset: typed closed-kind sections (not opaque JSON), draft/published split, SEO meta, versioning, real validation. Re-implementing it adds nothing; it's already sitting ported and unused. |
| `bos-demo/apps/api/internal/settings/` (header/footer as ad hoc JSONB on a `settings` singleton) | **Replace, pending §6's Svelte recon** | Real shape is `siteconfig.header_nav`, backed by a materially richer editor (facets, collections). Don't guess the schema without reading the Svelte data flow fully first. |
| `bos-svelte/src/lib/chrome/` (hand-built `Header.svelte`/`Footer.svelte` + `import.meta.glob` registry) | **Likely replace**, mechanism validated | The `import.meta.glob` registry idea is independently confirmed correct — the real Go side does the exact same thing via `init()`-registration (`section.go`). Not wasted insight, but the concrete components should align to whatever the real appearance editor actually renders once read. |
| `bos-svelte/src/lib/renderers/` (registry.ts mechanism + `Hero.svelte`/`Prose.svelte`) | **Keep the mechanism, reconcile the components** | Registry pattern is the right shape (matches `section.go` exactly). Field names in `Hero.svelte`/`Prose.svelte` should be checked against `section_hero.go`/`section_prose.go`'s real payload structs before assuming they match. |
| `bos-go/authfake/` + `bos-go/server/middleware/role.go` (`RequireRole`, roles `admin`/`content-manager`) | **Keep as infrastructure, rename later** | Mechanism is sound (real code does the same role-gate shape: `requireRole(locals, url, 'manager', 'super-admin')`). Real role vocabulary is `manager` / `super-admin` / `agent` — worth renaming to match once real RBAC is ported, not urgent now. |
| `bos-layout-presets.md` (written, never implemented) | **Do not implement as written** | The real section-kind registry already gives typed content blocks; the "surface" concept is a separate, richer axis (which pages get a draft/publish toggle, which get a `cmsSlug` at all) that a from-scratch "layout preset" doc didn't anticipate. Revisit after §6's Svelte recon, not before. |
| `bos.yaml`, the Postgres/Redis service records, the docker bundle, chi router wiring, `make dev`/`make api-dev` | **Keep, unaffected** | Infrastructure and dev-loop plumbing don't depend on which CMS data model wins. |

## 4. Client-info check

Grepped the entity/handler/repository files read for this plan (`cmspage.go` ×2, `platform/go/packages/cms/*.go`)
for "Bestie": clean outside test fixture files (`cmspage_test.go`, `entry_test.go`, `section_test.go` — expected
per `platform/CLAUDE.md`: "Ported code keeps upstream comments and test fixtures verbatim... each kit's own
refinement slice scrubs them"). The one real client-identifying string is the original app's own Go module path,
`github.com/shredbx/bestierealestate-api` (in the handler's import of its sibling `internal/repository`) — this
disappears naturally when the code is copied into `bos-demo`'s own module, the same import-path rewrite every
port in this repo already does; no separate scrub step. Actual Bestie brand copy (real page text, real contact
details) lives in that app's database/seed data, never inspected here and never copied — consistent with
"client information lives in the client's own repo."

## 5. First concrete scope — the CmsPage repository + handler only

Smallest coherent real slice: bring the missing layer for the entity that's *already* ported, replacing
`bos-demo`'s hand-rolled pages package. Not `faq`/`guides`/`services`/`properties` — bigger, separate, later.

| Scope | What | Gate |
|---|---|---|
| P0 | Resolve D15's migration-discovery question: how `bos-demo`'s migration runner combines a kit-shipped `migrations/` dir with its own | Confirmed with the user before P1 starts — not a code gate, a decision gate |
| P1 | Migration for a `cms_pages` table, shipped in `platform/go/packages/cms/migrations/` (not `bos-demo`'s own migrations dir — see D15 resolution in §2), columns matching `PostgresMapper.Columns()` exactly (`id, slug, title, body_markdown, details, draft_content, published_content, published, seo_meta, version, created_by, updated_by, created_at, updated_at, deleted_at`) | Migration applies cleanly against `bos-postgres` via whatever P0 decides; column list diffed 1:1 against `cmspage.go`'s `Columns()`/`FromRow()` |
| P2 | Port `CmsPageRepository` (`GetBySlug`, `UpsertBySlug`) into `platform/go/packages/cms/postgres.go` (the kit, not the consumer — see §2's D15 resolution), module paths rewritten | `go build`/`go vet` clean; a direct `UpsertBySlug` call round-trips title/body/details/sections/published/seo_meta independently (field-presence semantics preserved) |
| P3 | Port `CmsPageHandler` (`Get`, `GetPublic`, `Save`, `RegisterPublicContentRoutes`) into `platform/go/packages/cms/http.go` as `RegisterAdmin`/`RegisterPublic` (the kit, not the consumer), `bos-demo/main.go` wired to call them directly, existing `RequireRole` middleware reused at the call site the same way `settings.RegisterAdmin` is wrapped today | `bos-demo`'s hand-built `internal/pages` package deleted, not renamed; curl round-trip on `PATCH .../cms/{slug}` and `GET /api/pages/{slug}` matches the real handler's documented behavior (draft edit doesn't unpublish, empty `content: []` clears sections, absent `content` keeps them) |
| P4 (separate, not sized here) | Svelte admin (`manage/pages`, `[surface]` tabs, `appearance/{header,footer,theme}`) | Needs its own reconnaissance pass first — see §6 |

## 6. What this plan does not decide

- The Svelte admin/appearance port's own scope and kit placement — `appearance/header/+page.svelte` alone is
  541 lines; this needs a dedicated read-through before any file tree can be proposed honestly.
- `faq`, `guides`, `services`, `properties`, `transactions` ports — real, large, separate content types, each
  with their own `manage/<kind>/{content,design,seo,analytics}` admin pattern, for when those content types are
  actually being built (real-estate kit, later per the earlier roadmap conclusion).
- Whether to adopt the full "surface" registry concept now or defer it — at `bos-demo`'s current scale (3
  pages), it may be premature; revisit once more pages/facets exist.
- Renaming `admin`/`content-manager` roles to the real `manager`/`super-admin`/`agent` vocabulary — noted in §3,
  not decided.
- Any process-os schema/action/process for repeating this port pattern — the user's own framing ("build actions
  for automations when we need") is conditional; no automation modeled here. One observation for later: several
  steps here (verbatim-copy a Go package, rewrite its module path, wire its `go.mod` replace directives) already
  repeat exactly what Scopes 1–6e did during M0 — worth a process only once a third or fourth instance of the
  same shape shows up, not from two data points.
