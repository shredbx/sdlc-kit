# bos — admin modules: a kit's own nav entry, the pages list regression, a bos default theme

Last updated: 2026-09-28
tags: bos, bos-demo, admin, nav, kit-protocol, theme, proposal

Status: **proposal, not yet approved.** Extends `docs/proposals/bos-page-designer.md` (the `AdminShell`,
already built) and `docs/proposals/bos-site-chrome.md` (the settings tabs, already built) with the piece
neither one built yet: how an enabled kit gets *itself* onto the admin sidebar, instead of `bos-demo`
hand-typing every entry. Checked against the real code (`AdminShell.svelte`, `admin/+layout.svelte`,
`bos.yaml`, the Svelte `packages/` tree, the cms kit's Go side) before writing anything below, not from
memory of the design docs alone.

## 1. Why this document exists

You flagged three things together: the admin pages list disappeared (you can still create a page by
typing its slug, but there's no way to see what already exists); you want a "bos default theme" so
components look designed rather than bare; and, from watching how the real bestierealestate admin is
built, you want each enabled package (Pages, Settings, and later Contacts, Address book, Real estate) to
declare its own sidebar entry and its own list/configure capability — a "cms protocol" — with
`bos-demo`'s own `bos.yaml`-style configuration narrowing which ones actually show up, the same way
"framework carries everything, configuration narrows" already works for infra.

Checked first: this isn't new architecture to invent from nothing. `bos-system-design.md`'s decision
log already has **D18**, still only sketched: *"A Svelte manifest declares routes, admin navigation
entries, surfaces ... and block renderers."* A kit declaring its own admin surface is that sentence,
made concrete. It's also, precisely, `bos-app-roadmap.md`'s own **M6** milestone ("Identity and the
admin shell ... the admin shell and its navigation") — scheduled deliberately ahead of **M8** (7 more
kits: contacts, calendar, news, documents) and **M9** (real-estate, "the largest surface") for exactly
this reason. See section 3 for the full target list, read directly from the real app's own nav tree.

## 2. What already exists — checked directly, not assumed

- **`AdminShell`** (`bos-svelte/src/lib/admin/AdminShell.svelte`) is real and working: it calls
  `createTopbarSlots()`, renders `core-ui`'s `SidebarLayout`, and filters a `navItems: AdminNavItem[]`
  array by `role`. It does not generate that array — it's handed one.
- **`bos-demo/apps/web/.../admin/+layout.svelte`** hand-types that array today:
  `[{href:'/admin/pages', label:'Pages'}, {href:'/admin/settings', label:'Site configuration',
  roles:['admin']}]`. Two entries, two admin surfaces, both baked into the consumer's own route file —
  this is the exact "consumer carries logic that should live in the framework/kit" shape you flagged
  earlier this session for the cms port, now showing up again one layer up, in the nav.
- **The glob-registry pattern is already established three times** and proven working:
  `renderers/{kind}/index.ts` (a section renderer), `chrome/{header,footer}/{preset}/index.ts` (a chrome
  preset), and the not-yet-read `layouts/` proposal's same shape for layout presets — each a folder,
  each `import.meta.glob`'d, no hand-written switch, no generated file. Whatever an admin-module
  manifest looks like, it should be the same shape, not a fourth different mechanism.
- **`bos.yaml` has no enabled-kit list of any kind yet** — it only has `prefix`/`site_name`/
  `environment`/`api`/`web`. "Configuration narrows which packages show up" has no field to narrow from
  today.
- **Pages and Settings are not portable kit packages** — they live directly inside `bos-demo`'s own
  `apps/web/src/routes/admin/{pages,settings}/`, unlike `contacts`, `calendar`, `seo`, which are real
  Svelte packages under `platform/svelte/packages/<kit>/`. A kit can only export its own admin manifest
  if it's a kit — an app route can't export anything to be discovered.
- **The cms kit's Go side has no list endpoint** — `GET /admin/cms/{slug}` and `PUT` only. This is the
  literal cause of "pages disappeared": the old flat `pages` model this replaced (deleted this session)
  never had a list endpoint either, but its admin route still had a hardcoded list of known slugs to
  choose from; the cms port's replacement admin route (`admin/pages/+page.svelte`, my own stand-in from
  this session) has no such list, and no API to build one from. This is a **regression**, not a
  preexisting gap — the real bestierealestate app doesn't need this because its pages are fixed by a
  `SITE_SURFACES` registry; `bos-demo`'s dynamic page model has no equivalent, so it must have a real
  list endpoint.

## 3. The real target — this is not a hypothetical future case

An earlier draft of this document cited `bos-demo-mvp.md` §7's "rule of three" (don't generalize until a
third case exists) as a reason to hesitate. That was a mistake, corrected here rather than silently: this
whole track is **decomposing a specific, fully built, currently running production app** — the target
admin surface already exists, in full, on disk. Read directly from
`bestierealestate/apps/web-svelte/src/lib/nav.ts` (the real admin's own nav tree, not a guess), the real
target has at least ten distinct content-managed areas beyond Pages/Settings: Contact Book, Calendar,
Documents, Inquiries, Properties, FAQ/Guides/Services, News, Media Canvas/Watermarks, and user/system
Admin. `bos-app-roadmap.md` already schedules them, by name, across milestones already written down —
**M6** "Identity and the admin shell" (*"the admin shell and its navigation, the page editor"*), **M7**
media, **M8** "The other kits" (contacts, calendar, news, documents — 7 scopes), **M9** "Real-estate"
(properties, transactions, collections, agents — "the largest surface"). M6 sits deliberately *before*
M8/M9 specifically because a dozen more kits are already known to be coming.

"Rule of three" guards against inventing an abstraction for a case nobody has asked for yet. That does
not apply here — the cases are enumerated, named, and scheduled. Building the nav/capability manifest now
**is** M6, on schedule, not scope built ahead of need. This document commits to it directly (section 6)
instead of offering it as one of two options.

## 4. The immediate regression fix — needed either way

Independent of section 3's decision, "pages disappeared" is a bug against this session's own prior work
and should be fixed regardless of how the nav protocol question resolves:

- **Go**: `Repository.List(ctx) ([]CmsPage, error)` in `platform/go/packages/cms/postgres.go` (plain
  `SELECT` over non-deleted rows, no pagination yet — nothing in `bos-demo` has enough pages to need
  it). `GET /admin/cms` in `http.go`, registered in `RegisterAdmin` alongside the existing two routes.
  Returns slug/title/published/updated_at per row — enough for a list, not full content.
- **Svelte**: `admin/pages/+page.svelte` replaces the slug-typing form with a real table (slug, title,
  published badge, updated_at) sourced from a new `+page.server.ts` calling the new endpoint, each row
  linking to `/admin/pages/{slug}`; the "create new" affordance becomes a small "new page" action that
  still asks for a slug (creation-by-slug is a real, working mechanism from this session, not being
  replaced) but is no longer the *only* way to reach a page.

## 5. The bos default theme — reusing what already works, not D22's pipeline

Checked: there are two token mechanisms in this repo, not one, and they are not the same job.
**D22** (`bos-system-design.md` line 205, `bos-app-roadmap.md` M1) is a W3C-structured-YAML pipeline
whose entire point is **pixel-parity import of the real bestierealestate brand** — a Style Dictionary
spike, a one-shot importer reading the *old app's own computed CSS*, gates like "every color equals the
old app's computed value." That machinery is for the real client migration, and `bos-demo-mvp.md` §7
already deferred it explicitly: *"Branding/tokens (roadmap M1) — bos-demo runs on framework defaults
until this is picked up separately."*

What you're asking for now is smaller and different: `bos-demo` (and any other generic consumer) should
not look bare. `@sbx/core-ui`'s `IStylePreset`/`styleStore`/brand-CSS system (`style/IStylePreset.ts`,
`style/brands/{hub,shredbx,promptstudio}-brand.css`) is **already built, already proven** (three real
brands work today via `[data-brand="..."]` CSS custom properties) and completely unused by `bos-demo`.
Per D25 (reuse before writing), the right move is a fourth brand, not a new pipeline:

- `bos-svelte/src/lib/theme/bos-brand.css` — one new `[data-brand="bos"]` block, values chosen freely
  (this is a generic default, not a port of anything), covering the same token set `shredbx-brand.css`
  already defines (color ramp, type scale, spacing, radius, shadow, motion) so every `core-ui` component
  that reads those custom properties picks it up with no component changes.
  A light + dark preset registered in `stylePresetRegistry`, mirroring how `hub`/`shredbx` register
  theirs.
- `bos-demo`'s root layout sets `data-brand="bos"` (and reads/writes the mode via the existing
  `styleStore`) — one attribute, no new mechanism.
- This is framework work (`bos-svelte`/`core-ui`), consumer-neutral: any future generic consumer gets a
  presentable default for free, same as `bos-go`'s settings defaults today.

D22's real pipeline stays exactly as scoped — for the real bestierealestate port, later, pixel-matched
against the actual production CSS. This section does not touch it or substitute for it.

## 6. The nav/capability protocol — building it now, this is M6

Extract Pages and Settings out of `bos-demo`'s app routes into real kit packages
(`platform/svelte/packages/cms/ui-cms`, `platform/svelte/packages/settings/ui-settings` — mirroring
`contacts/ui-contact`'s shape), each exporting a `registeredAdminModule`:

```ts
interface AdminModule {
  id: string;                 // 'cms', 'settings'
  navItem: { label: string; href: string; icon?: string; roles?: string[] };
  // deferred until a module needs more than a nav entry (§8) — no list/configure
  // component slot yet, since AdminShell doesn't render one inline anywhere today.
}
```

The type lives in `bos-svelte` (`@sbx/bos-svelte/admin`), imported by each kit as a type-only import
(erased at build, so it carries no runtime coupling) — the "contract extracted as a tiny leaf package"
idea D15 already named, minimized to a type instead of a whole package since one interface doesn't
justify a new package yet.

Discovery does **not** mirror `renderers`/`chrome`'s `import.meta.glob`: per section 2's finding, a
kit's admin package is a separate installed package, not a file inside `bos-svelte`'s own folder, so the
glob can't reach it. Checked against the Go side before designing this: `bos-demo`'s own `main.go`
doesn't read `bos.yaml` to decide which kits to mount either — it just calls each enabled kit's
`RegisterAdmin` directly; **`main.go` itself is the configuration**. The Svelte side mirrors that
exactly rather than inventing a parallel YAML-driven mechanism nothing else in this repo uses yet:
`bos-demo`'s own `admin/+layout.svelte` (the app's composition root for the admin) imports each enabled
kit's `registeredAdminModule` and builds `navItems` from them — a plain array of imports, not a
`bos.yaml: kits:` field. "Code declares (the kit), the app's own composition root decides what's
enabled (the layout file, exactly like `main.go`), the framework renders it (`AdminShell`)."

`cms` and `settings` become the **reference implementation** of the protocol — not a two-case guess,
because the shape is being checked against all ten-plus §3 targets while it's designed (each needs at
minimum a `navItem`; several — Contacts, Properties, News — will need `list`; Settings-style
configuration screens need `configure`), even though only two are actually built yet. The next kit that
lands (per M8's own order, likely Contacts) either fits the shape or reveals what's missing — that's
normal for a first real implementation, not a sign it was premature.

## 7. Scope breakdown

| Scope | What | Gate | Depends on |
|---|---|---|---|
| N1 | `Repository.List` + `GET /admin/cms` (§4) | `go build`/`go vet` clean; curl returns all non-deleted pages | — |
| N2 | Real Pages list UI in `bos-demo` (§4) | Visiting `/admin/pages` shows every page created this session; clicking one opens its editor | N1 |
| T1 | `bos-brand.css` in `bos-svelte/src/lib/theme/` (full token set + light/dark/high-contrast variants + a base `body` rule, since core-ui ships no reset of its own) | `svelte-check` clean; SSR HTML includes the token values | — |
| T2 | `bos-demo` root layout imports the CSS; `app.html` sets `data-brand="bos" data-theme="dark"` statically | SSR HTML shows `data-brand="bos"` on `<html>`; the CSS's tokens appear in the rendered page | T1 |
| P1 | `cms`/`settings` extracted to real kit packages (`@sbx/ui-cms`, `@sbx/ui-settings`), each exporting UI components + a `registeredAdminModule`; `AdminModule` type added to `@sbx/bos-svelte/admin` | packages resolve and typecheck through `bos-demo`'s own `svelte-check` | — |
| P2 | `bos-demo`'s admin layout composes `navItems` from the enabled modules it imports, replacing the hand-typed array | Removing a module import makes its sidebar entry disappear without touching `AdminShell` | P1 |

**Correction found while building T1/T2**: registering a `bos` preset in `core-ui`'s
`stylePresetRegistry` (as originally planned here) turned out to be unsafe to wire up yet — importing
that module pulls in `styleStore`'s auto-init side effect, which defaults to its own hardcoded
`'hub-dark'` preset on every client hydration and would silently overwrite `data-brand="bos"` the
moment anything imports the registry. T1/T2 ship as a plain CSS default (SSR-set attribute + token
values) instead, with no dependency on `styleStore`. Runtime brand/mode switching through the registry
is real future work, not done here — revisit when `bos-demo` actually wants a switcher UI.

All four lines (N, T, P) are independent of each other and can be done in any order, or together —
sequencing below is a suggestion, not a dependency chain except where noted.

## 8. What this document does not decide

- **Per-content-type editing layouts with subtabs** (`RailsContentLayout` + a tab-builder, the way the
  real app shares one layout component across Properties and Pages with per-type tab data) — this is
  real, scheduled work (M8/M9), not deferred on principle. It waits because M8/M9 haven't started: the
  tab shape needs Contacts' or Properties' actual fields to design against, which don't exist as bos
  code yet. Picked up when M8 begins, not gated on an arbitrary case count.
- **A settings screen for arranging the sidebar itself** (drag-reorder, hide/show per admin) — depends
  on section 6's modules existing first (there's nothing to arrange until modules are data, not
  hand-typed code); not designed here.
- **`list`/`configure` capability details beyond the nav entry** (e.g. what exactly a package's
  `configure` screen is allowed to assume about its host) — deferred to when a second package
  (beyond Settings) actually implements one.
- D22's real W3C token pipeline — untouched, still scoped for the eventual bestierealestate M1 port.
