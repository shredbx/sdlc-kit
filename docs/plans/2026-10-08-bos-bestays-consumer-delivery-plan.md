# BOS Bestays consumer delivery plan

**Date:** 2026-10-08
**Status:** Proposed delivery plan. Scope and estimates require review before implementation.
**Inputs:** BOS system design and app roadmap; the supplied BR PostgreSQL DDL; the supplied
Bestays Supabase schema; existing platform packages and the BOS demo review.

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

The Bestays checkout was not available during this planning pass: the expected consumer directory
was empty/uninitialized. Its supplied SQL schema is not a substitute for the Go/Svelte source,
workflows, auth behavior, or migration history. Therefore the first task is a source-availability
and behavior-inventory gate. Do not claim a Bestays port or parity until that gate passes.

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
- Later bootstrap/example extraction only after the two consumers demonstrate the actual
  configuration and registration seams.

### Out of scope for the initial two-day checkpoint

- Complete parity for every BR back-office workflow.
- Full Bestays migration/import of production data or cutover.
- Payment processing, deposits, external calendar integrations, and other booking features not
  evidenced by the schema or explicitly required by a stakeholder.
- Production deployment, secrets migration, or connecting to production.
- Pixel-identical visual parity or a generalized generator built before both consumer profiles
  run.

The Bestays enum includes `sale`, `rent`, `lease`, and `sale-lease`; without reading source rows or
the app, migration code must not assume that every legacy record is rent-only. Define the
authorized migration population and mapping before any import.

## Delivery checkpoints

The checkpoints are evidence gates and delegation boundaries, not a mandatory process to automate.
Each checkpoint receives its own reviewed scope and exact before/after tree before implementation.
Do not start dependent implementation while its prerequisite evidence or decision is missing.

| # | Deliverable | Acceptance evidence | Estimate after source access |
|---|---|---|---:|
| 0 | **Source and behavior inventory**: make BR and Bestays source checkouts available; inventory BR's routes, admin modules, adapters, models, migrations, and Bestays booking/listing journeys. Mark each behavior `reuse`, `extract`, `consumer-only`, `defer`, or `decision required`. | Traceable behavior-to-kit matrix; source versions recorded; unresolved rules named (booking lifecycle, date boundaries, overlaps, unit/whole-property semantics, price basis, guest data, roles). No production access. | 0.5–1.5 days |
| 1 | **Consumer configuration and composition proof**: one explicit app config per consumer, common Go/Svelte registration, startup validation, and enabled-feature gates. | Both app profiles start locally. BR enables sale/lease and has no rental/booking registrations; Bestays enables rent/booking and has no sale/lease registrations. Focused config/route tests plus curl health and route checks. | 1–2 days |
| 2 | **Shared back-office extraction slice**: extract the common admin shell/registration surface and the first high-value shared modules from BR into BOS without changing BR's behavior. | BR runs through BOS registration; normalized route/API checks match the recorded BR baseline for the extracted slice. Bestays can register the same modules without importing BR-specific app code. | 2–4 days |
| 3 | **Property and adapter composition**: make the shared property foundation independent of a mandatory sale/lease selection; select Postgres/media adapters by consumer configuration. Map BR fields and Bestays's legacy property/unit shape explicitly. | Unit tests cover valid BR sale/lease, valid Bestays rent-only, and rejected disabled offerings. Local Postgres reads/writes preserve representative property and media references. Existing BR property behavior stays green. | 2–4 days |
| 4 | **Optional booking module**: implement the minimum agreed rental reservation lifecycle and safe public availability surface, separate from sale/lease deal transactions. | Tests cover date boundaries, overlapping concurrent reservations, property/unit ownership, lifecycle transitions, and public-field redaction. Manual curl checks exercise the agreed public availability and authorized booking/admin paths. | 2–4 days |
| 5 | **Remaining shared back-office modules and Bestays app wiring**: move only the inventoried reusable BR functionality needed by Bestays; keep product-specific routes/configuration in each consumer. | Per-module route/API checks and the minimum affected Go/Svelte checks pass; both apps have their own config/seed/brand while sharing BOS-owned code. | 2–5 days |
| 6 | **Data-port and integrated acceptance**: implement a reviewed import/mapping for the authorized Bestays scope, and run the clean local stack against both profiles. | Sanitized fixture import checks counts, relationships, representative values, and application reads. Curl smoke: health, public listing/detail, safe availability, admin-auth boundary, and disabled-offering rejection. No dump data committed. | 1–3 days |
| 7 | **Bootstrap/example extraction**: derive the smallest bootstrap and examples from accepted BR and Bestays compositions, not from an assumed feature grid. | A fresh app can be composed from the demonstrated profiles; generated/hand-maintained examples do not drift; a clean start and smoke checks pass. | 1–2 days |

The rough sum is **11.5–25.5 focused person-days after both source repositories and local
dependencies are available**, with the widest uncertainty in the BR feature inventory, the
Bestays application behavior, and the property/booking mapping. This is an initial planning range,
not a delivery commitment. Re-estimate after checkpoint 0 using the actual module count, test
health, and amount of consumer-specific code found.

## The first two-day checkpoint

Two days is a realistic target for a **runnable architecture proof**, not for extracting every
back-office feature, completing booking, migrating all records, and achieving full parity.
Timebox the first implementation checkpoint to:

### Day 1 — make both profiles compose

- Confirm both consumer sources and the local BOS baseline are available; stop and report a
  blocker if Bestays source is still missing.
- Use existing framework/package code first; identify the minimal missing common admin/property
  registration surface.
- Add app configurations that select BR sale + lease versus Bestays rent + booking without
  enabling the other profile's offerings.
- Start isolated local Postgres and required dependencies; expose health checks.
- Validate profile loading and ensure disabled routes/modules do not register.

**Day 1 gate:** both profiles build/start or an exact missing-source/module blocker is recorded;
profile-level tests show the distinct enabled sets.

### Day 2 — prove one real path in each profile

- Run one representative BR property/admin path through BOS and compare it with the available
  baseline.
- Run one Bestays rent-only property plus minimal booking/availability path against local
  PostgreSQL, using only behaviors verified from the Bestays source or an explicitly approved
  provisional contract.
- Exercise with curl: health; public property list/detail; safe availability; authorized admin
  access; rejected or absent sale/lease route in Bestays; rejected or absent rental/booking route
  in BR.
- Run the smallest relevant Go tests plus framework/config checks and one end-to-end database
  test. Record commands, addresses, and results.

**Day 2 gate:** the two profiles are demonstrably composed from common BOS code, their offering
boundaries hold over HTTP, one real path per consumer works, and deferred work is estimated from
observed gaps.

If the source checkout is not ready, Day 1 can deliver configuration/route-gating proof and the
Bestays part of Day 2 must be a schema-backed spike only—not represented as a working consumer or
parity result.

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

Once sources are available, these workstreams can be delegated without competing edits:

- **Inventory:** read-only BR/Bestays behavior-to-kit matrix and booking rule evidence.
- **Framework composition:** Go config/module registration and profile tests.
- **Svelte/admin extraction:** shared shell and selected admin module registration.
- **Booking domain:** isolated schema/domain contract proposal and tests, after booking rules are
  approved.
- **Data mapping:** Bestays legacy-to-BOS field/relationship mapping and sanitized-fixture plan.

Only one owner should integrate edits to the shared app config, route registry, and local service
bundle. Do not delegate several agents to independently redefine the same module contract.
Review every delegated change against the consumer boundary: no Bestays behavior belongs in BR by
default, and no BR-only behavior should leak into Bestays.

## Risks and estimate triggers

- **Missing source checkout:** blocks reliable route, behavior, and data mapping. The current
  worktree had no Bestays files, only the supplied DDL.
- **"All BR admin" is not yet a bounded list:** checkpoint 0 must inventory it; each actual module
  added to the reusable surface changes the estimate.
- **Booking semantics are underspecified by the schema:** overlap, cancellation/status, whole-
  property versus unit-level inventory, and public availability semantics need product decisions.
- **Current shared property validation requires sale or lease:** it needs an approved offering-aware
  model change before Bestays can be rent-only.
- **Supabase behavior is broader than SQL DDL:** auth, RLS, grants, public views, functions, and
  triggers need source review and a deliberate plain-Postgres replacement.
- **Current transaction money reads:** `SaveMoney` is called on create/update, but the current
  service `Get`/`List` paths do not call `LoadMoney`; address this separately if it is within the
  BR data-compatibility acceptance scope.
- **Production data:** no production dump is restored or committed without explicit authorization
  and an isolated, access-controlled environment.

Re-estimate upward if the apps require new workflows rather than extraction, if tests are missing
or failing at baseline, if schema semantics do not map cleanly, or if production-shaped import is
required as part of the two-day checkpoint.

## Decision requested before implementation

Approve the two-consumer composition target and the Day 1/Day 2 proof boundary. Then show the exact
before/after tree and obtain approval for checkpoint 0's source inventory or checkpoint 1's
configuration slice before editing implementation files. Each later definition/scope still
requires its own approval; this plan is not authorization to implement the full table above in one
unreviewed run.
