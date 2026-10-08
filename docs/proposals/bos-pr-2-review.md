# PR #2 review: BOS demo, CMS pages, layout regions, and SEO

**Reviewed:** 2026-10-07
**PR:** [#2 — bos-demo: static pages, layout regions, SEO wiring, production visual pass](https://github.com/shredbx/sdlc-kit/pull/2)
**Head inspected:** `45408e518c4090f4dc32ad1eed8cbf3511d78c14` (`worktree-bestierealestate--stage-2`)
**Status:** Architecture and scope review, not a merge approval or full line-by-line code/security
review. The source BestieRealEstate submodule was not initialized. A subsequently supplied
PostgreSQL dump was inspected for DDL only; it was not restored, and the source migration history
was not available for comparison.

## What the PR contains

The PR is a full vertical slice for a database-backed, editable BOS demo:

| Area | Delivered in the PR |
|---|---|
| Go CMS kit | PostgreSQL page repository, embedded kit migrations, public and admin handlers, slug validation, page listing, a `layout` field, and typed section payloads |
| Go app assembly | `bos-demo` wires the database, migrations, CMS handlers, site settings, the fake auth adapter, and role middleware in `apps/api/main.go` |
| Svelte BOS framework | Admin shell, header/footer registries and defaults, layout and renderer registries, `hero` and `prose` renderers, and a BOS default theme |
| Svelte UI packages | `ui-cms` for page list/editor/content/settings/SEO screens; `ui-settings` for general settings and header/footer editing |
| Public web | Home and slug routes load database pages, resolve a layout and renderer, and emit SEO tags through the existing `ui-seo` package |
| Demo infrastructure | Local Go + Svelte workspaces, `bos.yaml`, Makefiles, Docker bundle, persistent data volumes, and Postgres/Redis dev services |
| Design notes | Proposals for the demo, CMS port, layout presets, page designer, admin modules, and site chrome, plus the implementation/handoff logs |

The changes total 131 files and +8,033/−14 lines. The PR description reports successful Go build/vet/tests, Svelte check, `process-cli check`, and end-to-end route/admin checks. Those checks were not rerun for this review. The PR explicitly leaves browser-rendered visual verification and real Markdown rendering incomplete.

## Where it fits the supplied architecture

| Architecture layer | Fit | Assessment |
|---|---|---|
| SvelteKit guidelines | Partial | Public and admin data loads execute on the server, but they call the Go API directly rather than through typed per-request app services. The same-origin proxy exists, but those page loads do not compose through the app-framework `Feature`/service pattern. |
| App framework | Partial | Vite registries, package exports, a server hook, and local workspace wiring are useful building blocks. There is no generic `createPlatform`/feature composition contract, no framework bootstrap, and no demonstrated adapter selection or per-request service container. |
| BOS extension | Strong vertical-slice fit | Admin shell, CMS editor, settings, page layout selection, Go-owned routes, and a kit-owned API surface are BOS-shaped. The actual `AdminModule` is only `{ id, navItem }`; permissions/resources/widgets/pages and generated resource CRUD are not implemented. |
| Existing BOS constructor | Mixed | The PR follows “code declares, data selects” for renderers and layouts. It demonstrates a DB page source and persisted layout, but not the file/API/DB source-neutral contract, content-type collections, or the constructor's explicit block/slot model. |
| Existing BOS system plan | Partial | The PR puts Go CMS handlers/migrations in the `cms` kit, a plausible D15(b) direction, but D15/D18 remain documented as open/proposed. The demo's Go `main.go` and Svelte route files are manual composition points, not an enabled-kit manifest/bootstrap. |

The best interpretation is **a BOS reference demo and a framework integration spike**. It is not yet
the app framework itself, nor proof that an arbitrary product can be bootstrapped from it. It also
does not yet establish the required compatibility between the source product's production data
dump and the target BOS data model.

## Architectural strengths to preserve

1. **A real end-to-end integration.** The demo proves useful seams across Go persistence, Svelte
   rendering, admin UIs, and the development bundle rather than presenting only interfaces.
2. **Kit-owned Go behavior.** `cms/http.go`, its repository and embedded migrations show how
   functional behavior can travel with the CMS kit. The public projection excludes internal audit
   fields and the public handler enforces the published gate.
3. **A registry instead of renderer conditionals.** Vite `import.meta.glob` is a fitting
   build-time registry mechanism for Svelte. The PR uses separate folders for renderers/layouts
   and gives consumers a place to add code-owned registrations.
4. **Incremental settings updates.** Header, footer, and general settings use separate typed update
   paths so an edit to one part need not overwrite the others.
5. **Re-use of the existing SEO package.** `SeoHead` and `SeoEditor` avoid inventing a second,
   narrower SEO schema for the demo.
6. **Demo infrastructure is reusable.** The generic service `volumes` field is a small
   infrastructure capability rather than a one-off compose edit in the application.
7. **The demo is visibly consumer-shaped.** It links to the local platform workspaces and has
   application-owned API/web folders. Keep it explicitly configured and composed by hand through
   the Go and Svelte frameworks for the first proving pass; do not require a bootstrap to build it.
   That working composition, together with its scoped change history, is evidence for a later
   bootstrap design.

These are useful implementation assets. Preserving them does not make every current interface a
framework contract.

## Material gaps and mismatches to settle

### 1. One content model, multiple sources

`bos-constructor.md` describes pages as files first, then the same page/content contract backed by
an API or database. PR #2 chooses database-backed pages immediately, which is a reasonable demo
choice and may fit a product that already edits pages. The missing link is the stable contract:
the demo's `CmsPage` and `SectionData` do not yet demonstrate that file content can produce the
same public DTO and layout behavior.

The application architecture should have one schema/DTO and explicitly selected source adapters.
For an unedited landing site, use a file source without a database. For an editable site, seed an
empty database from files, then treat the database as authoritative. Avoid maintaining two
independent schemas or silently choosing between a checked-in page and a database row.

### 2. `kind` is not a layout region

The PR's default layout sends all `hero` sections to its hero region and every other section to
main. The legal layout excludes `hero` and sends everything else to main. This is simpler than the
constructor's named slots/regions and blocks a future page from putting two sections of the same
kind into different regions, or using a renderer in a region for which it is valid but not its
default.

Make the layout declare its regions and accepted block types. Each content item should either carry
an explicit region/slot or be checked against an equally explicit placement declaration. Renderer
type, placement, and data schema are separate concerns.

### 3. The Go/Svelte composition contract is manual

The demo calls `cms.RegisterPublic`/`RegisterAdmin`, `settings.RegisterPublic`/`RegisterAdmin`,
and middleware by hand in `main.go`. Svelte route files directly choose API paths and page
components. This is entirely acceptable for the first example, but it does not prove the proposed
app framework can compose enabled feature modules, migrations, routes, client components, Markdown
renderers, and storage adapters from one validated app spec.

Keep explicit imports and normal SvelteKit route stubs; do not build hidden route magic. Add a
documented, framework-neutral feature/module contract only when it can be exercised by the
manually configured demo; derive bootstrap later from the working implementation.

### 4. The `AdminModule` is a nav manifest, not the BOS module in the guide

`@sbx/bos-svelte/admin` currently defines only an id and nav item. Role filtering is a UI
convenience. It does not yet carry the broader `BosModule` contract (permissions, resources,
widgets, or custom pages), and no generated resource CRUD exists. The Go API remains the
authoritative route gate; `authfake` supplies a fixed `admin` role for the demo.

Treat this as an early seam for menu composition, not as evidence that the business-module
architecture or real identity model is settled. Do not describe fake auth as a production
authentication flow.

### 5. Demo setup is not reproducible from a clean checkout yet

The PR description says seven published pages exist in the development database and are not in the
diff. The demo README documents starting/stopping the persistent datastore but does not document a
page seed/import command. The new CMS migration creates the table; the initial pages/settings
migrations insert the original settings row, and the later migration drops the temporary page
tables. No migration or tracked seed inserts the seven final CMS pages.

As a result, the reported populated dev database is not the same as a clean local start. Before
calling this a repeatable example, provide deterministic demo seed data (or an explicit seed
command) and prove `clean local data → start services → seed → expected pages` without relying on
an untracked, pre-populated database volume.

### 6. Static public route loads are not yet the typed BFF pattern

The public layout and slug-page server loads use `process.env.PUBLIC_API_URL` and call Go endpoints
directly. This stays server-side in the files inspected, so it does not itself prove a browser
request goes directly to Go. It nevertheless bypasses the supplied app-framework pattern of
typed, per-request services and makes the external address appear in a public-prefixed setting.
The server hook's generic `/api/*` proxy is a separate transport path.

For the final architecture, Svelte server routes should call typed feature services, with Go
clients/adapters beneath them. Keep any broad pass-through only where transport forwarding is the
feature and constrain it according to the SvelteKit guide.

### 7. Draft, publication, and content behavior are narrower than their names may suggest

The editor saves the structured section list to `published_content`. A `published` boolean controls
public visibility, but this slice does not demonstrate a distinct draft-edit/preview/publish
workflow; the API comments say a save writes directly to the served content slot. The database
retains `draft_content`, but this editor path does not use it. Also, the editor can add, reorder,
and remove sections, but the review did not find a page delete operation in the implemented
handler surface.

That may be adequate for the stated static-page demo. Describe it as direct editing with a
published visibility gate unless/until a separate draft workflow is implemented and tested.

### 8. Tokens and visual proof are still a separate concern

The PR deliberately uses `bos-brand.css`, not the proposed D22 token pipeline. That is acceptable
for a neutral demo theme, but should not become the source of consumer branding or a substitute for
the brand/token import and parity work. The PR itself says a browser-rendered visual check remains
to be done.

### 9. Production-model/data compatibility is not proven by the demo

The requirement is to preserve the source product's Go/domain model, SQL schema, and persisted
data contract while composing those capabilities into BOS. This applies across the selected
domains, not just CMS pages. The supplied dump contains 76 tables, including contacts, calendar,
CMS, real-estate, inquiries, images, and transactions. Its `cms_pages` DDL includes slug/title/
Markdown, details, draft/published content, publication state, and SEO metadata, giving useful
structural evidence for the CMS slice. It does not, by itself, prove that the complete dump
restores and is readable through the PR's code.

Schema inspection also exposed relevant work in the current sdlc-kit checkout: the generated
transaction mapper omits all six sale/lease amount-and-currency columns present in the dump and
`Transaction`, but the service has separate `SaveMoney`/`LoadMoney` companions. `Create`/`Update`
call `SaveMoney`; `Get`/`List` do not call `LoadMoney`, and there are no callers elsewhere in the
package, so normal reads still do not hydrate the money fields. The companion write also is not
shown as atomic with the deal-row write. In addition, `property_units.cover_image_id` has no FK to
`images(id)`, and independent transaction property/unit foreign keys do not ensure the selected
unit belongs to that property. These observations are from the current checkout, not verified
findings against PR #2's exact head.

### Bestays rental/booking fit (separate from PR #2)

A second supplied schema describes a pre-BR Supabase app with three core tables:
`bestays_properties`, `bestays_property_units`, and `bestays_bookings`. This is not evidence that
PR #2 implements a booking module. It does show a plausible second consumer for the framework,
provided the feature boundary is explicit:

- Compose shared property/contact/content/media foundations as needed; configure Bestays for
  **rent + booking only**. Do not register sale or lease routes, admin actions, or transaction
  types for Bestays.
- Keep BR configured for **sale + lease only**. Do not enable rental/booking in BR absent a
  stakeholder request. Shared source code can contain an optional capability, but each consumer's
  module registration and API must enforce its enabled set.
- Treat booking as its own optional kit/module, not just `transaction.TypeRent`: the existing
  transaction engine models deal terms and close outcomes; a booking needs date-based availability,
  overlap/concurrency handling, and its own reservation lifecycle.
- The current shared `Property` validation requires sale or lease, so rent-only Bestays needs an
  offering-aware invariant or a neutral property core. The Bestays schema's JSONB images/location,
  single `price`, units, and enum need a deliberate mapping to BOS property/media contracts;
  preserving its schema verbatim is not the same as preserving behavior.

The Bestays DDL has a date-order check but no overlap constraint, no same-property/unit constraint,
and no booking status, currency, or price basis. Its anonymous SELECT policy can expose booking
rows—including guest fields—for published properties. Broad permissive authenticated policies also
make the owner-scoped insert policy ineffective as a restriction. `auth.users`, `auth.uid()`,
Supabase grants/RLS, views, triggers, and functions need a deliberate port to the selected
PostgreSQL and Go-auth model; do not copy them unchanged. These are schema-level findings only:
the Bestays rows and application behavior were not inspected.

Keep this as a separate acceptance gate: restore an authorized BR production dump into an isolated
local BOS database; run the planned migrations/mappings; compare schema constraints, indexes, row
counts, key relationships, representative values, and application-level reads across the selected
domains. The supplied dump's row data was not read, and it was not restored. Keep any full dump
outside Git in an access-controlled local location; use sanitized copies for repeatable checks
where possible. Never connect the demo to or write into production. If the schema changes, a tested
migration must preserve meaning and data. UI markup, components, and theme may change or be copied
independently; visual parity is not the data-compatibility gate.

## Recommended disposition

- Keep the PR's CMS Go code, Svelte registries, SEO wiring, settings screens, bundle support, and
  demo as candidate building blocks.
- Before treating them as framework contracts, settle the common page/section schema, explicit
  layout regions, app/BOS split, module registration shape, and source/seed authority.
- Compare the source Go/domain types and migrations with the BOS target before claiming production
  data compatibility; retain a reproducible dump-restore/migration check as the gate. Keep UI/theme
  choices separate from this data contract.
- Add a clean-start seed path and verify the demo from an empty data volume.
- Retain `bos-demo` as the manually configured integration acceptance example. First prove the Go
  and Svelte module/configuration seams there. Later, derive bootstrap templates from this working
  example and the implementation history; keep the generated example and bootstrap source from
  drifting.
- Reconcile proposal status labels and roadmap/`CLAUDE.md` status only after the architectural
  decisions are approved. Building a proposal does not implicitly approve it.

## Verification scope

Read the PR metadata, diff stat/file list, design proposals, and selected source files on the PR
head recorded above. Separately inspected only table/constraint DDL in the supplied PostgreSQL
BR dump and table/view/policy DDL in the supplied Bestays schema; did not read `COPY` data, restore
either dump, inspect the Bestays application, or compare either source against its migration history.
The PR's test results are attributed to its description, not independently verified here. A
reviewer-side `git diff --check` reported trailing blank lines at EOF in
`projects/demo/bos-demo/deploy/.env.example` and `deploy/docker-compose.yml`. No build, test,
browser run, source-submodule inspection, or full code/security audit was performed.
