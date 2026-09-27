# bos-demo — layout presets: slots a page actually uses, not just "main"

Last updated: 2026-09-28
tags: bos, bos-demo, layout, slots, presets, admin, proposal

Status: **Scope L1–L4 built and verified**, with one correction found mid-build (see below). Builds
the piece `docs/proposals/bos-page-designer.md` §3 designed but explicitly deferred ("the offline
design pass you described"). Nothing here contradicts `bos-constructor.md` §4.1–4.5 (layout presets,
slots, resolution order) — it is that model, built the DB-native way `bos-demo` already committed to,
not the file-based `site/pages/**` way the original roadmap's M2 assumed.

**Correction found while building this scope:** §2/§3 below assumed `page_sections.slot` (the old
flat `bos-demo` `pages`/`page_sections` tables) was a real, unused column just waiting to be read. It
no longer exists — those tables were dropped in migration `005_drop_pages_and_sections` when the
`cms` kit replaced them (`docs/proposals/bos-cms-port-plan.md`), and the new `cms.Section` envelope
(`platform/go/packages/cms/section.go`) has `Kind`/`Payload`/`Hidden`, no slot field at all. Rebuilding
a slot column would duplicate what `Kind` already signals. What was actually built: each layout
component partitions the `sections: SectionData[]` it's given **by `kind`** — the `default` layout
renders the first `hero`-kind section(s) in its hero region and everything else in main; `legal`
renders only main and structurally excludes any `hero`-kind section even if one is present in the
data (verified: a `terms` test page with a `hero` section forced into its `content` renders no hero
region — the hero payload is present only in the client hydration JSON, never in the DOM). One new
column was still needed — `cms_pages.layout` (which registered preset a page uses) — added via
`platform/go/packages/cms/migrations/20260928000000_add_page_layout.{up,down}.sql`, since the
create-table migration already ran against the live dev DB and can't be edited in place.

## 1. Why this document exists

You asked for pages to carry a real layout — "for about, set whether it uses a hero section, and
what kind" — as the mechanism underneath a bigger content-model ask (page types, scopes, imported BR
content). Layout presets are the one piece all of that depends on, and it's smaller than it looks:
`page_sections` already has a `slot` column (built with the sections table itself, D4), but nothing
today ever uses a slot other than `"main"` — the admin's Content tab hardcodes `const slot = 'main'`,
and the public route renders every section through one flat `<SectionList>` with no slot separation
at all. This document is that gap, closed.

## 2. What already exists — checked directly

- `page_sections.slot` (`text`, default `'main'`) — real column, unused beyond its default.
- The renderer registry (`bos-svelte/src/lib/renderers`) — `hero` and `prose` renderers already
  exist and already work; a layout doesn't need to invent renderer logic, only decide **where each
  slot's blocks render on the page**.
- The chrome registry (`bos-svelte/src/lib/chrome`) — the exact mechanism this document reuses for
  layouts: `import.meta.glob`, one folder per preset, no hand-written switch.
- The public route today (`(site)/+page.svelte`, `(site)/[slug]/+page.svelte`): a hardcoded
  `<h1>{page.title}</h1>` followed by `<SectionList sections={data.page.sections} />` with no layout
  or slot awareness — a placeholder from before renderers/sections existed, and the thing this scope
  replaces.

## 3. The model — layout preset = a component that knows its own slots

Per `bos-constructor.md` §4.1: a layout preset is "a named skeleton with slots... code (a component
and its metadata)." Concretely, mirroring the chrome registry exactly:

```
bos-svelte/src/lib/layouts/
  types.ts        # RegisteredLayout { id, label, slots: string[], component }
  registry.ts     # import.meta.glob('./*/index.ts', { eager: true })
  index.ts        # barrel
  default/
    Layout.svelte # renders a hero region (if the hero slot has blocks) above a main region
    index.ts
  legal/
    Layout.svelte # main region only — no hero slot exists for this layout at all
    index.ts
```

Only the two presets actually needed right now: **`default`** (slots `hero`, `main` — for the
homepage and `about`) and **`legal`** (slot `main` only — for terms/privacy/cookies). `landing` and
`minimal`, named in the original roadmap, are easy to add later the same way (one more folder) — not
built speculatively now.

A layout component takes the page's full `sections: SectionData[]` and partitions them by `slot`
itself (`sections.filter(s => s.slot === 'hero')`, sorted by `position`) — the same filter-and-sort
logic already in the admin's Content tab, just per-slot instead of hardcoded to `'main'`. Because
`legal`'s component only ever reads its `main` slot, a legal page **cannot visually produce a hero**
no matter what's in the data — a real instance of bos-constructor's "a layout refuses a block in a
slot that does not allow it," achieved structurally rather than by a separate compatibility check
(that check, for a slot accepting only certain renderer *kinds*, is a further refinement — not needed
for what's asked now: whether the hero slot exists at all, which this already handles).

**Public rendering changes**: the page's own `<h1>{title}</h1>` placeholder goes away — content
should declare its headline through a block (a hero's `headline` field, or a prose section's own
text), matching "code declares, data selects" rather than the route template inventing content. The
page's `title` still sets `<svelte:head><title>`, that never changed. Concretely this means the home
page (currently one plain `prose` section, no headline field) needs a hero block added as part of
this scope so it doesn't visually go blank — a one-time content fix, not a mechanism change. An
unregistered/unknown `layout` value falls open to `default`, the same fails-open rule already used
for an unknown renderer id.

## 4. Admin UX

- **Settings tab** (`/admin/pages/[id]/settings`, already built): gains a `layout` `<select>` sourced
  from the layout registry, saved through the existing `/meta` endpoint (extended to accept `layout`
  alongside slug/title — still one page-configuration concept, not a new endpoint).
- **Content tab**: instead of one hardcoded "Sections — main" list, shows one list **per slot the
  page's current layout defines** (`default` → "Hero" then "Main"; `legal` → just "Main"). Add/
  remove/reorder work exactly as today, scoped to whichever slot's list the button belongs to. No
  renderer-to-slot restriction yet (any renderer can go in any slot the layout offers) — deferred,
  per §3.

## 5. API

- Migration: `ALTER TABLE pages ADD COLUMN layout text NOT NULL DEFAULT 'default';` — existing pages
  (home, features, contacts) all get `default` and keep rendering exactly as before (their sections
  are all in `main`; the `hero` region simply renders nothing when empty).
- `Page.Layout string` added to the struct; `PATCH /pages/{id}/meta` accepts `layout` alongside the
  existing `slug`/`title`. No server-side validation of the layout id's value — it's data selecting
  among code-declared web-side presets (§3's fails-open rule handles an unrecognized one), the same
  trust boundary already accepted for a section's `renderer` id.

## 6. Scope breakdown

| Scope | What | Gate |
|---|---|---|
| L1 | Migration + `Page.Layout` + `/meta` accepts `layout` | `go build`/`go vet` clean; a `PATCH .../meta` round-trips a layout change |
| L2 | `bos-svelte/src/lib/layouts/{types,registry,index}.ts`, `default/` and `legal/` presets, `"./layouts"` package export | `svelte-check` clean |
| L3 | Public routes resolve layout via the registry instead of the hardcoded `<h1>+SectionList`; home page gets a hero block | SSR-confirmed: home shows a real hero region above its prose main content; a `legal`-layout test page shows no hero region even if a hero-slot section were forced into its data |
| L4 | Admin Settings gets the layout picker; admin Content tab becomes slot-aware (one list per slot) | SSR-confirmed: switching a page to `legal` shows only a "Main" section list; switching to `default` shows "Hero" and "Main"; adding a hero block through the admin appears in the public hero region |

## 7. What this document does not decide

- Renderer-to-slot compatibility (e.g. refusing a `prose` block in a slot meant for `hero`-shaped
  content) — `bos-constructor.md` §4.3 names this as the real gap the earlier bos had; worth its own
  pass once more than two layouts/renderers exist to make the restriction meaningful.
- `landing`/`minimal` presets — added the same way, when a real page needs one.
- The page-type/scope classifier (homepage-is-a-singleton, "pick Legal → Terms") from the prior
  conversation's conclusion — a separate, smaller scope that builds on top of this one, not part of it.
- Visual design of `default`'s hero region or `legal`'s narrower main column — minimal, functional
  styling only, same as every other admin/public page built so far.
