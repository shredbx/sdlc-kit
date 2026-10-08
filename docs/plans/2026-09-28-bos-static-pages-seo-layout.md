# bos-demo — static page types, layout regions, SEO wiring, real content

Scopes A–D of the plan agreed on top of `docs/proposals/bos-layout-presets.md` and
`docs/proposals/bos-site-chrome.md`. Ask: recreate bestierealestate's homepage and legal pages as
real, production-shaped static pages, wire SEO, then be ready to start on content packages. Commits:
`529554b`, `d18bb1e`, `a977a2f`, `f7ec70f` on `worktree-bestierealestate--stage-2`.

## What was actually asked, and the honest boundary drawn up front

BR's real homepage (`apps/web-svelte/src/routes/(public)/+page.svelte`, read directly, nothing
copied) is not a static page — it's built from `HeroSlideshow`, `PropertySearch`, `PropertyCard`,
`FAQSection`, `HomeMapBand`, all backed by live property data. Recreating it pixel-for-pixel needs
the properties content package (roadmap M9, "the largest surface"), which does not exist. What this
scope built instead: a real, production-styled generic landing page (`default` layout: hero + prose),
honest about not being that page. BR's legal pages (`terms`/`privacy`/`cookies`) genuinely are simple
prose shells (`PublicPageShell` + `ContentPage`) — those were fully, honestly recreated in shape.

## Scope A — layout regions (hero vs. main)

**Correction found mid-build**: the layout-presets proposal assumed `page_sections.slot`, a column
that lived on the pre-cms-port `pages`/`page_sections` tables (dropped in migration
`005_drop_pages_and_sections`). The new `cms.Section` envelope has `Kind`/`Payload`/`Hidden`, no slot.
Fix: a layout partitions the sections it's given **by `kind`**, not a stored slot — `default` renders
the first `hero`-kind section in a hero region and everything else in main; `legal` renders main only
and structurally drops any `hero`-kind section. One real column was still needed: `cms_pages.layout`
(which preset a page uses).

- `platform/go/packages/cms/migrations/20260928000000_add_page_layout.{up,down}.sql`
- `cmspage.go` (`Layout` field + mapper wiring), `postgres.go` (`UpsertBySlug` gains a `layout`
  scalar param, COALESCE semantics like `title`), `http.go` (`savePageRequest.Layout`,
  `publicCmsPage.Layout`), `cmspage_test.go` (column-count assertions updated 15→16, not skipped)
- `platform/svelte/frameworks/bos-svelte/src/lib/layouts/{types,registry,index}.ts`,
  `default/{Layout.svelte,index.ts}`, `legal/{Layout.svelte,index.ts}`
- `ui-cms`: `CmsPage.layout`, new `PageSettingsScreen.svelte` (layout picker), `PageEditorLayout.svelte`
  gains a Settings tab
- bos-demo public routes (`(site)/+page.svelte`, `(site)/[slug]/+page.svelte`) resolve via the layout
  registry instead of a hardcoded `<h1>+SectionList`

**Gate**: Go build/vet/test clean (cms package + bos-demo/apps/api); svelte-check clean; SSR-confirmed
a `terms` test page with a `hero` section forced into its data renders no hero region — the hero
payload only appears in client hydration JSON, never the DOM.

## Scope B — production visual pass

Real typography/spacing/color on Header, Footer, Hero, Prose using the `bos-brand` tokens (shipped
earlier as T1/T2) instead of hardcoded hex. Two small additions beyond pure CSS, disclosed rather than
silently folded in:

- Header had no site identity at all — `bos-site-chrome.md` §4 specified "site title/logo" but it was
  never wired. Added `siteTitle` to `RegisteredHeader`'s signature, piped from `settings.site_title`.
- Hero's heading changed `<h2>`→`<h1>` — the route's own hardcoded `<h1>` is gone since Scope A, so a
  hero is the only section that can supply the page's top-level heading.

**Known gap, not fixed here**: a `legal`-layout page with only a `prose` section has **no `<h1>`** —
`Prose` still renders plain text, not real markdown (a standing, documented deferral —
`docs/proposals/bos-demo-mvp.md`). `@humanspeak/svelte-markdown` is already a `core-ui` dependency
with a proven safe-rendering pattern (`ChatMessage.svelte`/`.security.ts`) if this gets picked up —
it's a real behavior change, not styling, so it wasn't folded into this pass.

**Disclosed limitation**: no browser-automation tool (Playwright/`chromium-cli`) is available in this
environment. Verification was CSS-cascade tracing (brand token names confirmed to match what
`brand-bridge.css` aliases them to) plus SSR markup inspection, not an actual rendered screenshot.

## Scope C — SEO wiring

The Go side already stored the full shared `seo.SeoMeta` (`og_*`, `canonical_url`, `noindex`) and
`@sbx/ui-seo` (`SeoHead`, `SeoEditor`, `resolveSeo`, JSON-LD builders) was already ported and fully
working — neither needed building. The gap was purely wiring: `ui-cms`'s `PageSeoScreen` had
reimplemented a narrower 2-field form, and no public route emitted `SeoHead` at all.

- `ui-cms/types.ts`: `SeoMeta` re-exported from `@sbx/ui-seo/types` (was a local, narrower duplicate)
- `PageSeoScreen.svelte`: now renders the real `SeoEditor` (live SERP/social previews), saving via
  `toSeoDraft`/`fromSeoDraft` — the same resolution `SeoHead` renders from, so the preview can't drift
- `(site)/+page.svelte`, `(site)/[slug]/+page.svelte`: render `SeoHead`
- Package/workspace additions: `@sbx/ui-seo` as a dependency of `ui-cms` and `bos-demo/apps/web`;
  `platform/svelte/packages/seo/ui-seo` added to `bos-demo`'s `pnpm-workspace.yaml`

**Gate**: view-source on `/about` (with a stored `og_image`/`meta_description` override) shows the
full title/description/canonical/OG/Twitter set; admin SEO tab shows the real editor with live
previews pre-filled from the stored value.

## Scope D — real content (data, not code — nothing here is in git)

7 published `cms` pages: `home`, `about`, `features`, `contacts` (`default` layout), `terms`,
`privacy`, `cookies` (`legal` layout) — every one with a real `meta_description`. Header nav
(Home/About/Features/Contact) and a two-section footer (Site / Legal) wired through
`PATCH /api/admin/settings/{header,footer}`.

**Two mistakes made and self-caught while doing this** (both logged in
`docs/proposals/bos-site-chrome.md`):
1. Sent `header`/`footer` in the body of `PATCH /api/admin/settings` — that endpoint only reads
   `{site_title, tagline}` and does a full overwrite with no field-presence semantics, so it silently
   dropped the header/footer payload **and** wiped `site_title`/`tagline` to empty (they weren't in
   that request either). Caught via an immediate follow-up `GET`; fixed by re-sending through the
   correct `PATCH /settings/header` / `/settings/footer` endpoints and restoring the wiped fields.
2. The chosen nav linked to `/features` and `/contacts`, which existed under the pre-cms-port model
   but not the current one (no such `cms` rows). Caught the same way (checking the admin pages list
   before trusting the links); fixed by creating real `features`/`contacts` pages rather than leaving
   dead links.

**Gate**: all 7 public routes return 200 with correct, non-duplicated `<title>`; header/footer nav
renders real links on every page; admin Pages list shows all 7; layout separation holds across the
whole set (hero+main on the 4 `default` pages, main-only on the 3 `legal` pages).

## What this doesn't decide

- Content packages (contacts kit, properties kit, …) — the plan's own next phase, deliberately not
  started; which one to build first needs its own discussion.
- Real markdown rendering for `Prose` (would fix the missing `<h1>` on the 3 legal pages) — flagged in
  Scope B, not built.
- A visual QA pass with an actual browser tool — flagged in Scope B, not available in this
  environment.
