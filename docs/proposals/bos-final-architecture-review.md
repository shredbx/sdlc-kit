# BOS and app framework: architecture review

**Date:** 2026-10-07
**Status:** Review and recommendations only. Nothing in this document is approved as an
architecture decision, implemented, or modeled as a process-os definition.

This review brings together the existing BOS proposals, the three Svelte/app/BOS architecture
guides supplied for this review, and the implementation on [PR #2](https://github.com/shredbx/sdlc-kit/pull/2).
It reads the PR branch's source as well as its design notes. The companion documents are:

- [`bos-pr-2-review.md`](bos-pr-2-review.md) — scope, fit, gaps, and disposition of the PR.
- [`bos-bootstrap-and-app-framework-architecture.md`](bos-bootstrap-and-app-framework-architecture.md) —
  proposed target layering, bootstrap/example arrangement, and storage/media adapter boundaries.

Consumer-specific routes, data, brands, and migrations remain in their respective consumer
repositories. The BestieRealEstate submodule was not initialized in this worktree, so this review
did not inspect its source code or migration history. A PostgreSQL dump was subsequently supplied
and its DDL was inspected without reading its row data or restoring it. This provides schema
evidence, but does not establish application-level compatibility or successful data migration.

## Executive finding

The documents are compatible at the level of the dependency direction and security principles,
but they describe different maturity and composition models. The SvelteKit guides offer a useful
**app-framework design**: explicit feature modules, server/client boundaries, per-request services,
typed BFF calls, and platform adapter ports. The existing BOS proposals add a **consumer and parity
strategy** that the guides do not: separately deployed consumers, Go-owned business APIs, source
parity, and a staged route from static content to editable content. The BOS extension guide adds a
third layer: **business administration** (admin shell, resources, roles, audit, and CMS editing).

The cleanest synthesis is therefore:

> **App framework is the general composition and runtime layer. BOS is an optional business/admin
> extension built on that layer. Product apps own their spec, code, and adapter selection. First
> configure and prove a representative demo directly through the Go and Svelte frameworks; derive
> bootstrap from that working app and the implementation history, rather than making bootstrap a
> prerequisite for the demo.**

This makes a landing site an app-framework consumer without BOS. A product that needs editing can
enable BOS later; a product that is expected to need business editing immediately can select the
BOS profile at creation. The first `bos-demo` should be hand-composed through the framework APIs
and configuration, so it proves what those APIs actually need. Bootstrap is a later extraction
from that working shape, informed by the per-scope changes and their history. It should be an
additive enablement path, not a second independently maintained scaffold or a rewrite of the app.

PR #2 is valuable evidence for the BOS side of this picture: it connects a Go CMS package, a
SvelteKit public site, editor/settings screens, page layouts, SEO, and a development bundle. It is
not yet evidence that the app framework can bootstrap both a plain site and a BOS app, that the
page model works across file and database sources, that a BR production data dump can be restored
into the BOS schema without losing its modeled data, or that the proposed registration interfaces
are stable. The key migration target is the **model/data/SQL layer**, not pixel-identical UI:
presentation and theme may be adapted or deliberately copied while the imported records retain
their meaning and the services can read them. Treat the PR as a vertical-slice contribution, not
the final architecture.

## Inductive review: what the evidence supports

### From the architecture documents

| Evidence | What it supports | Limitation |
|---|---|---|
| SvelteKit architecture guidelines: BFF, server-only modules, no mutable server request state, schema validation, explicit rendering and security decisions | A safe SvelteKit application boundary and operational rules | These are mostly app-level patterns; they do not define cross-platform Go/Svelte composition or product bootstrapping |
| Platform app-framework guide: `packages → frameworks → apps`, a `Feature` contract, ordered hook composition, per-request services, client registries, typed CMS sources | A useful base framework that composes reusable packages without making the consumer hand-wire every request | `createPlatform` is a proposed TypeScript shape, not an existing, tested cross-language contract; some points are explicitly marked `[VERIFY]` |
| BOS extension guide: `BosModule`, admin navigation/resources/pages, Go-owned authorization/audit, resource allow-lists | A distinct business/admin layer that can be optional and can reuse app capabilities | Resource-driven CRUD and the module interface are proposals, not capabilities implemented by PR #2 |
| BOS system/constructor/roadmap: separate consumers, product kits, four configuration layers, parity gates, "code declares, data selects" | Consumer isolation, product ownership, a staged proof strategy, and a content model with adapters | D15 (where kit wiring lives), D18 (kit entry points), D21 (constructor), and the source/layout model still need to be reconciled with the PR |

The app-framework guide's dependency direction (`packages ← frameworks ← apps`) fits the existing
kit layout. Its migration order—first build an app without the framework and extract afterward—is
not the current BOS plan's approved working direction: BOS has chosen a framework spine and an old
app parity oracle. The useful lesson to adopt is the **framework trap mitigation** (small,
documented extension points; ordinary route stubs; ejectability; freeze only after real consumers),
not that alternative migration sequence.

### From PR #2

The PR demonstrates that:

- A `cms` Go package can own its page repository, PostgreSQL migration, and public/admin HTTP
  registration surface.
- A small Svelte renderer/layout registry can be discovered with Vite `import.meta.glob`.
- A product-like demo can assemble Go and Svelte apps locally, serve an editable CMS, render public
  pages, edit site chrome, and use the existing SEO UI package.
- The infrastructure bundle can express persistent volumes for a self-contained demo.

It also shows where the current abstractions are still local to this one demo: the app manually
registers routes in `main.go`, its Svelte pages fetch the Go API directly from server loads, the
admin module contains only a navigation entry, and the page layout assigns sections to regions by
their `kind` rather than by a declared slot/region. The demo is not created by a shared bootstrap
and the seven real published pages are not in the PR diff or a clean-start seed.

The PR's chosen database-backed pages are not inherently at odds with a generic app framework.
They conflict with a **single fixed page-source assumption**. The common model should keep the page
and block/section contract stable while allowing a file source for a static app and a database
source for an edited site. Files can seed an initially empty database; once editing begins, the
runtime database becomes authoritative. Do not keep a separate file page model and DB page model
with different fields and routing semantics.

### From the production-data compatibility objective

The success criterion is not simply “the new screens look similar.” BOS is being composed around
the existing product data model, so the Go domain types, SQL schema/migrations, persisted field
semantics, and required relationships must be compatible with an authorized production dump from
the source application. The concrete gate should restore that dump into an isolated local BOS
database, apply only the approved compatibility/migration steps, and verify row counts, constraints,
relationships, and representative domain reads/writes. Keep a full dump outside Git in an
access-controlled local location; sanitized copies can support repeatable checks. Any intentional
schema transformation must be explicit, repeatable, reversible where applicable, and checked for
data preservation.

The original UI can be changed along the way or its theme copied as useful; visual similarity is
a separate optional parity goal, not a substitute for data-model and SQL compatibility.

### Schema evidence from the supplied dump

The supplied PostgreSQL 17 dump contains 76 tables, including the selected contacts, calendar,
CMS, real-estate, inquiry, image, and transaction domains. This was a schema-only inspection:
table definitions and constraints were reviewed, while `COPY` row data was deliberately not
read; the dump was not restored. The source submodule and its migration history remain
unavailable, so the dump is not yet tied to a specific source-code revision.

The DDL supports a substantial existing model rather than a greenfield BOS schema. The `contacts`
table's 26 columns align with the current Go contact mapper; category links and notes are separate
child tables. `events` has the 13 columns used by the Go calendar mapper, with reminders and
references in child tables. Real-estate listings use normalized location, size, room, policy, and
unit tables; the dump also contains sale/lease flags, prices, lifecycle, and transaction tables.
The `cms_pages` table carries the expected slug/title/Markdown fields plus details, draft and
published content, publication state, and SEO metadata.

The review also found concrete gaps in the current checkout that must be resolved or explicitly
accepted before claiming data compatibility:

1. `platform/go/packages/real-estate/transaction/mapper.generated.go` omits the dump's
   `sale_price_amount`, `sale_price_currency`, `lease_rent_amount`, `lease_rent_currency`,
   `lease_deposit_amount`, and `lease_deposit_currency` columns. The service has companion
   `SaveMoney` and `LoadMoney` methods, so it is inaccurate to say money is never persisted:
   `Create`/`Update` call `SaveMoney`, but `Get`/`List` do not call `LoadMoney`, and a search found
   no callers of `LoadMoney`. Therefore normal service reads do not hydrate those values; the
   separate write also is not shown as atomic with the row write.
2. The dump has a `property_units.cover_image_id` column, but no foreign key from it to
   `images(id)`, despite the current `PropertyUnit` model describing that relationship.
3. `transactions.property_id` and `transactions.unit_id` each have foreign keys, but those
   independent constraints do not ensure that the selected unit belongs to the same property.
   The service must enforce that invariant or the schema must encode it.

These observations come from DDL and source in the current sdlc-kit checkout. They are not proof
that the corresponding PR #2 branch contains or fixes the same mapper/schema. No restore, row
count comparison, data-value inspection, or application-level read/write test was performed. The
attached dump contains production data and must remain outside Git and other shared artifacts.

### Schema evidence from the supplied Bestays dump

The supplied Bestays schema contains three core tables—`bestays_properties`,
`bestays_property_units`, and `bestays_bookings`—plus public views, Supabase auth references,
triggers, functions, grants, and row-level security policies. Its property model is materially
smaller and differently shaped than the current shared Go property model: a single `price`,
`transaction_type` enum, JSONB location/images/cover image/metadata, and separate units with a
name. The current Go `Property` requires `for_sale` or `for_lease`; a rent-only Bestays app cannot
currently pass that invariant. The shared `PropertyUnit` instead has sale/lease prices and
structured size/room fields. This is a manageable migration target, not a drop-in schema match.

The architecture should keep three concepts separate:

- **Shared property core:** the physical listing, unit hierarchy, content, and media references,
  with only selected offerings enabled for a product.
- **Sale/lease transaction capability:** enabled for BR as stakeholders require; absent from
  Bestays' composition unless explicitly requested.
- **Rental booking capability:** an optional booking domain for Bestays, not an alias for
  `transaction.TypeRent`. The existing transaction engine models a deal lifecycle and close
  outcome; a booking needs date-window availability, reservation lifecycle, and concurrency rules.
  Reusing lower-level property and contact contracts is sensible, but booking should have its own
  kit/module, storage, validation, and UI/API registrations.

This yields a configuration boundary rather than two forks of all shared code: BR selects sale
and lease and does not register rental/booking; Bestays selects rent and booking and does not
register sale/lease. The current shared property's listing-intent validation must become
offering-aware (or be separated from the neutral property model) before this composition works.
Test both consumer profiles so a disabled offering has no enabled routes/actions and cannot be
created through the API. Do not add rental defaults or runtime data to BR without a specific
stakeholder request. The Bestays enum contains `sale`, `rent`, `lease`, and `sale-lease`; because
row data was not inspected, the migration must define which legacy records are in scope rather
than assume the database contains rental rows only.

The booking DDL is a useful starting point, but does not yet prove a safe booking system:

- `end_date > start_date` is checked, but no constraint or transaction-level mechanism prevents
  overlapping bookings. `unit_id` is nullable and independently references a unit; the schema
  does not ensure that the unit belongs to `property_id` or define how a whole-property booking
  conflicts with unit bookings.
- There is no booking status, payment/deposit state, guest contact reference, currency, or declared
  price basis in the table. These are unresolved domain requirements, not fields to invent from
  schema alone.
- `bestays_bookings` stores `guest_name` and `notes`, while its anonymous SELECT policy exposes
  booking rows for published properties. In combination with table grants, this can make those
  fields publicly readable. The schema also has broad permissive `authenticated` policies with
  `true`; additional owner-scoped insert policies do not narrow permissive policies, which combine
  with OR semantics. Rebuild authorization deliberately in the Go/API boundary and expose only a
  public availability projection, never booking records or guest details.
- `created_by` references Supabase's `auth.users`, and policies call `auth.uid()`. Roles, auth
  schema/functions, grants, RLS, and Supabase-specific extensions/publication cannot be copied
  unchanged into plain PostgreSQL. Map identity to the chosen BOS auth model and encode ownership,
  admin access, and public reads explicitly. Rebuild views/functions/triggers against the target
  schema and test their security behavior.

Thus the domain boundary is promising, but "easy to port" depends on keeping this as a separately
enabled booking slice and resolving these data-integrity, authorization, and price semantics before
cutover. The schema inspection did not read records, inspect the Bestays source application, or
execute its RLS/functions.

## Deductive review: constraints the final architecture must satisfy

Starting from the goals in the supplied architecture and the existing BOS decisions yields these
invariants:

1. **There is one product-app composition point.** Go and Svelte need language-specific
   registrations, but they cannot have unrelated feature lists, adapter choices, or middleware
   assumptions. One app spec is validated against both sides; code explicitly imports/assembles
   modules so the build remains statically analyzable.
2. **Packages and adapters do not depend on frameworks.** A domain kit declares ports; a selected
   adapter implements them. A framework composes modules and enforces cross-cutting runtime order.
3. **App framework provides capabilities; BOS provides business administration.** SSR/BFF,
   request context, component/renderer registries, Markdown handling, content-source composition,
   theme/SEO integration, and middleware composition belong in the app layer when supported by
   real use. Admin shells, editable business resources, permissions, audit workflows, and the
   BOS content editor belong in BOS.
4. **The same consumer can adopt BOS incrementally.** Bootstrap may choose the BOS profile on day
   one, but adding BOS to an existing app must add only the BOS dependency, configuration, and
   route stubs. It must not regenerate or overwrite the consumer's application.
5. **Bootstrap is not runtime code generation.** It writes normal, consumer-owned files once.
   Runtime registries and packages remain imported from the framework. Overrides remain explicit
   and ejectable.
6. **Storage choices are typed and narrower than “storage”.** Relational persistence, content
   sources, object/file storage, and image transformation are different ports. A single
   `storage` interface should not force PostgreSQL, Supabase, filesystem, and Cloudflare Images
   into a false common abstraction.
7. **Security and cache behavior are designed into the layer boundary.** Server-only credentials
   stay out of the browser; the browser uses same-origin routes; service/API boundaries validate
   and authorize; Go remains authoritative for business authorization and audit. Public content
   may be cached only when it is not personalized.
8. **Data compatibility is a first-class gate.** Preserve the source model and SQL/data contract
   through BOS composition, or document/test a deliberate mapping that proves the dump's
   application data survives. UI/theme parity is a separate choice.
9. **Evidence and approvals remain incremental.** PR #2's demo is not blanket approval to model
   every future module or adapter. Build the smallest framework slice that a real app requires,
   verify it against the demo/consumer, and approve each new definition separately.

## Recommendation

Adopt the layered direction as a **proposal**, not yet as a recorded decision:

```text
product app
  ├── app framework (per platform): composition, request lifecycle, typed registries, adapters
  ├── optional BOS extension: administration and business workflows
  ├── domain kits: CMS, identity, media, property, etc.
  ├── adapter packages: Postgres/Supabase SQL, file/object storage, image processing
  └── app-owned spec, seed, runtime data, and product-specific code
```

Keep the existing four data/configuration layers: app specification, seed, runtime data, and
environment secrets. Keep the current BOS spec as the single product composition source for a BOS
app until a separately approved naming/migration decision says otherwise. Do not add a second
configuration file that duplicates the same feature, database, or API settings.

Resolve the current design forks before turning the demo's local conventions into framework
contracts:

| Decision to settle | Recommended direction | Why |
|---|---|---|
| App versus BOS responsibility | App framework is base; BOS is opt-in and depends on app | Supports both simple landing sites and edited product/admin apps without forcing BOS on every app |
| Page/content source | One schema and page/block model with file, API, and database adapters; seed files are not a second live authority | Reconciles the static-first constructor with the PR's proven database-backed editor |
| Layout placement | Layout declares named regions/slots; content records an explicit placement or an equivalent validated mapping | A renderer kind is not a region; one renderer may appear in multiple regions |
| Kit wiring (D15/D18) | Keep wiring with its functional kit, but depend on a small framework-neutral module contract | Avoids a package importing a framework just to register routes or migrations |
| Demo and bootstrap sequence | First configure the integrated demo directly in Go and Svelte; derive bootstrap later from the proven setup and its history | Tests the framework seams before freezing them into a generator or template |
| Source data compatibility | Treat domain structures and SQL/data preservation as a migration acceptance gate; UI/theme may evolve independently | BOS is composed around an existing product model, not a visual-only recreation |
| Storage | Distinct ports for relational data, content sources, blobs, and transformations | Provider choices are not interchangeable at the same abstraction level |

The proposed platform plans, demo configuration, eventual bootstrap path, and adapter seams are
in the companion architecture proposal.
These recommendations do not authorize a code change, a process definition, or a product-specific
migration. The next step is to discuss and approve the individual architecture choices, then show
the exact tree for the first implementation scope.

## Review limits and verification record

- PR #2 was open when checked on 2026-10-07: base `main`, head
  `45408e518c4090f4dc32ad1eed8cbf3511d78c14`, 131 files, +8,033/−14. GitHub reported no
  review decision and `UNSTABLE` merge state at that time.
- Read PR #2's summary, file list, implementation proposals, and selected Go/Svelte source,
  including the CMS HTTP/repository/section code, demo composition, public page loads, admin
  screens, registries, and README/Makefile. This was an architecture-fit review, not a full
  line-by-line code or security review.
- The PR description reports Go builds/tests, Svelte check, `process-cli check`, and end-to-end
  route/admin checks as passing. Those checks were not rerun for this review. The PR itself says
  browser-rendered visual verification and real Markdown rendering remain incomplete.
- A read-only `git diff --check` reported trailing blank lines at EOF in the new demo
  `.env.example` and `docker-compose.yml`; no files in this review worktree were changed during
  inspection.
- Project-specific source was not inspected. Apply the landing-app/BOS distinction to each named
  product only after its requirements and current code are reviewed in that product's repository.
