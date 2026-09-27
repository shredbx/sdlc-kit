# bos-demo — site chrome: header/footer, app-level config, page content untouched

Last updated: 2026-09-27
tags: bos, bos-demo, chrome, header, footer, settings, admin, proposal

Status: **proposal, not yet approved.** Extends `docs/proposals/bos-page-designer.md` (the admin shell,
`Modal`-based editors, and the renderer-registry mechanism — nothing here contradicts it) with the one
piece it explicitly deferred to later (§6: "Whether `AdminShell` also needs ... chrome — out of scope").
Checked against the real code (`bos-demo`'s current `settings` package and routes, `bos-svelte`'s renderer
registry, the roadmap and consumer-plan docs) before writing anything below.

## 1. Why this document exists

Every `bos-demo` page today renders bare — no header, no footer, `apps/web/src/routes/` has no root
`+layout.svelte` at all (confirmed: none exists). You asked for the next scope to add the topbar and
footer BR uses, plus an app-level admin surface to enable/disable each one and pick its "preset," with
pages themselves resolving that as a default and only customizing their own content area — per-page
chrome override deferred until actually needed, but the architecture should allow it later without a
redesign.

This is exactly the roadmap's own M2 ("organisms `Header` and `Footer` driven by `site.yaml` and
`nav.yaml`" — `bos-app-roadmap.md` line 172), reached here specifically for `bos-demo` rather than
`bestierealestate`, and specifically **database-backed** rather than `site.yaml`/`nav.yaml` files — which
is not a new decision, it's the same one already made for pages and settings in `bos-demo-mvp.md`, and
the current `settings.go`'s own doc comment already names this exact moment: *"nav/footer editing waits
for roadmap M2, when header/footer exist as renderable regions."* This document is that wait being over.

## 2. What already exists — checked directly, not assumed

- **`settings` is already a DB-backed singleton**, not a file: one row (`id = 1`) in Postgres, `Store.Get`/
  `Store.Update`, `GET`/`PATCH /api/admin/settings`, and an admin route at `/admin/settings` (currently a
  plain form for `site_title`/`tagline` only — no tabs, since there was nothing else to show a tab for
  yet). Extending this same singleton, not introducing a new resource, is the natural next step.
- **The renderer registry mechanism is already built and proven**
  (`bos-svelte/src/lib/renderers/{types,registry,index}.ts` — `import.meta.glob('./*/index.ts', {eager:
  true})`, no hand-written switch, no generated file). Chrome presets need the exact same shape, just a
  second, separate registry (a header preset and a footer preset are not interchangeable with each other
  or with a content renderer).
- **`AdminShell` already links to `/admin/settings`** in its sidebar (confirmed in the prior scope's SSR
  check) — no new nav wiring needed, only what that route renders.
- **There is nothing to literally "port" as a file.** `bos-demo`'s consumer mount
  (`consumers/clients/bestie/bestierealestate`) is still M0's placeholder app — one `.svelte` file total,
  confirmed by direct search — the real header/footer UI lives only in the original production app, which
  is not and has never been imported into this repo (`feedback_no-client-info-in-sdlc-kit`). What's real
  and checked is the **shape** of BR's config, already extracted into `bos-page-designer.md` §4 by reading
  `bos-consumer-plan.md` §13 directly: `header_nav[]` (`type: link|separator, title, page_id, custom_path,
  params, icon, enabled, mobile, new_tab`) and `footer.sections[]` (`heading, source, links[]`). "Port it
  here" means rebuild to that proven shape in `bos-svelte`, the same "reuse the shape, not the code" call
  page-designer already made — there is no source file to move.

## 3. Data model — extend the existing `settings` singleton

Two new `jsonb` columns on the existing `settings` table, not a new table (still one site, one row):

```sql
ALTER TABLE settings
    ADD COLUMN header jsonb NOT NULL DEFAULT '{"enabled": true, "preset": "default", "nav": []}',
    ADD COLUMN footer jsonb NOT NULL DEFAULT '{"enabled": true, "preset": "default", "sections": []}';
```

Shape (typed in Go, since — unlike a page section's opaque `content` — the admin and the renderer both
need to agree on these fields):

```
header: { enabled: bool, preset: string,
          nav: [{ type: "link"|"separator", title, page_id?, custom_path?, icon?, enabled, web, mobile, new_tab }] }
footer: { enabled: bool, preset: string,
          sections: [{ heading, links: [{ title, page_id?, custom_path?, enabled, web, mobile }] }] }
```

Two fixes over BR's own shape, both already called out in `bos-page-designer.md` §4 and applied here for
real: `web`/`mobile` visibility is symmetric on every nav item **and** every footer link (BR only has
`mobile` on nav items today), and the `page_id`-or-`custom_path` duality is kept on both nav items and
footer links (a link either resolves to a real page's route or is a literal URL — the same
content-type-reference pattern `bos-constructor.md` already uses elsewhere). `footer.sections[].source` is
dropped for now — it's for a dynamically-sourced footer column (e.g. "latest guides"), which needs the
content-type/source model from M3; out of scope until then, easy to add as an optional field later.

## 4. The chrome registry — same mechanism, a second instance

`bos-svelte/src/lib/chrome/header/<preset>/index.ts` and `.../chrome/footer/<preset>/index.ts`, each
exporting a `registeredChrome` object (`id`, `label`, `component`), discovered by its own
`import.meta.glob`, exactly mirroring `renderers/registry.ts`. Two registries, not one, because a header
preset and a footer preset are never chosen from the same list. First preset in each: `default` header
(site title/logo + the `nav[]` list, filtered by `enabled`/`web` at render time), `default` footer
(`sections[]` rendered as columns, filtered the same way). This is framework code (`bos-svelte`) —
consumer-neutral, same as the page renderers.

## 5. Public rendering — the first chrome `bos-demo` will ever have

A new `apps/web/src/routes/+layout.svelte` (none exists today) with a sibling `+layout.server.ts` that
loads `/api/settings` (a new **public**, unauthenticated read — `header`/`footer`/`site_title` only, not
the admin's full settings shape) once per navigation. The layout renders the chosen header preset, then
`{@render children()}` (today's bare page content, unchanged), then the chosen footer preset. If
`header.enabled`/`footer.enabled` is false, that slot renders nothing — the "enable/disable" the user
asked for. A page's own content area is completely unaffected by any of this: it keeps resolving its
`main`-slot sections exactly as it does today. This is the literal app-default-vs-page-content split
requested — chrome lives one layout level up from the page, the page never touches it.

## 6. Admin UI — `/admin/settings` grows tabs, same patterns already proven

`/admin/settings` moves from one flat form to `TabbedPageShell` with three tabs — `General` (today's
site title/tagline form, unchanged), `Header`, `Footer` — the same route-driven-tabs pattern already
proven at `/admin/pages/[id]` (`+layout.svelte` defines the tabs, each tab a sibling route). `Header`
and `Footer` tabs each show: an enable toggle, a preset `<select>` sourced from that chrome registry
(mirroring the "Add section" renderer `<select>` already built), and a reorderable list of nav
items/footer sections — add/edit opens a `Modal` form (same component, same pattern as the page section
editor you just reviewed), remove and reorder are inline buttons like the page sections list.

## 7. API — typed, not opaque

`Settings` struct gains `Header`/`Footer` fields (typed Go structs matching §3's shape, validated on
`PATCH` — unlike page-section `content`, which is deliberately opaque per-renderer, chrome fields are
known and worth validating, e.g. rejecting a nav item with neither `page_id` nor `custom_path`). A new
public, unauthenticated `GET /api/settings` returns only what the public layout needs (`site_title`,
`header`, `footer`) — distinct from `GET /api/admin/settings`, which stays behind `RequireAuth` (today
`authfake.Always()`) and returns the full row for editing.

## 8. Per-page chrome override — deferred on purpose, not blocked

Not built now, per your own call ("we will build it when and if we need"). Confirmed it doesn't need a
redesign later: a page is already its own row with its own sections: adding an optional
`chrome_override: {header?, footer?}` field to a page later is additive to the existing `PageWithSections`
shape, resolved by the same layout that already reads site defaults — nothing in §3–§6 needs to change
shape to allow it. Nothing to build for this now.

## 9. Scope breakdown

| Scope | What | Gate |
|---|---|---|
| C1 | Migration (§3), `Settings` struct + validation, `GET /api/settings` (public) alongside the existing admin one | `go build`/`go vet` clean; curl both endpoints, confirm the public one omits nothing sensitive and the admin one round-trips a header/footer edit |
| C2 | `bos-svelte/src/lib/chrome/{header,footer}/default/*`, the two registries, root `+layout.svelte`/`+layout.server.ts` in `bos-demo`'s web app | Public page SSR HTML shows header nav links and footer sections wrapping the existing page content; disabling either in the DB makes it disappear from SSR output |
| C3 | `/admin/settings` tabs (General/Header/Footer), nav/footer editors in `Modal` | Three distinct routes/tabs render (same SSR-route-count proof used for the page designer); an added nav item persists and appears in the public header after reload |

## 10. What this document does not decide

- Visual design of the `default` header/footer preset (spacing, mobile menu behavior) — same "offline
  design pass, this just fixes the shape" stance as `bos-page-designer.md` §6.
- A second (non-`default`) preset for either slot — built when there's a real second layout that needs
  one, not speculatively.
- `footer.sections[].source` (dynamically-sourced footer columns) — waits for M3's content-type/source
  model, noted in §3 as an easy additive field later.
- Per-page chrome override's actual UI — §8 only confirms it's architecturally unblocked, not designed.
