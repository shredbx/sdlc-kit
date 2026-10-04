# bos: our plan as decided, the unmerged PR, an outside design, and my review (proposal, 2026-10-04)

Status: **review and recommendation. Nothing modeled, nothing built, nothing decided by this document.** It collects (1) what we have decided
for porting the first consumer into the bos structure, (2) where the unmerged PR #2 stands against that, (3) an outside design (three
documents the user brought) compared point by point, and (4) my version, with the evidence for each suggestion. Every suggestion needs the
user's approval as its own item (repo rule: per-definition approval).

Generic by rule: this repo holds no consumer information. The first consumer's own facts are in its repo
(`consumers/clients/bestie/bestierealestate/docs/bos-consumer-plan.md`) and are cited by section, not copied.

## 0. What was read, and what was not

Read in full: `docs/proposals/bos-system-design.md`, `bos-constructor.md`, `bos-app-roadmap.md`; the three outside documents
(`platform-app-framework.md`, `platform-bos-extension.md`, `sveltekit-architecture-guidelines.md`, from `~/Downloads/sdlc/`); the consumer plan sections 1, 2 and 15;
`docs/plans/2026-09-27-handoff-next-worktree.md`; `bos-svelte`'s `api-pass-through.ts`.

PR #2 (`worktree-bestierealestate--stage-2`, 131 files, +8033): summarized by a read-only sub-agent from its diff stat and its plan and proposal files
(`git show`). **Not read: its Go and Svelte source. Not run: any build or test.** Where this document says the PR "built" something, that is the
PR's own plan text, not something I verified.

Not checked at all: whether the original app's SvelteKit code matches the outside design's rules (hooks, state, remote functions); whether
`check_kit_edges.py` also forbids a package importing a framework (it checks kit-to-kit edges; section 5, R6).

## 1. Our plan, as decided

Source: `bos-system-design.md` section 5 (decisions D1 to D26). Status below is what those documents say, not my judgement.

| Topic | Decision | Status per the documents |
|---|---|---|
| Consumers | each is its own repo, mounted under `consumers/clients/{client}/{project}`; own deployable bundle; no multi-tenancy (D1, D2) | confirmed by the user |
| Second consumer | same kits plus a `bookings` kit, minus selling (D3) | stated by the user |
| Approach | **spine first, baseline parity**: an empty app on `bos-go` and `bos-svelte`; the old app, run read-only, is the oracle; each milestone ports one part and matches its golden files (D5, D20) | direction from the user |
| Layout | `platform/<lang>/{packages/<kit>/<package>, frameworks/}`; a kit is a functional family named by what it does (D13, D14) | accepted |
| Config layers | spec, seed, runtime, environment (D17) | the four layers confirmed by the user; details proposed |
| Reuse | running code is moved and configured, never rewritten; ported packages imported as they are (D25) | direction from the user |
| Names and links | every module path inside sdlc-kit; consumers link by local directory; nothing fetched (D24) | direction from the user |
| Tests | none while the core is built; gates are offline build, golden files, `bos check` (D26) | direction from the user |
| Constructor model | a site is pages, layout presets, slots, blocks, content types, sources; "code declares, data selects" (D21); brand tokens (D22); assets (D23) | **proposed, not approved** |
| Kit wiring | where a kit's handlers, repositories, migrations and admin pages live (D15); kit entry points (D18) | **open / proposed** |

Ladder (roadmap): B0 baseline harness, **M0 empty app (done)**, M1 branding, M2 atomic library and first pages, M3 content types and sources,
M4 constructor experience and the remaining static pages, then the dynamic phase M5 database, M6 identity and admin, M7 SEO and media, M8 the other
kits, M9 real-estate, M10 production bundle and cutover (its own plan and approval).

Done on `main`: 34 Go modules and 12 Svelte packages ported with their own tests equal to the originals; the unit-record model (Python and Go
slices); `bos-go` and `bos-svelte` extracted from the running app; the first consumer's empty app runs natively and as a docker bundle, 18 of 18
golden cases equal (handoff note). **Next on `main`: M1 branding.**

## 2. The unmerged PR #2

What it is (per its own docs): a generic demo app, `projects/demo/bos-demo`, plus kit and framework pieces built under it: the `cms` Go package gains
a Postgres repository, HTTP handlers and migrations; `bos-go` gains a fake auth adapter and a role middleware; `bos-svelte` gains an admin shell,
header and footer chrome, two layouts (`default`, `legal`), two renderers (`hero`, `prose`) and a plain default theme; new Svelte packages
`cms/ui-cms` and `settings/ui-settings`; `ui-seo` is wired into the page editor and the public head; the infrastructure `service` schema gains
`volumes`. Six new proposals (site chrome, page designer, layout presets, demo MVP, CMS port plan, admin modules). No consumer pointer and no
`CLAUDE.md` change.

It diverges from `main` in ways its own documents admit:

| # | Main says | The PR does | Why it matters |
|---|---|---|---|
| 1 | pages are files (`site/pages/*.md`), the router is the file system (`bos-constructor.md` 4.2); M2 to M4 are static | pages, header, footer and settings are **database rows**, edited in an admin; the PR's chrome document says so explicitly | two page models for one product |
| 2 | order is M1 branding, then M2 to M4, then the dynamic phase | the demo builds admin and content first and a plain `bos-brand.css` theme; it is deliberately not the D22 token pipeline | M1 is still "next" in `CLAUDE.md`, which the PR does not update |
| 3 | D15 (where kit wiring lives) is open, lean (b) inside the kit folder, to be decided by the first dynamic milestone | `cms/http.go` and its migrations sit in the kit; the PR calls it "offered as the resolution to D15, worth confirming with the user" | a de facto decision without the user's approval |
| 4 | kits expose a manifest (D18) and `bos.yaml` plus spec keys configure them (D17) | an `AdminModule` interface in `@sbx/bos-svelte/admin`; "`main.go` itself is the configuration" | a second mechanism beside D18 |
| 5 | slots hold blocks, with compatibility checked (`bos-constructor.md` 4.3) | sections are partitioned by `kind`; no stored slot; the PR's own port plan says the layout-presets document should not be implemented as written, after it was | the PR's documents disagree with each other |
| 6 | D26: no tests while the core is built | edits `cmspage_test.go` column-count assertions | minor; not checked whether forced |
| 7 | five of the six new proposals say "not yet approved" | most are built | status labels do not match the tree |

None of this is wrong by itself. The risk is that the repo ends with two plans, and that approval status (the user's rule) is silently passed by
building first.

## 3. The outside design, in brief

Three documents, written for a SvelteKit estate with a Go API and a Python service:

- **Guidelines** (Doc 1): a single SvelteKit app done right. Browser never talks to the backend; a backend-for-frontend (BFF) with typed services,
  not a generic proxy; opaque server-side sessions in an HttpOnly cookie; no mutable module state on the server; authorization in the service layer;
  validate every boundary with a schema; rendering decided per route (prerender only what is identical for every visitor); per-request service
  container; a component registry driven by serializable screen definitions; a security and performance checklist.
- **Platform and app framework** (Doc 2): a monorepo `platform/{contracts, go, python, svelte}` with `packages <- frameworks <- apps`; a **Feature**
  interface (config, dependencies, server hooks, per-request services, client registrations) composed by `createPlatform(...)`; the consumer writes
  about ten small files; a catch-all route for CMS pages and generated stubs for fixed routes; section and content tables with section data validated
  by a JSON Schema in Go on write and again in the BFF on read; content from an API or from files behind one port; extension points as a closed
  list; **migration order: one consumer without the framework, then extract, then a second consumer, and freeze the extension points only after that**;
  "rule of three"; "the framework trap".
- **BOS extension** (Doc 3): an admin layer on the framework: a `BosModule` (nav, permissions, resources, widgets, pages); **resource-driven CRUD**
  from one `defineResource` declaration, with allow-listed resource names, sort and filter fields, mass-assignment stripping, and a fixed order
  validate, authorize, scope to tenant, call Go; Go-owned authorization and append-only audit with signed actor claims; hardened admin sessions
  and MFA; admin as a separate app when its security needs differ.

The documents tag each claim documented, pattern (the author's design) or verify. Their own "verify" list (route precedence of a catch-all, server-only
protection across workspace packages, `App.Locals` merging from a package) is not settled by the documents; each is a check we would run on our code.

## 4. Comparison, point by point

| Topic | Outside design | Ours | Verdict |
|---|---|---|---|
| Layers | `packages <- frameworks <- apps`, acyclic | `packages/<kit>/<package>` and `frameworks/`; shared kits never import feature kits; CI check of kit edges | **Same direction; ours adds kits and an enforced edge check** |
| Where consumers live | `apps/` inside the monorepo | consumers are separate repos mounted as submodules | Differs on purpose (D1). Their "consumer matrix in CI" needs a deploy-time snapshot rule: `docs/proposals/deployment-snapshot-strategy.md` |
| Cross-language contracts | `platform/contracts/` (OpenAPI, JSON Schema); a generated API client | none yet; the old app mirrors API types by hand (consumer plan section 1) | **Gap.** Our own open row "generate content types into Go and TS, or validate at run time" is the same question |
| Composition | one TS `createPlatform` with `Feature` objects | Go `Module` contract plus route groups (D18); Svelte manifest; shim routes generated from kit manifests (D17) | Different shape for the same job. Ours keeps the API in Go; theirs puts composition in the BFF |
| Backend access | typed BFF services; a generic proxy only if forwarding is the feature, then allow-listed | a same-origin `/api/*` pass-through in `bos-svelte`: forwards every path to the Go API, which "is the single API for web and mobile" (`api-pass-through.ts`) | **Deliberate and different** (see R4) |
| Sessions | opaque session id, server-side store, `__Host-` cookie | the Go `identity/auth` kit issues access and refresh cookies, `SameSite=Strict`, rotating refresh with a silent replay in the pass-through (consumer plan section 11) | Different model, ported as it runs (D25). Not evaluated against Doc 1 section 5 here |
| Pages and CMS | section and content tables, page/slot/position/status; section `data` validated by type schema in Go | `cms` package ported verbatim (kind and payload); constructor's page and block model (proposed); PR builds DB-backed pages | Compatible ideas; the PR and `main` disagree between themselves (section 2) |
| Content source | `cms({source: 'api' or 'static'})` | content type plus a source adapter: `inline`, `file`, `api`, `db`; fake for each (D21) | **Ours is the more general form of the same port** |
| Admin | `BosModule` plus resource-driven CRUD | kits own their admin pages (D15 open); PR adds `AdminModule`; no resource declaration | **Gap, deliberately late** (R2) |
| Authorization and audit | Go owns both; signed actor claims; allow-lists on every list and form | the old app has audit and roles, ported inside `identity` and `rbac`; allow-lists not stated in our documents | Likely equal in substance; not stated as a rule (R3) |
| Boundary enforcement | dependency-cruiser or eslint boundaries; a day-one canary that a `.svelte` file importing a server entry fails the build | `check_kit_edges.py` checks kit-to-kit edges only | **Partial** (R6) |
| Branding | CSS variables from presets in `branding` and `theme` packages | W3C-structured YAML tokens, three tiers, generated output never committed (D22) | Ours is the heavier design; justified by the parity gate on computed values, not by the outside text |
| Assets | not covered | stable and hashed URL classes, derived favicon set, self-hosted fonts (D23) | Ours only |
| Parity with an existing app | not covered (greenfield) | five parity levels against the running original (D20) | Ours only; it is what makes a port safe |
| Migration order | one consumer first, extract, second consumer, then freeze | strangler replaced by spine first with a baseline (D5); the second consumer is M4b | Compatible; theirs is explicit that extension points freeze late, ours does not say so |
| Risks named | framework trap, rule of three, config becomes a language, CMS scope creep | risks table in the roadmap; "port only what the app uses" | Theirs names the **framework trap** more bluntly (R8) |
| Stack versions | profile: SvelteKit 2 or 3, remote functions optional | `bos-svelte` peers on SvelteKit `^2.0.0`; the consumer pins 2.50.2 | State the target major once (R9) |

## 5. My version: recommendations

Each has its evidence, its cost, and what I would not do. None is decided.

**R1. Settle the fork between `main` and PR #2 before M1 starts. (the one that matters)**
Evidence: section 2, rows 1 to 4; the consumer plan section 14 records that the original already stores seven pages in a database table *and* has
them as Svelte files, so "some pages exist twice" and the oracle must show which copy wins. A file-first static phase would rebuild that split.
My lean: the **page** is a content type whose source for the port is `db` (the PR's direction), with files kept as the **seed** (D17 already says this:
files seed a fresh environment, editors win afterwards) and as the source for consumers that have no database. That keeps the roadmap's parity gates
and its token and asset work, changes only which source M2 and M3 use first, and does not discard the PR. Options for the user:
(a) keep `main`'s order and merge the PR later as the dynamic phase's start; (b) adopt my lean and re-order M2 to M5 around the `db` source;
(c) merge the PR as a demo only and keep both plans. I recommend (b), with the PR's status labels, D15 and `CLAUDE.md` "Where things stand" updated in the same step.
Cost: one design scope to rewrite the affected roadmap rows. Would not do: merge the PR first and reconcile later.

**R2. Adopt the resource declaration for the admin phase, but only when the second admin entity needs it.**
The outside `defineResource` (one declaration gives list, detail, form, delete, actions, nav and permissions, with allow-lists) answers a measured
problem: the consumer plan section 1 records heavy parametric repetition in admin routes. Our rule is "port only what is used" and the outside
document's own "rule of three" says the same. So: write it at M6 from the first two real admin entities of the original, not now, and make the PR's
`AdminModule` the container it plugs into. Not adopting now keeps this inside "nothing speculative".

**R3. Write the admin security rules into the kit contract once, from the original's behaviour.**
Outside rules worth stating for every `bos-go` route group: allow-listed sort and filter fields, a page-size cap, mass-assignment stripping, validate then
authorize then scope then call, audit for every mutation. The old app already has audit and roles; the work is to record which of these it already
enforces (read its handlers, do not assume) and make the route-group declaration (D18) carry them. Evidence needed first: read of the original's
admin handlers. Not read.

**R4. Keep the `/api/*` pass-through, but record it as a deliberate decision and add three settings.**
It is a generic proxy, which the outside Doc 1 section 4.2 permits only when forwarding is the feature and then wants an allow-list. Our reasons are
real (one Go API for web and mobile; the golden files fix its behaviour; moved, not rewritten, D25), so I would not replace it with typed BFF services.
Gaps: no timeout (already a known gotcha in the handoff), it forwards every response header, and there is no path allow-list. Cheapest fix: a timeout and an
outbound header strip as settings; an allow-list as a spec key only if the user wants one. Verify against the golden files after any change.

**R5. Decide the contract story at M3, not before.**
The open row "content types generated into Go and TS, or validated at run time" and the outside "contracts" folder are the same decision. I would pick
**generate TS types from the content-type schemas** (small, file-based, offline) at M3, and defer an OpenAPI-generated client until the route table exists
(the oracle's step 4 needs a `buildRouter` that the original does not have). Evidence for deferring: building OpenAPI for about 258 hand-registered
routes (consumer plan section 9) is a milestone of its own.

**R6. Extend the boundary check to the two rules it does not cover.**
(1) a package never imports a framework; (2) client code never imports a server entry, plus the outside document's day-one canary (a throwaway
`.svelte` file importing a server entry must fail the build). First step is to read `check_kit_edges.py`'s rules, since I only verified that it checks
kit edges. Small, and it protects `bos-svelte`'s `server/` entry.

**R7. Write the framework-trap rules into `bos-svelte`'s README at M2.**
Adopt: a closed list of documented extension points; override by path (D21 already); an "eject" path per feature (the underlying package stays
importable); extension points freeze only after the second consumer (M4b). We already have two of the four; the other two cost a paragraph.

**R8. State the SvelteKit target once.**
`bos-svelte` peers on SvelteKit `^2.0.0`, the consumer pins 2.50.2 with Vite `^5.0.0`. The outside Guidelines list what changes in v3 (config moves into the Vite
plugin, `$lib` becomes `#lib`, env modules, `$app/stores` removed). Record "v2 now; v3 is its own scope" in the framework README so no scope drifts onto v3 by accident.

**R9. Add a cache and cookie rule per route class to the M2 gate.**
The outside design's point: public pages must not set cookies or vary on session, or a CDN cannot cache them. Add to the M2 and M3 gates a check that
public static pages answer without `Set-Cookie` and with an explicit cache header, measured on the old app first (not measured here).

**What I would not adopt**: a TypeScript `createPlatform` with `Feature` objects as the composition root (ours composes through Go modules and kit
manifests; two roots would conflict with D18); SvelteKit-held sessions in Redis (Go owns authentication here and it is ported as it runs); a repo-top
`contracts/` folder now (R5); Valibot as a mandated library (nothing in our packages uses it; check before choosing a validator).

**Where the outside design is weaker than ours**: it has no parity method for a port, nothing on assets, fonts or tokens, and its dependency rules
are conventions to lint where ours has a CI check; it also assumes consumers live inside the monorepo.

## 6. A port path, in my order (for approval, not started)

1. Decide R1 (one decision, one scope, docs only).
2. Land PR #2 after its labels, D15 and `CLAUDE.md` are corrected; or hold it, per the user's choice.
3. M1 branding (D22 tokens; the PR's `bos-brand.css` stays a demo default and is not the token pipeline).
4. M2 to M4 pages and chrome on the source decided in R1; gates P-TXT, P-PIX, P-TOK as in the roadmap, plus R9.
5. M5 database and runtime site configuration; M6 identity and admin with R2 and R3; the original's authentication defect is fixed in the original first
   (design section 7).
6. M7 to M9 SEO, media, other kits, real-estate; M10 production bundle: its own plan, and the snapshot rule in
   `deployment-snapshot-strategy.md` is its input.

## 7. Questions for the user

1. R1: option (a), (b) or (c) for the fork between `main` and PR #2?
2. Is D15 resolved as the PR offers (kit wiring inside the kit folder)? If yes it should be recorded with a date.
3. R4: keep the generic pass-through with a timeout and a header strip, or move to typed BFF services?
4. Which of R2 to R9 should become scopes, and in what order? I would start with R1 and R6.
