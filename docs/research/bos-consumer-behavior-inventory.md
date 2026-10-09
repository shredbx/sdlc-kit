# BOS consumer behavior inventory

**Status:** Checkpoint 0 working inventory; pinned-source inspection started, runtime verification
pending.
**Scope:** BestieRealEstate (BR) and Bestays, as pinned by the `sdlc-kit` checkout.
**Boundary:** Read-only inventory. No consumer code, production data, credentials, or migration
changes are included.

## Evidence baseline

| Evidence | Revision or location | What it establishes | Limit |
|---|---|---|---|
| `sdlc-kit` checkout | `80d0527` (merge of PR #4) | The architecture and delivery-plan documents reviewed below are merged. | Does not prove either consumer's current runtime behavior. |
| BR submodule pin | `4bc61769208a09a9c046a92caac92deadd348247` | The pinned consumer source is now checked out. `apps/api/main.go`, the Svelte root page, and `tests/oracle/` were inspected. | This checkout is the new M0 BOS shell, not the full historical BR implementation. The old API oracle records routes and HTTP behavior, but the old app source itself is not present here. |
| Bestays submodule pin | `eeddc43a3d90d07aecc2c0c2ab83967c1d96faba` | The pinned source includes the Next.js/Supabase Bestays app. Public listing/detail, booking API handlers, booking SQL, and filtering code were inspected. | No runtime was started; source behavior has not been exercised against local services or data. |
| [Delivery plan](../plans/2026-10-08-bos-bestays-consumer-delivery-plan.md) | Section “Evidence baseline and limits” | Reports a separate source snapshot at `a96d9d058fb312eb896e2509710905a4acd6c8bb`, including a Bestays Go API. | The snapshot path `references/shredbx` is absent from this checkout. The pinned Bestays submodule inspected here is Next.js/Supabase and has no tracked Go files. Verify that these are the intended same baseline before using the plan's Go API description as current-source evidence. |
| [Architecture review](../proposals/bos-final-architecture-review.md) | Sections “Schema evidence from the supplied Bestays dump” and “Deductive review” | Records schema-level concerns about rent-only properties, booking integrity, public guest data, and Supabase authorization. | DDL review is not a live application, row-data, or runtime verification. |
| BOS packages | `platform/go/packages/`, `platform/svelte/packages/`, and the `bos-go` / `bos-svelte` frameworks | Reusable platform code exists; the delivery plan names candidate domains including property, transactions, contacts, identity/RBAC, calendar, CMS, FAQ, SEO, and media. | Package presence and name overlap do not establish behavior parity or a ready-made consumer composition seam. |

## Journey-to-module matrix

The route-level evidence below is source-traced, but runtime behavior is still unverified and the
candidate BOS mappings are hypotheses. Use `reuse`, `extract`, `consumer-only`, `defer`, or
`decision required` for the final disposition.

| Consumer / journey | Source evidence: route, handler, UI, storage, auth | Observed behavior and test baseline | Candidate BOS surface | Initial disposition | Open evidence / decision |
|---|---|---|---|---|---|
| BR — current demo home and API health | `apps/web/src/routes/+page.server.ts` loads `GET /health`; `+page.svelte` renders `PlaceholderHome`; `apps/api/main.go` starts `bos-go` with no product dependencies. `tests/oracle/baseline/api/root.json` records the old API's legacy route list. | Source-traced; local run pending. The consumer README reports the API baseline previously matched 18/18 cases, but this was not rerun here. | Reuse the `bos-go` / `bos-svelte` shell. Legacy business routes need source-backed extraction decisions; the old route list alone is not their behavior contract. | reuse (shell); decision required (legacy feature) | User can run `make dev`, `make check`, and `make parity` in the BR consumer. Identify one real public or admin journey from the old app before selecting a business slice. |
| Bestays — browse rental listings and filter availability | `projects/bestays_app/web-nextjs/src/apps/bestays-web/app/(public)/listing/page.tsx` validates query parameters with `FilterCriteriaSchema`; `entities/filter/libs/filter-service.ts` reads published rent rows from `CMS_PROPERTY_LISTINGS_VIEW`, filters location/price, and excludes a property if any booking overlaps the requested dates. | Source-traced; local run pending. The service returns an empty list on listing-query error; its booking lookup does not inspect the returned error. | Property/search, media, and a booking-aware availability surface are candidates. | extract (behavior); decision required (availability contract) | Confirm whether availability is property-wide or unit-aware. Current filtering excludes the whole property for any overlapping booking, even though booking data has unit IDs. Capture one filter journey and its local result. |
| Bestays — open a rental detail | `app/(public)/p/[id]/page.tsx` loads `PUBLIC_PROPERTY_DETAILS_VIEW`, restricts to `transaction_type = rent`, and builds title/description/share metadata. | Source-traced; local run pending. | Property, media, and SEO are candidates. | extract (behavior); decision required (field contract) | Record the visible fields and the actual data/view source in a safe local run; compare against the BOS property contract before porting. |
| Bestays — create/manage a CMS booking | `app/api/properties/[id]/bookings/route.ts` requires authentication, validates input, checks the property, requires `unit_id` when active units exist, and delegates overlap rejection to database triggers. `db/sql/6.booking.sql` through `8.fix-overlap-detection-for-units.sql` evolve the trigger and unit model. | Source-traced; local DB/API run pending. | Separate booking module with property/unit, persistence, and authorization contracts. | decision required | Resolve lifecycle/status, unit/property consistency, overlap/concurrency, price basis, and guest-data rules. Verify the migration sequence and actual local trigger behavior before adopting it. |
| Bestays — public availability calendar | `app/api/public/properties/[id]/bookings/route.ts` validates property/unit UUIDs and selects `id`, `start_date`, `end_date`, and `unit_id`; it omits guest name and notes. | Source-traced; local run pending. | Public availability projection, not the booking record itself. | extract (behavior); decision required (response contract) | The current handler returns an internal booking ID, while the delivery plan's acceptance check says public responses must omit internal booking IDs. Confirm the required response shape and test it. |
| Both — selected shared back-office behavior | Plan names identity, CMS/guides, SEO, contacts, media, calendar, and admin shell; this inventory has not yet traced matching BR and Bestays journeys. | Pending local runs and BR historical-source evidence. | Candidate existing BOS package/framework surfaces. | decision required | Inventory actual screens, routes, APIs, roles, and data dependencies before deciding what is common. |
| Bestays legacy Supabase data / auth behavior | Supplied DDL and checked-in migration scripts; no production rows read. Pinned app uses Supabase clients and `auth`/RLS assumptions. | Not run against a target database. | Plain-Postgres persistence and selected BOS identity boundary are candidates. | defer / decision required | Define the authorized record population and field/relationship mapping; no production access or import is approved by this inventory. |

## Decisions and constraints to carry forward

- The proposed target is two separate consumers composed from shared BOS surfaces, not a copy of BR
  forked into Bestays. BR selects sale/lease; Bestays selects rent/booking. These selections need
  tests proving disabled offerings are not registered or reachable.
- Current shared property validation is reported to require sale or lease, so the rent-only
  Bestays profile is not yet proven compatible.
- Booking is not equivalent to the existing sale/lease transaction lifecycle. Do not derive booking
  rules from the schema alone or expose booking records/guest details through public availability.
- Do not assume the supplied legacy Bestays Supabase schema describes the same data or behavior as
  the pinned dashboard source. No row data has been read or imported.
- A manual, bounded composition proof should precede bootstrap or generated examples. The consumer
  inventory does not authorize implementation of either.

## Local baseline to capture

Run each pinned consumer locally using only approved local configuration/data. For one concrete
journey in each app, send back:

1. The exact checkout/source revision and startup command.
2. The actual local URL and port printed by the app.
3. The steps performed and the observed result, including relevant screen/API output.
4. Any test or smoke-check command and its result.
5. Anything that cannot be run without production access, private credentials, or data.

Do not paste secrets or production records. These observations are still pending; no app was run as
part of preparing this sheet.

## Checkpoint 0 exit gate

This inventory is ready for review only when the selected BR and Bestays journeys have source
revisions and route/API/UI/storage evidence, the mismatch between the delivery-plan snapshot and
the pinned Bestays submodule is resolved, their candidate BOS disposition is explicit, and
unresolved booking, role, data, and migration decisions are recorded. Local runtime checks and a
historical BR source comparison are still pending. Then choose and approve the smallest composition
proof; do not start dependent feature implementation before that review.
