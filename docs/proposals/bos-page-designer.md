# bos — the page/layout designer: admin shell wiring and the renderer registry

Last updated: 2026-09-27
tags: bos, admin, designer, layout, renderer-registry, topbar-slots, proposal

Status: **proposal, not yet approved.** Extends `docs/proposals/bos-constructor.md` (the page/block/
content-type/source model, §4.1–4.5 — nothing here contradicts it) with the concrete mechanics two
things need: the admin UI actually having a real shell to grow into, and a way for a package (kit) to
register a renderer bos-svelte can discover. Everything below was checked against the real code
(`platform/svelte/packages/ui/core-ui`, `platform/python/frameworks/agent-framework`, and
`projects/demo/bos-demo`'s current admin), not written from memory of the design docs alone.

## 1. Why this document exists

`bos-demo`'s MVP 0.0.1 (`docs/proposals/bos-demo-mvp.md`) built a working but deliberately flat admin:
one route (`/admin`), two tabs swapped by client-side state, hand-rolled forms. That was the right
scope for "prove the loop." The next ask is a real **page/layout designer** — pick a layout preset,
see its slots (topbar, hero, footer, ordinary sections), assign content to each slot from
shared/configured sources, edit in a popup. That's a materially bigger admin surface (a Pages list →
a page → its layout → its sections, nested), and building it on the same flat-page shortcut would mean
re-doing this work a third time. This document designs the two structural pieces first.

## 2. The admin shell gap — precise, not approximate

`core-ui`'s `TabbedPageShell.svelte` (`platform/svelte/packages/ui/core-ui/src/lib/components/layouts/`)
is real, mature, ported code — route-driven tabs, a detail mode, a topbar-forwarding mechanism for page
actions. It depends on `useTopbarSlots()` (`.../stores/topbarSlots.ts`), whose own doc comment says
exactly what's required:

> "A shell layout (e.g. an admin `(admin)/+layout.svelte`) calls `createTopbarSlots()` once. It renders
> the layout's toolbar/actions snippets so they read from the returned reactive store. Descendant pages
> call `useTopbarSlots()` and assign their own snippets... Must be called inside a shell layout that
> called `createTopbarSlots()`. Throws if no provider is found."

Checked directly: **no such shell layout exists yet.** `CollapsibleSidebarLayout.svelte` (the other
ported layout in the same folder) never calls `createTopbarSlots()` — grepped the whole package, zero
matches outside `TabbedPageShell` itself. And `bos-demo`'s current `apps/web/src/routes/admin/+page.svelte`
is a single flat route with client-side tab state (`activeTab = $state('pages')`), not a route group at
all — so `TabbedPageShell` genuinely cannot be dropped in today; it would throw immediately (no provider).

**Proposed fix**, once approved as its own scope: a new component in `bos-svelte` (exact name decided at
that scope, e.g. `AdminShell.svelte`) that:

1. Calls `createTopbarSlots()` once.
2. Renders the actual chrome: a sidebar nav (built on the already-ported `CollapsibleSidebarLayout`) plus
   a topbar bar that renders the registered `toolbar`/`actions` snippets from the store.
3. Is used by the app's own `src/routes/(admin)/+layout.svelte` (a real SvelteKit route group,
   replacing the current flat `/admin/+page.svelte`), so **every** admin screen sits inside one shell.

Each admin screen (Pages list, Settings, and later a page's own designer) then becomes its own route
(`/admin/pages`, `/admin/settings`, `/admin/pages/[id]`) using `TabbedPageShell` for whatever tabs that
screen itself needs (e.g. a page's own designer might have "Layout" / "SEO" tabs) — which is exactly
what `TabbedPageShell`'s route-driven tab strip is for. This is framework work (`bos-svelte`), not
app work — any consumer gets a real admin shell for free, not just `bos-demo`.

`Modal.svelte` needs no such fix — it's a self-contained overlay (`open`/`onclose`/`header`/`footer`/
`children` snippets, no context dependency) and is already proven working in `bos-demo`'s delete-confirm
flow. It's the right primitive for the section popup editor described below, as-is.

## 3. The page/layout designer model — mapping to what's already designed

Nothing here is a new model — it's `bos-constructor.md` §4.1–4.5, made concrete for the admin UI:

| Admin concept | `bos-constructor.md` vocabulary |
|---|---|
| Pick a layout for a page | the page's `layout:` field, resolving to a **layout preset** (a named skeleton with **slots**) |
| "topbar, footer, hero, normal sections" | the layout preset's slots — organisms `Header`/`Footer` (site-wide, from `site.yaml`/`nav.yaml`), `Hero`, and one or more ordinary content slots |
| "define what we want on that page from available/shared items" | a **block**: `{renderer, content \| source, settings}` — `source` binds to a **content type**'s collection, configured once and reusable by any page |
| view/edit in a popup | `Modal.svelte` (§2 above) — the block's editor form (pick renderer, pick inline content or a source binding, set `settings`) opens in a `Modal`, not a page navigation |

**First layout presets** (shape only — exact visual design is the offline pass you mentioned; this just
fixes the slot names so the admin and the renderers agree on them): `default` (topbar, hero, main,
footer), `legal` (topbar, main — a single `prose` block, footer), `landing` (topbar, hero, one or more
main sections, footer), `minimal` (main only — no topbar/footer, e.g. a print or embed view). This
matches the layout preset list already named in the roadmap (M2's `default`/`legal`, M4's `landing`).

**Admin UX for a page's sections**, concretely: `/admin/pages/[id]` shows the page's layout's slots in
order; each slot lists its current blocks (empty allowed — falls open to the layout's own defaults, per
`bos-constructor.md`'s resolution order); "Add section" opens a `Modal` to pick a renderer from the
registry (§4), then either inline content (typed by the renderer) or a source binding (collection +
filter + sort + limit, per §4.4's `source:` shape); reordering is drag or up/down within the slot.

## 4. Topbar and footer configuration — reusing BR's shape, fixing its gaps

The old app's real shape (`site_config`, measured in `docs/bos-consumer-plan.md` §13, the consumer
repo — not duplicated here, referenced only): `header_nav[]` (`type: link|separator, title, page_id,
custom_path, params, icon, enabled, mobile, new_tab`) and `footer.sections[]` (`heading, source, links[]`).
That's real, working prior art — reuse the *shape*, not the code (it's Postgres-row-shaped for the old
schema, not a straight port). Two concrete improvements, both already implied by fields the old shape
half-has:

- **Per-surface visibility is already a field (`mobile: bool`) on nav items but not on footer links** —
  extend it symmetrically so both nav and footer sections/links declare `web`/`mobile` visibility, not
  just nav.
- **A nav item's target is already either a `page_id` reference or a `custom_path` string** — keep that
  duality (a link either resolves to a real page's route or is a literal URL), since it's exactly
  `bos-constructor.md`'s content-type-reference pattern applied to navigation.

This lives in `site.yaml` (`bos-constructor.md` §4.1: "Site: identity, defaults, navigation, footer, SEO
defaults... `site/site.yaml`"), not a new file — nothing new to name here, just the concrete field list,
finalized when this scope is actually built.

## 5. The renderer registry — the mechanism, verified against the pattern it's modeled on

`bos-constructor.md` §3 already named `agent-framework`'s "add a folder, restart" discovery as the
convention to mirror. Read the actual code (`platform/python/frameworks/agent-framework/src/agent_framework/core/registry.py`)
to get the mechanism right rather than the phrase alone:

> "Discovers agents from an importable package: each `<agents_package>.<name>.config` module becomes
> one `RegisteredAgent`. No build step — add a folder, restart." — walks the package directory,
> imports each subfolder's `config` module, collects the typed object it exports.

Python can do this at runtime because `importlib` can import an arbitrary path at any time. A Vite/
SvelteKit app can't do the same thing in the browser, but Vite has the direct build-time equivalent:
**`import.meta.glob`**, which scans a folder pattern and returns a map of path → module, resolved at
build/dev time (and re-scanned automatically by Vite's dev server when a matching file is added — no
manual registry file to hand-edit, which is actually a small improvement over `sbx.framework`'s old
approach of *generating* a `section-registry.ts` file, per the research in
`docs/research/sbx-framework-inventory.md`: one fewer moving part, same "never hand-written" guarantee).

Proposed convention: each renderer lives in its own folder,
`bos-svelte/src/lib/renderers/<kind>/index.ts`, exporting a typed `registeredRenderer` object (mirroring
`RegisteredAgent`'s shape: an id, the component, and the metadata `bos-constructor.md` §4.3 already
requires — what content shape it accepts, e.g. `list` vs a single entry, so `bos check`'s compatibility
check has something to check against). `bos-svelte` builds the registry once via
`import.meta.glob('/src/lib/renderers/*/index.ts', { eager: true })`. A kit package (`properties`,
`contacts`, …) that wants to contribute a renderer for its own content type adds a folder here — "program
new one and reload to assign," same spirit as the Python side, adapted to how Vite actually works.

## 6. What this document does not decide

- Exact visual design of the layout presets (topbar/footer look, spacing, responsive behavior) — the
  offline design pass you described; this document only fixes the slot vocabulary so that pass has
  something stable to design against.
- The exact TypeScript shape of `registeredRenderer` / the metadata a renderer declares — decided when
  the first real renderer (beyond the MVP's plain title+body) is actually built, the same "build to
  prove, then model" discipline used for the ported packages.
- Whether `AdminShell` also needs `bestierealestate`-specific chrome — out of scope; this is framework
  work, consumer-neutral by construction (D24/D25).
- A scope-by-scope before/after tree for building any of this — that's the next step once you confirm
  this shape is right, same pattern as `bos-demo-mvp.md`'s own §6.
