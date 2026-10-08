# BOS Bestays consumer delivery plan

**Date:** 2026-10-08
**Status:** Proposed delivery plan. Scope and estimates require review before implementation.
**Inputs:** BOS system design and app roadmap; PR #2 (`worktree-bestierealestate--stage-2`);
the canonical `shredbx` source checkout; existing platform packages and frameworks; the supplied
Bestays Supabase schema.

## Evidence baseline and limits

This plan is updated against the canonical monorepo snapshot at
`references/shredbx`, commit `a96d9d058fb312eb896e2509710905a4acd6c8bb` (2026-10-02).
The BR project history in that snapshot last records project commit `4c5286ed4` (2026-10-02).
BR's own deployment documentation describes `shredbx/bestierealestate` as a generated mirror;
the monorepo, not that deployment repository, is the source used here. The snapshot has not been
compared with a newer monorepo revision or with the running production deployment.

The inspected scale matters to the estimate: BR has a 3,860-line API composition file, 115 API
handler source files, 310 SQL migrations, and 229 files in its Svelte admin route group (88
`+page.svelte` files). Bestays is a separate, much smaller Postgres dashboard: its Go API is 977
lines and it has five SQL migrations. Its booking schema includes `bookings` and `booking_units`,
but the inspected booking-create handler writes only the booking row; it does not create booking
unit rows. Treat this as a behavior and completeness question to resolve, not a feature to copy
without review.

The current BOS checkout already has Go packages covering domains including property,
transactions, contacts, identity/RBAC, calendar, CMS, FAQ, SEO, media, persistence and reference
data. `bos-go` currently supplies common settings, middleware, health/root routes and an API mount;
`bos-svelte` supplies security headers, same-origin API pass-through and a placeholder home. Neither
currently implements the complete cross-platform feature registration and adapter selection
required for thin consumers. Package name overlap is not proof of BR behavior parity.

This was a source/layout inventory, not a complete route-by-route behavior matrix. No BR production
dump rows were inspected or restored; neither application was run; PR #2's reported checks were not
rerun. Any data compatibility, production parity, current live behavior, or test-health claim still
needs its own evidence gate.

## Goal

Make both BestieRealEstate (BR) and Bestays thin consumers of BOS:

- BOS owns reusable Go/Svelte app composition, shared back-office capabilities, domain kits,
  adapters, and optional business modules.
- A consumer owns its app configuration, brand, seed/runtime data, secrets, and genuinely
  product-specific code.
- BR enables the shared back-office capabilities it already needs, property sale and lease, and
  their deal workflows. BR does not enable rental/booking unless a stakeholder requests it.
- Bestays enables the same applicable shared back-office capabilities as BR, except sale/lease,
  and adds rent and date-based booking.
- A later product should be able to select a different set of BOS kits and adapters through
  configuration rather than copy either consumer's application.

The delivery is successful when both consumers run through the same framework composition,
produce their intended behavior, and cannot use an offering that their configuration disables.
This is not a visual-clone requirement. Data structures, API behavior, persistence, and
authorization must be mapped and verified; branding and UI may differ.

## Architecture recommendation: extract shared behavior, do not fork the consumer

**Do not copy all of BR into Bestays and then refactor the copy.** Use the BOS framework and
ported platform packages as the shared starting point. Treat BR as the reference application and
behavior oracle. Where BR still owns reusable handlers, admin screens, or adapters, extract those
into their appropriate BOS kit/framework surface and keep the consumer-specific decisions in BR's
configuration. Compose Bestays from the same reusable modules plus its booking module.

This follows the established BOS direction: framework spine first; old consumer behavior as
read-only reference; move/configure code that already works instead of rewriting it from
descriptions. Copying the entire BR app would initially bring sale/lease assumptions, product
branding, config, routes, and data into Bestays, then require discovering and unpicking those
couplings. A narrow, temporary copy is acceptable only as a measured extraction aid for an
individual module, with its source and destination and the removal gate recorded; it is not the
consumer architecture.

Both consumer sources are now available in the pinned monorepo snapshot. The first checkpoint is
therefore a bounded behavior-to-module inventory—not source acquisition. It must determine which
BR capabilities are genuinely shared, which belong only to BR, what Bestays currently does, and
what must be designed or completed before either app can be called a thin consumer.

The working example should initially be a manually composed consumer, proving the framework
seams directly. Once that composition works, use the framework bootstrap to create the example
app(s) from their consumer configurations, and compare the generated result with the manually
validated behavior. The examples then document real configurations; bootstrap is not a prerequisite
for proving the framework.

## Scope boundaries

### In scope

- Inventory current BR back-office behavior and match each behavior to an existing BOS package,
  framework surface, or a specific extraction task.
- Common app composition/configuration and per-consumer module/adapter registration.
- Reusable BR back-office behavior required by both products, including the applicable admin
  shell and selected CMS/guides, SEO, contacts, media, identity, calendar, and other modules
  confirmed by the inventory.
- A property foundation configurable for BR sale + lease and Bestays rent-only listings.
- A distinct, optional Bestays booking module with a defined booking lifecycle and availability
  rules.
- Plain PostgreSQL persistence, with deliberate replacements for Supabase-specific dependencies.
- A runnable local profile for each consumer and short HTTP smoke checks.
- Bootstrap and example applications derived from the manually validated consumer compositions;
  the examples should demonstrate distinct BR and Bestays capability selections.

### Out of scope for the initial two-day checkpoint

- Complete parity for every BR back-office workflow.
- Full Bestays migration/import of production data or cutover.
- Payment processing, deposits, external calendar integrations, and other booking features not
  evidenced by the schema or explicitly required by a stakeholder.
- Production deployment, secrets migration, or connecting to production.
- Pixel-identical visual parity or a generalized generator built before both consumer profiles
  run.

The supplied legacy Bestays Supabase schema is not identical to the canonical Bestays dashboard's
Postgres schema, and its enum includes `sale`, `rent`, `lease`, and `sale-lease`. Do not infer the
authorized import population from the project name or schema alone. Review the actual source and
record scope with the user before any data import. Do not access production or commit raw dumps.

## Delivery checkpoints

The checkpoints are evidence gates and delegation boundaries, not a mandatory process to automate.
Each checkpoint receives its own reviewed scope and exact before/after tree before implementation.
Do not start dependent implementation while its prerequisite evidence or decision is missing.

| # | Deliverable | Acceptance evidence | Preliminary estimate |
|---|---|---|---:|
| 0 | **Behavior-to-module inventory**: trace selected BR and Bestays journeys through routes, handlers, UI, packages, and migrations. Mark each `reuse`, `extract`, `consumer-only`, `defer`, or `decision required`. | Source revisions recorded; bounded capability list and route/API baseline; unresolved booking, role, data, and migration decisions stated. | 1–2 days |
| 1 | **Consumer config and composition proof**: define explicit BR and Bestays capability selections and validate them at startup. | Tests prove BR selects sale/lease but not rent/booking, and Bestays selects rent/booking but not sale/lease; selected and disabled route registrations are checked. | 2–4 days |
| 2 | **Go app composition slice**: demonstrate compiled module registration, common middleware, route mounting, Postgres/Redis configuration and selected migrations. | A manually composed local app starts; selected routes work; disabled routes are absent; focused Go checks pass. | 3–6 days |
| 3 | **Svelte app composition slice**: establish server-only feature services and consumer UI/admin registration while retaining ordinary SvelteKit route stubs. | A selected public/admin path uses the shared composition seam; module selection matches Go; targeted Svelte checks pass. | 3–6 days |
| 4 | **BR reference vertical slice**: move one agreed high-value BR feature through BOS without changing its source behavior or product-specific data semantics. | Route/API and representative persistence behavior match an explicitly recorded BR baseline; tests cover relevant authorization and error paths. | 4–8 days |
| 5 | **Property, offerings and adapters**: support BR sale/lease and Bestays rent without requiring either product to expose the other's offering; configure PostgreSQL, Redis and media storage by consumer. | Domain/config tests reject disabled offerings; local persistence checks preserve representative property, transaction and media relationships. | 4–8 days |
| 6 | **Bestays booking slice**: implement or complete a separate optional booking module; settle unit assignment, lifecycle, availability and guest-data rules before exposing endpoints. | Postgres-backed tests cover date boundaries, conflicting concurrent reservations, unit/property consistency, state transitions and public-field redaction; authorized HTTP paths pass. | 4–8 days |
| 7 | **Local integration and data acceptance**: run the selected profiles with isolated local PostgreSQL, Redis and pgAdmin; define and test only an approved legacy import scope. | Clean-start documented; curl checks cover health, representative public/admin paths and disabled offerings; sanitized fixtures verify counts, relationships and representative values. | 2–5 days |
| 8 | **Bootstrap and examples**: derive bootstrap from the validated app composition, then create example app(s) from consumer configs. | A fresh example is bootstrapped and runs; BR and Bestays examples show distinct capability sets and do not drift from the tested compositions. | 2–4 days |

The preliminary range for this **selected-module integration pilot is 25–51 focused person-days**,
not including broad parity for every BR admin workflow or a production data migration. A wider
conversion of BR into a thin consumer, plus completed Bestays booking and approved legacy-data
porting, is an order-of-magnitude **60–120+ focused person-days**. These are planning ranges, not
commitments: checkpoint 0 must narrow the selected module count, baseline test health, and
extraction-versus-rewrite ratio before implementation estimates are treated as delivery forecasts.
The previous 11.5–25.5-day range predates canonical source access and is superseded.

## The first two-day checkpoint

Two days is a realistic target for a **runnable architecture proof**, not for extracting every
back-office feature, completing booking, migrating all records, and achieving full parity.
Timebox the first implementation checkpoint to:

### Day 1 — establish the composition seam

- Confirm the exact local BOS baseline and capture the behavior for one agreed BR path and one
  Bestays path from the pinned source; do not imply the snapshot proves the live production state.
- Define the smallest explicit module/config selection seam using existing packages and frameworks.
- Configure the two capability profiles without implementing all module extraction.
- Start only the isolated local services needed for the chosen paths; keep ports/data separate.
- Validate profile loading and that disabled offering routes are not registered.

**Day 1 gate:** both profiles select the intended capabilities in tests and the initial missing
framework seam is bounded. This is not a claim that either full consumer is composed.

### Day 2 — prove one bounded path

- Implement one narrow vertical slice chosen from the inventory—prefer an existing BOS package
  with a directly testable BR and Bestays use, rather than promising full booking in two days.
- Run the path against isolated local PostgreSQL if the selected behavior requires persistence;
  record any missing adapter or model contract instead of hiding it behind a mock.
- Exercise with curl: health, the chosen public/admin path, and checks proving disabled offering
  routes are absent. Record the exact local address, commands and results.
- Run focused package/config tests and one database integration test only if the slice depends on
  persistence.

**Day 2 gate:** a small shared-code integration path is runnable and its consumer boundaries are
measurable. The result is an architecture proof, not a thin-consumer conversion or booking/data
parity milestone.

If a representative path cannot safely fit the timebox, narrow it to config/route-gating proof and
report that no product behavior was ported. Do not manufacture a Bestays booking contract from its
schema alone.

## Minimal validation strategy

Prefer a handful of high-value checks over a broad early test suite:

1. **Profile tests:** BR accepts sale/lease and rejects rent/booking; Bestays accepts rent/booking
   and rejects sale/lease. Check both config validation and actual route/module registration.
2. **Domain tests:** property offering invariants; booking date validity, overlap serialization,
   unit/property consistency, and state transitions.
3. **Persistence test:** one Postgres-backed booking/property round trip including concurrent
   conflicting booking attempts; one sanitized import mapping fixture.
4. **HTTP smoke:** capture the actual local URL/port from the app output; run curl against health,
   public listing/detail/availability, a protected admin path, and disabled-offering paths.
   Public responses must not include guest names, notes, or internal booking IDs.
5. **Small package checks:** run the existing Go tests for changed domain packages and the narrow
   Svelte check for changed framework/admin packages. Run broader CI only if targeted checks expose
   cross-package effects.

Do not guess final endpoint names before checkpoint 0 inventories the existing handlers and the
Bestays journeys. Record the exact curl commands in the implementation checkpoint when those
routes are known.

## Delegation boundaries

After checkpoint 0 and individual scope approval, these workstreams can be delegated without
competing edits:

- **Inventory:** read-only completion of the BR/Bestays behavior-to-kit matrix and booking rule
  evidence from the pinned monorepo snapshot.
- **Framework composition:** Go config/module registration and profile tests.
- **Svelte/admin extraction:** shared shell and selected admin module registration.
- **Booking domain:** isolated schema/domain contract proposal and tests, after booking rules are
  approved.
- **Data mapping:** Bestays legacy-to-BOS field/relationship mapping and sanitized-fixture plan,
  after the authorized legacy-record scope is settled.

Only one owner should integrate edits to the shared app config, route registry, and local service
bundle. Do not delegate several agents to independently redefine the same module contract.
Review every delegated change against the consumer boundary: no Bestays behavior belongs in BR by
default, and no BR-only behavior should leak into Bestays.

## Risks and estimate triggers

- **Snapshot provenance:** the canonical source is pinned and locally inspectable, but this review
  did not compare it to a newer upstream revision or verify it against the live deployment.
- **"All BR admin" is not a bounded deliverable:** BR's 229 admin route files show why checkpoint
  0 must name the selected modules; each added feature changes the estimate.
- **Bestays booking completeness and rules:** the local API does not write booking-unit rows on
  create. Resolve unit assignment, overlap/concurrency, cancellation/status, inventory semantics,
  and public availability before treating the module as ready.
- **BR baseline breadth:** 310 migrations and extensive app-owned route/handler code mean that
  package-name overlap cannot substitute for behavior, SQL, authorization, and data comparison.
- **Current shared property validation requires sale or lease:** it needs an approved offering-aware
  model change before Bestays can be rent-only.
- **Legacy Supabase migration is distinct from the local Bestays app:** auth, RLS, grants, public
  views, functions, triggers and the allowed row population need source review and a deliberate
  plain-Postgres mapping.
- **Current transaction money reads:** `SaveMoney` is called on create/update, but the current
  service `Get`/`List` paths do not call `LoadMoney`; address this separately if it is within the
  BR data-compatibility acceptance scope.
- **Production data:** no production dump is restored or committed without explicit authorization
  and an isolated, access-controlled environment.

Re-estimate upward if the apps require new workflows rather than extraction, if tests are missing
or failing at baseline, if schema semantics do not map cleanly, if broad BR admin parity is added,
or if production-shaped import is required as part of the two-day checkpoint.

## Decision requested before implementation

Approve the two-consumer composition target and the Day 1/Day 2 proof boundary. Then show the exact
before/after tree and obtain approval for checkpoint 0's source inventory or checkpoint 1's
configuration slice before editing implementation files. Each later definition/scope still
requires its own approval; this plan is not authorization to implement the full table above in one
unreviewed run.
