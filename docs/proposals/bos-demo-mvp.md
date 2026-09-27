# bos-demo — MVP 0.0.1: prove the constructor generically before configuring one client

Last updated: 2026-09-27
tags: bos, bos-demo, mvp, admin, persistence, proposal

Status: **proposal, not yet approved.** Parent: `docs/proposals/{bos-system-design,bos-constructor,bos-app-roadmap}.md`
(decisions D1–D26). This document does not replace the roadmap's M0–M10 milestones — it inserts a new,
separate build target ahead of them: a generic reference build that proves the frameworks and the admin/content
loop, before `bestierealestate` (or any other client) is configured on top of the result.

## 1. Why this exists

The `bos` track had been building milestone-by-milestone directly inside the `bestierealestate` consumer repo
(M0 landed there: `bos-go`/`bos-svelte` linked, the placeholder page, the app bundle). The user's direction, given
2026-09-27, changes the build target: **build `bos` the product, and its own demo, first** — proving real-estate-kit
functionality and package integration generically — and only later run a `bos new`-style bootstrap process to
generate a real client instance, at which point a client repo (`bestierealestate`, `bestays`, …) becomes a
configuration/finishing pass on generated output rather than something built by hand scope-by-scope.

This reprioritizes the riskiest, most novel part of the constructor (admin + dynamic content management) ahead of
branding polish (roadmap M1) and ahead of the static-pages sequence (M2–M4), on the reasoning that the admin/content
loop is where the earlier `bos` attempt (`sbx.framework/projects/bos`) actually failed (see `bos-constructor.md` §2)
and is worth retiring early, even unbranded.

## 2. Where it lives

`bos-system-design.md` D1 says every **client** consumer is its own repo, submoduled under
`consumers/clients/<client>/<project>` — that governs deployment and repo arrangement for actual clients and is
unaffected by this document. A demo is not a client. It lives at **`projects/demo/bos-demo/`**, a new sibling to
`projects/products/` and `projects/services/` (confirmed: this path does not exist yet anywhere in this repo's
history — checked `git log --all -- projects/demo`, the `.gitignore`, and the working tree. It is being created
fresh, not resumed).

`bos-demo` is shaped like a thin consumer (`bos.yaml`, `apps/api`, `apps/web`, `deploy/`) because it has to actually
exercise `bos-go`/`bos-svelte` the way a real consumer would, linked by local directory exactly as
`bestierealestate` is (D24) — it is just not a submodule and not client work.

## 3. Persistence: database now, and a real gap this surfaces

Decision (2026-09-27, this document): **Pages and Settings are Postgres-backed from the start**, not file-sourced.
This pulls part of roadmap M5 forward deliberately — accepted, because the point of `bos-demo` is to prove the
dynamic admin/content loop, not to re-litigate the static-first ordering that still applies to a real client build.

Datastore shape: **new sibling service records**, matching the repo's own existing precedent
(`chat-api-postgres.yaml` is already its own service, separate from `postgres-dev`, for the same reason) rather than
adding a second database inside the shared `postgres-dev` container:

- `bos-postgres` (`kind: datastore`, `image: postgres:16`, its own port)
- `bos-redis` (`kind: datastore`, `image: redis:7`, its own port)
- a `bos-dev` service-bundle joining them (and, once built, `bos-api`/`bos-web`)

**Gap found while planning this:** the `bundle-compose` template has no `volumes:` support at all today — not for
`postgres-dev`, not for `chat-api-postgres`. Neither currently survives a `docker compose down`. The user wants
`bos-demo`'s data to survive a rebuild (a gitignored local bind-mount), which needs a small, genuinely-justified
schema extension, not a workaround:

- new type `infrastructure/type/volume-list.yaml` — `sequence` of `"host:container"` strings, exactly mirroring
  the existing `port-list` type
- `service.yaml` gains one optional field: `volumes: {type: volume-list, required: false}`
- `bundle-compose/files/docker-compose.yml.jinja` gains one `{% if s.volumes %}` block, mirroring the existing
  `ports` block

This is mechanism 1 of the four allowed extension mechanisms (extend the schema directly) — small, and reused by
any future service that needs it (not `bos-demo`-specific).

## 4. Auth: a fake adapter behind the real interface, not a bypass hack

The already-ported `identity/auth` package (`platform/go/packages/identity/auth`) has everything needed: `Claims{Role
string}`, and `AuthExtract`/`RequireAuth` middleware that take an `AuthService` interface. No dev-bypass exists
today, and none is needed — a `FakeAuthService` implementing that same interface (mirroring the "fake adapter
chosen by configuration" pattern already used for `agent-framework` and named in `bos-constructor.md` §3) returns
fixed `&Claims{Role: "admin"}` regardless of input. `RequireAuth` and any future permission check run unmodified
against a real `Role` string. Swapping in the real Postgres-backed `AuthService` at a later milestone touches only
which implementation is wired in `bos.yaml` — nothing in the admin routes changes. This is what "preserve RBAC from
the start" means concretely: the seam exists now, real auth drops in later.

## 5. MVP 0.0.1 feature scope

- **Homepage.** One real page (content-type `page`, DB-sourced), rendered by a minimal loader — title + body only.
  No blocks, renderers, layout presets, or content-type/source selection yet (explicitly deferred, per the user).
- **Admin, two tabs — Pages and Settings** — built on the already-ported `core-ui` primitives
  (`TabbedPageShell.svelte`, `Tabs.svelte`, `Modal.svelte`; `platform/svelte/packages/ui/core-ui/src/lib/components/`),
  not new component work:
  - **Pages tab:** list, create, delete, enable/disable. No preset selector, no data-source picker yet (deferred).
  - **Settings tab:** a minimal singleton (site title/tagline to start — deliberately smaller than the old app's
    full `site_config` shape; nav/footer editing waits for when header/footer actually exist as renderable regions,
    roadmap M2).
- **Auth:** gated by `RequireAuth` + the `FakeAuthService` (section 4) — always authenticated as `admin`, real login
  deferred.

## 6. Scopes, in dependency order

| # | Scope | What | Gate |
|---|---|---|---|
| S1 | `volume-list` type + `service.volumes` field + `bundle-compose` template block | schema/template only, no records yet | `process-cli check` clean; a scratch record with `volumes` renders a compose file with the bind mount |
| S2 | `bos-postgres`, `bos-redis` service records; `bos-dev` service-bundle; `projects/demo/bos-demo/deploy/` rendered and run | `bootstrap-bundle` | both containers healthy; `data/postgres/`, `data/redis/` populated and gitignored; `down` + `up` preserves data |
| S3 | `bos-demo` app skeleton: `bos.yaml`, `apps/api` (bos-go), `apps/web` (bos-svelte) | replays M0's own recipe here instead of in `bestierealestate` | `make dev` shows the placeholder page, API healthy — same shape as M0's original gate |
| S4 | `FakeAuthService` in `bos-go`, selected by `bos.yaml` config; `RequireAuth` wraps `/admin` and its API | new adapter behind the existing interface | an unauthenticated request to a public route works; the admin route always resolves as `admin`-role |
| S5 | Pages domain: Postgres table, Go repository, admin API (list/create/delete/toggle) | new — earlier than roadmap M5, deliberately | CRUD round-trips against `bos-postgres` |
| S6 | Settings domain: Postgres singleton row, Go repository, admin API (get/update) | new | round-trips against `bos-postgres` |
| S7 | `bos-svelte` admin shell: `/admin` route, `TabbedPageShell` with Pages/Settings tabs, wired to S5/S6 | reuses ported `core-ui` components | create/delete/enable/disable a page from the browser; edit a setting |
| S8 | The homepage itself: one DB-sourced `page` entry rendered by a minimal loader | ties S3+S5 together | visiting `/` shows the page created in the admin |

Each scope still gets its own before/after tree and its own approval before being written, per this repo's
workflow — this table is the sequence, not a standing authorization to build all eight.

## 7. Deferred, explicitly

- Layout presets, block/renderer/source selection for Pages (roadmap M2–M4's territory)
- Real authentication (magic link / password, roadmap M6)
- Branding/tokens (roadmap M1) — `bos-demo` runs on framework defaults until this is picked up separately
- A generic schema-driven `/admin/[kind]` pattern (seen as prior art in `sbx.framework`, `docs/research/sbx-framework-inventory.md`
  candidate addition) — worth revisiting once a third admin-manageable kind exists (rule of three), not built for
  two kinds
- The `bos new` bootstrap process that later generates a real client from this proven reference (mentioned in
  roadmap M4 as "the real `bos new`") — out of scope until `bos-demo` itself is proven
