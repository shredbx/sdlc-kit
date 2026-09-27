# bos — the constructor: how a site is built from pages, blocks, content types and sources

Last updated: 2026-09-26
tags: bos, constructor, pages, blocks, presets, branding, atomic, file-structure, proposal

Status: **proposal, not yet approved.** Parent: `docs/proposals/bos-system-design.md` (decisions D18 to D26). Companion:
`docs/proposals/bos-app-roadmap.md` (the milestones). This document replaces the earlier entry-points draft; the dynamic-feature part of that
draft is section 8 here. Nothing here is modeled or built. Each definition still needs its own approval when the milestone that proves it
lands.

Everything specific to the first consumer (its pages, colors, routes, numbers) is in that consumer's repo, `docs/bos-consumer-plan.md`.

---

## 1. What we are building, and the rules

Two frameworks, one for the API (`bos-go`) and one for the web (`bos-svelte`), that work like a **constructor**. An app is mostly data:
a site is a set of **pages**; each page picks a **layout preset**; the layout has **slots**; each slot holds **blocks**; each block picks a
**renderer component**, a **content type** and a **data source**. Branding (logo, colors) is data. Presets exist at every level. The app's
own code is what the constructor cannot express.

Rules, from your direction and from the evidence below:

1. **The old app is the baseline.** It already works, so every milestone must produce an output comparable to the part of the old app it
   ports (the roadmap's parity gates). Nothing is built that the old app does not do.
2. **Start from an empty app.** It runs and builds with both frameworks attached before anything else is added.
3. **Static first, dynamic later, with the same model.** A page and a block do not change when their data moves from a file to the API to a
   database; only the source changes.
4. **Code declares, data selects.** Capabilities (components, renderers, sources, modules) are typed code with typed metadata. Data
   (YAML, Markdown) only chooses among them. Data never contains code, and code is never stored as data.
5. **The framework is a package the app imports.** It does not generate files into the app's tree.
6. **Only what the old app uses is imported.** New framework code is written for the constructor; the ported libraries are used when a
   milestone needs them.

## 2. Lessons from the earlier bos

The earlier attempt (`sbx.framework/projects/bos`) is a YAML row store, a Python resolver that generates a SvelteKit app, and a component
library stored as templates. What its own audits and retrospectives show:

| Kept (it worked) | Dropped (it hurt) |
|---|---|
| A page with **zero code**: a page row plus section rows | **Components as opaque template text.** A component was four files (row, file row, template row, body); props were declared on a separate role row |
| Overlay order **app ← theme ← fail-open default** | **Props were fiction.** 14 of 16 component rows disagreed with their artifact; `props` had zero readers |
| A layout, header and footer **chosen by one reference**, with a refusal when the choice implements the wrong role | **A block's data source was hardwired** in a resolver that switched on role names, so a new content type meant editing Python |
| **Validation at write time** that blocks empty content ("anti-blanking") | **Generated files committed into one shared app tree**: the last project run overwrote it (seven recorded drift events) |
| Safe-link handling, fail-open defaults | **A heavy runtime**: Python under the live site, one process spawn per request |
| Typed collections (38 FAQ items, derived detail routes) from data rows | **A page took four to five rows**; the home page was hand-coded outside the model; the resolver was a monolith; about 1,050 rows and gates |

Consequences for bos: a page is **one file**; a component is **one folder of real code with its metadata beside it and derived from the same
declaration**; sources are **registered adapters** chosen per binding; the home page is an ordinary page; nothing is generated into the
app; the runtime is Go and SvelteKit only; `bos check` runs in seconds.

## 3. What we take from the market

Conventions below were confirmed against each project's current documentation.

| Solution | The idea | What bos adopts | What bos leaves |
|---|---|---|---|
| **WordPress** (block themes) | `theme.json` is the central configuration (settings say what is allowed, styles say what is applied); folders `templates/`, `parts/`, `patterns/`, `styles/`; block patterns; child themes | `brand.yaml` in the same shape (allowed vs applied is a later refinement); page/layout/block presets as "patterns"; **override by path** (an app file shadows the framework's) | the PHP template hierarchy; content in a database from day one |
| **Shopify** (Online Store 2.0) | a JSON template lists **sections and their order**; a section's schema declares its settings, blocks and **presets**; `settings_schema.json` (structure) is separate from `settings_data.json` (values) | a page lists blocks per slot; a renderer's metadata declares its settings and its presets; **structure in code, values in data** | Liquid; theme-editor JSON as the source of truth for pages |
| **Next.js** (App Router) | the **file system is the router**; nested layouts; **route groups** `(group)` share a layout without adding a URL segment; **private folders** `_x` are not routable; a per-page `metadata` export | `site/pages/` is the router; `_layout.yaml` sets a subtree's layout; `(group)` and `_private` with the same meaning; per-page SEO in front matter | React Server Components; a route file per page |
| **NestJS** | **feature modules**; **dynamic modules** with `forRoot` (configure once), `register` (configure per caller), `forFeature` (customize the root configuration); the CLI **generates** resources (`nest g resource`) | three configuration levels for kits (framework root, kit, block); **generators** (`bos g page`, `bos g collection`, `bos g renderer`) | decorators; reflection |
| **Astro** | **content collections**: `defineCollection` with a **loader** (`glob` for a folder of files, `file` for one file, or a custom loader) and a **schema**; `getCollection`; **live collections** for runtime data; layouts | a **content type** is a schema plus a default source; sources are adapters; the same collection is a folder of files today and an API tomorrow | MDX in pages; islands as a general rule |
| **W3C Design Tokens and Style Dictionary** | a token is `$value` with a `$type`, in groups, with `{alias}` references and `$extends`; a pipeline turns sources into platform outputs through **transforms**, **formats** and hooks, configured by `source`, `platforms`, `buildPath` and `files` | the token structure (three tiers, aliases, modes as overrides); the pipeline's extension points as our **plugin** kinds; sources kept apart from generated output | authoring in JSON; design-tool sync |
| **agent-framework** (this repo) | `src/<pkg>/{core, api, server}`; each adapter is a `base` interface with the implementations beside it; **"add a folder, restart"** discovery; a **fake adapter chosen by configuration**; flat tests that use fakes; `core` never imports `server` | the same layering in `bos-go`; folder discovery for renderers and modules; fakes for every source so the app runs with no external service; tests without network | its lack of a declarative spec and of dev/prod profiles (it has two environment switches) |

## 4. The model

### 4.1 The parts

| Part | Meaning | Defined in |
|---|---|---|
| **Site** | identity, defaults, navigation, footer, SEO defaults, which sources are used | `site/site.yaml` |
| **Brand and tokens** | logo variants, colors by role, typography, radius, spacing, shadow, motion; presets and plugins for them | `site/brand/` (section 4.6) |
| **Page** | a route; a page **is a content entry of type `page`**; one file | `site/pages/**` |
| **Layout preset** | a named skeleton with **slots**, a maximum width, default blocks per slot | code (a component and its metadata) |
| **Block** | one item in a slot: a renderer, its content (inline or from a source), its settings | the page file |
| **Renderer** | a component that draws content of a declared shape | code (an organism, with metadata) |
| **Content type** | a typed schema (fields, references) with a default source | `site/types/*.yaml`, or the framework's |
| **Data source** | an adapter that yields entries of a content type | code (adapter); chosen in data |
| **Preset** | a named bundle at the level of a site, a layout, a page or a block | data, framework-provided and app-provided |

### 4.2 A page is one file

```markdown
---
# site/pages/terms.md   →  the URL /terms
layout: legal                      # a layout preset; default comes from site.yaml
title: Terms of Service
seo: {description: "The terms that apply to this site."}
blocks:                            # optional: fill or replace a slot's default blocks
  hero: {renderer: hero, content: {eyebrow: Legal, headline: Terms of Service}}
---
The body is Markdown. The `legal` layout renders it in its `main` slot with the `prose` renderer.
```

A page with several blocks, one bound to a collection:

```markdown
---
# site/pages/index.md   →  /
layout: landing
blocks:
  hero: {renderer: hero, content: {headline: "Find your place", image: media/hero.jpg}}
  main:
    - {renderer: feature-cards, content: {items: [{title: One, text: …}, {title: Two, text: …}]}}
    - renderer: accordion
      source: {collection: faq, where: {featured: true}, sort: position, limit: 5}
      settings: {expand_first: true}
---
```

**Routing.** `index.md` is a folder's root; `about.md` gives `/about`; `legal/terms.md` gives `/legal/terms`; `(marketing)/pricing.md` gives
`/pricing` and shares the group's `_layout.yaml`; a folder or file starting with `_` is never a route (shared partials live there);
`guides/[slug].md` with `collection: guide` renders one page per entry. A `redirect:` key in front matter, or `redirects` in `site.yaml`,
declares a redirect. There are no route files in the app for static pages: the framework provides one catch-all route.

### 4.3 The block anatomy

```yaml
- renderer: accordion            # an id in the renderer registry
  content: {…}                   # inline content, typed by the renderer   (exclusive with source)
  source:                        # or: a binding to a data source
    collection: faq              # a content type's collection
    where: {featured: true}
    sort: position
    limit: 5
  settings: {expand_first: true} # presentation options, typed by the renderer's metadata
  preset: faq-home               # optional: a block preset that pre-fills the above
```

**Compatibility is checked, not assumed.** A renderer declares what it accepts (an inline schema, or a content type and a shape such as
`list`). `bos check` and the loader refuse a block whose content or source does not fit its renderer, and a layout refuses a block in a slot
that does not allow it. This is the one thing the earlier bos lacked most (a refusal when a choice implements the wrong role).

### 4.4 Content types and sources

```yaml
# site/types/faq.yaml
name: faq
fields:
  question: {type: string, required: true}
  answer:   {type: markdown, required: true}
  category: {type: ref, to: faq-category}
  featured: {type: bool, default: false}
  position: {type: int}
source: {kind: file, dir: content/faq, format: md}   # the default source for this collection
```

| Source kind | Reads from | Available in | Phase |
|---|---|---|---|
| `inline` | the block itself | web | static |
| `file` | a folder of Markdown or YAML files, or one file | both stacks | static |
| `api` | a `bos-go` endpoint | web | dynamic |
| `db` | the database | `bos-go` | dynamic |

A source is chosen **per collection and per profile** in `site.yaml`, so development can read files while production reads the API. A
**fake** source (in memory) exists for every kind so the app and its tests run with no external service.

Because a page is itself an entry of type `page`, moving pages from files into the database (so an editor can change them) is a change of
that one source, not a change of any page.

### 4.5 Presets

| Level | What it is | Example |
|---|---|---|
| Site | default layout, navigation skeleton, a set of block presets | `real-estate` |
| Layout | a layout component with its slots | `default`, `landing`, `legal`, `listing`, `article`, `minimal` |
| Page | a starting page for the generator | `legal-page` = layout `legal` + a prose body |
| Block | a renderer with pre-filled settings, content or source | `faq-home` = `accordion` bound to featured FAQ entries |

**Resolution order** (the earlier bos's rule, kept): the value on the page ← the app's presets ← the framework's presets ← the renderer's own
defaults, and every step **fails open** to a sensible default. Presets live beside the kind they preset (section 4.6, the kind-folder convention), in the framework and in the app, and an app
file with the same id overrides the framework's.

### 4.6 Branding and design tokens

Everything about how the site looks lives in **one folder, `site/brand/`**, and everything in it belongs to one of five kinds: a **config**, the
token **source** files, **presets**, **assets**, and **plugins**. A kind's folder is created when its first item exists, never scaffolded empty.

```
site/brand/
├── brand.yaml                    the entry config: identity, which theme preset, logo references, modes, output options, checks
├── tokens/                       the token SOURCE files: the W3C Design Tokens structure, authored as YAML, one file per category
│   ├── primitive/                raw values: palette.tokens.yaml · typography.tokens.yaml · size.tokens.yaml · radius · shadow · motion
│   ├── semantic/                 roles that point at primitives: color.tokens.yaml (primary, accent, surface, text, …) · typography · spacing
│   ├── component/                only when a component needs its own token: button.tokens.yaml · header.tokens.yaml · card.tokens.yaml
│   └── modes/<mode>/             overrides of semantic tokens for a mode: modes/dark/color.tokens.yaml
├── presets/                      named, reusable starting points
│   ├── palette/  typography/  scale/     one file each: a set of primitives, a font pairing with its scale, a spacing/radius/shadow/motion set
│   └── theme/                    a bundle that names a palette, a typography, a scale and a mode set
├── assets/                       SOURCE files the site ships
│   ├── logo/                     mark.svg · wordmark.svg · logo.svg (for light backgrounds) · logo-dark.svg
│   ├── favicon/                  favicon.svg · favicon.png · apple-touch-icon.png
│   ├── fonts/                    local font files (pinned, so builds and captures do not depend on a font host)
│   └── images/                   default social image
└── plugins/                      the app's extensions of the token pipeline (code): transforms, formats, validators
```

**Generated output is never committed and never lives in `site/` or `apps/`.** The build writes `.bos/build/tokens/{tokens.css, tokens.ts,
tokens.json}` (git-ignored) and the web app imports them through the framework's alias; the Go API reads only `brand.yaml` and the assets (to
serve the name and the logo addresses) and never transforms tokens.

**The entry config, `brand.yaml`:**

```yaml
name: Example
theme: default                      # a theme preset: the app's own, else the framework's; the tokens below override it
modes: [light, dark]                # light is the base; another mode is a folder of overrides under tokens/modes/
logo:
  mark: assets/logo/mark.svg        wordmark: assets/logo/wordmark.svg
  light: assets/logo/logo.svg       dark: assets/logo/logo-dark.svg
  favicon: assets/favicon/favicon.svg
output: {prefix: bos, css: true, ts: true, json: true}
check: {wcag: AA, pairs: [[color.text, color.surface], [color.on-primary, color.primary]]}
```

**Token source files** follow the W3C Design Tokens structure (`$value`, `$type`, `$description`, groups, `{alias}` references, `$extends`),
authored in YAML. A color may be written as a hex string; the pipeline normalizes it to the specification's color object. Three tiers, each in
its own folder, so a change is made at the right level:

```yaml
# tokens/primitive/palette.tokens.yaml     tier 1: raw values, named for what they are
palette:
  $type: color
  ocean: {500: {$value: "#1F4E79"}, 300: {$value: "#5B8DB8"}}
  amber: {500: {$value: "#D98E04"}}
  neutral: {50: {$value: "#F7F7F5"}, 900: {$value: "#1B1B1B"}}
```
```yaml
# tokens/semantic/color.tokens.yaml        tier 2: roles, named for what they are FOR; components use only these
color:
  $type: color
  primary:    {$value: "{palette.ocean.500}"}
  accent:     {$value: "{palette.amber.500}"}
  surface:    {$value: "#FFFFFF"}
  background: {$value: "{palette.neutral.50}"}
  text:       {$value: "{palette.neutral.900}"}
```
```yaml
# tokens/component/button.tokens.yaml      tier 3: only when a component needs its own token
button:
  background: {$value: "{color.primary}", $type: color}
  radius:     {$value: "{radius.md}",     $type: dimension}
```
```yaml
# tokens/modes/dark/color.tokens.yaml      a mode overrides semantic tokens only
color:
  $type: color
  background: {$value: "{palette.neutral.900}"}
  text:       {$value: "{palette.neutral.50}"}
```

The output names carry the neutral prefix: `--bos-color-primary`, `--bos-palette-ocean-500`, `--bos-spacing-md`; a mode is a `[data-theme="dark"]`
block (and optionally a `prefers-color-scheme` block). **Components use only semantic and component tokens**, so a brand change touches no
component.

**Resolution order** (the earlier bos's rule, kept, and every step fails open): the app's `tokens/` ← the selected theme preset ← the framework's
default preset. An empty `brand/` therefore still yields a working, accessible site. Layouts and renderers declare in their `meta.ts` which
roles they use; `bos check` reports a role that is used and not defined, and a token nothing uses.

**Presets.** Four kinds, each a file under `presets/`: `palette`, `typography`, `scale`, and `theme` (a bundle of the other three plus modes). The
framework ships a small neutral set; an app file with the same id overrides it. A preset is data; it contains no code.

**Plugins** extend the pipeline at three points, which are the hook points of Style Dictionary: a **transform** (change a token's value, name or
attributes), a **format** (write an output file), and a **validator** (report problems). A plugin is one TypeScript file in `plugins/`,
discovered by folder ("add a file, restart"), exporting `definePlugin({kind, name, …})`. The framework's own transforms, formats and validators
(hex to color object, rem sizes, font stacks, name with prefix; CSS variables, TypeScript module, JSON; schema, alias cycles, contrast) use the
same interface. **Only built-ins exist at first**: the `plugins/` folder appears with the first app plugin, and the mechanism is not built
further until a real one needs it.

**Engine.** Style Dictionary is the candidate pipeline (standard token format, transforms, formats and hooks; it accepts the W3C structure). The
first branding milestone starts with a short spike: adopt it unless it cannot reproduce the baseline brand's computed output, in which case a
small pipeline of our own with the **same file formats and plugin interface** replaces it. The formats and the plugin interface above are the
contract; the engine is replaceable.

**Basic branding** is the logo and the colors (palette and roles, light and dark logos, favicon); typography, radius, spacing, shadow and motion
follow in the same structure.

#### Importing an existing brand

An app that already has a brand does not retype it. A **one-shot importer that lives in the app's repo** (never in the framework) reads the
existing tokens (CSS custom properties, a brand file), resolves every alias chain to a final value, classifies each token into the tiers
above, and writes the token sources **and a mapping file**, `site/brand/tokens/_import/mapping.yaml`: one row per old token, with its new path,
its tier, its resolved value and a status (imported, alias, derived, deferred, or not-a-token). The rules, and what the framework provides for them:

- An old alias that only themes a component library the new app does not use is recorded as an **alias**, not re-emitted.
- A **derived token** (a color mixed with transparency, a tint) is declared as a base token plus a percentage in `$extensions.bos`, and a built-in
  transform emits the `color-mix`, so a brand change still flows through.
- A value a component takes **from data** (a hero's background chosen by an editor) is a **renderer setting**, not a token.
- A token whose component has not arrived yet is **deferred**: kept, and exempt from the unused-token check until the component exists.
- **Raw values are refused.** `bos-web check` fails on a literal color, size, radius, shadow or duration in a framework or app component's styles,
  and on a reference to a token that does not exist, so a component can only be as branded as its tokens allow.
- Verification is by **value**: the computed value of every old token in the running old app equals the computed value of its mapped new token.

### The kind-folder convention

Every configurable kind follows the same rule as `brand/`: **one folder, holding its config, its source files, its presets and its plugins,
each sub-folder created only when its first item exists.** Presets live beside the kind they preset, not in one shared bucket. An app item
with the same id as a framework item overrides it. Generated output goes to `.bos/`, never into `site/` or `apps/`.

| Kind | App folder | Holds | Framework counterpart |
|---|---|---|---|
| Site | `site/` | `site.yaml`, `nav.yaml`, `presets/` (site-level bundles: default layout, navigation skeleton, block presets) | `bos-svelte/src/lib/site/` |
| Brand and tokens | `site/brand/` | the tree above | `bos-svelte/src/lib/theme/` |
| Pages | `site/pages/` | page files, `_layout.yaml`, `_partials/`, `_presets/` (page skeletons for the generator) | `.../lib/content/` (the loader) |
| Content | `site/content/<type>/` | entries as Markdown or YAML | |
| Media | `site/media/` | images and files that pages and entries refer to | |
| Types | `site/types/` | content types | `bos-go/core/content/` |
| Blocks | `site/blocks/` | `presets/` (block presets) | `.../lib/renderers/` |

### 4.7 Assets and static delivery

Section 4.6 says where the **sources** live (`site/brand/assets/` and `site/media/`). This section is the other half: how they are **delivered**.
The rule: **sources are edited in `site/`; the delivery is generated into `.bos/build/static/` and `.bos/build/assets.json`; nothing is copied
into `apps/web/static/` by hand.** The web build serves the generated folder as its static folder (SvelteKit's stable-URL folder, pointed at
the generated output; the exact setting is confirmed in the first branding milestone).

#### The asset map

Every asset has an entry in a generated, typed **asset map**: its id, its source, its output URL, its class (below), its variants, its cache
policy, and who references it (the document head, a stylesheet, the manifest, structured data, a social card). `bos-web check` fails on a
reference to a missing asset, an asset nothing references, two assets with the same URL, and a **stable file that would shadow a route** (a
static file wins over a route in the Node adapter, so a route and a file must never share a URL).

#### Two URL classes

| Class | What | URL | Cache |
|---|---|---|---|
| **Stable** | files that a browser, a crawler or another site requests by a fixed name: `/favicon.ico`, `/apple-touch-icon.png`, `/manifest.webmanifest`, the default social image | fixed | short (revalidated); when a change must bust caches, the asset moves to a versioned path and the stable URL redirects or is regenerated |
| **Hashed** | everything the build itself references: fonts, logo images and photographs in pages, stylesheets | the file name carries a content hash | `public, max-age=31536000, immutable` |

`robots.txt`, `sitemap.xml` and `llms.txt` are also stable URLs, but they are generated **routes**, not files.

#### What the browsers and crawlers require

The pipeline is built to satisfy these, and each is a test (the file exists, has the right size, format and alpha channel):

| Need | Why |
|---|---|
| `/favicon.ico` at the origin root, with 16 and 32 pixel images | some browsers (Safari among them) request it blind and **ignore** a `<link rel="icon">` set at run time |
| a PNG icon, and no SVG-only favicon | Safari cannot render an SVG icon; modern browsers also accept an `icon.svg` |
| `apple-touch-icon.png`, **180 × 180, opaque (no alpha)** | the iOS home-screen and bookmark icon; transparency renders black |
| manifest icons **192 and 512**, plus a maskable 512 | Android and installable-app icons |
| absolute, public, cookie-free URLs for the social image and the structured-data logo | crawlers and unfurlers fetch them outside the site; the default social image is 1200 × 630 |
| fonts from the **same origin**, `font-display: swap`, a preload with `crossorigin` for the critical faces | a font served from another origin needs CORS; a swap avoids invisible text |
| a logo with a raster fallback where an SVG cannot be used | favicons and social images cannot use CSS to tint or resize an SVG |

#### Conventional names and derivation (the Next.js file-metadata idea)

A file dropped into `site/brand/assets/` **with a conventional name is wired automatically and wins**: `favicon.ico`, `icon.svg`, `icon.png`,
`apple-touch-icon.png`, `og-default.png`, `manifest.webmanifest`. Anything not supplied is **derived from `logo.mark`**:

| Output | Derived as |
|---|---|
| `favicon.ico` | multi-size, 16, 32 and 48 |
| `icon.png` | 64 × 64 |
| `icon.svg` | copied, when the mark is an SVG |
| `apple-touch-icon.png` | 180 × 180, the mark on an opaque background (`color.primary` unless `brand.yaml` says otherwise) |
| `icons/icon-192.png`, `icon-512.png`, `icon-maskable-512.png` | from the mark, with the maskable safe zone |
| `og-default.png` | 1200 × 630, the mark centered on the background color |
| tinted light and dark logo rasters | for the surfaces where CSS cannot tint (favicons, social images) |

#### Logo

Variants are `mark`, `wordmark`, `light` and `dark` (SVG preferred, PNG allowed). The `Logo` atom picks by mode, with explicit `width` and `height`
so nothing shifts. **One resolver in the framework** answers "which logo URL does the site render", and one answers "the same, absolute" for
structured data and social cards; no component inlines its own fallback, and a test fails the build if one does.

#### Fonts

A typography token names a family; the family's **source** is either local files (`assets/fonts/<family>/`) or a font-package id in a typography
preset (an open-licensed family from a package such as Fontsource). The pipeline copies the `woff2` files to hashed URLs, generates the
`@font-face` rules (weights, styles, `font-display: swap`, `unicode-range` when the source has subsets), emits the preload for the faces marked
critical, and builds the fallback stack from the tokens. **Every family needs its license file next to it**, and `bos-web check` fails without
one. **No font host is ever contacted at run time or in a test**; acquiring the files (a package install, or a copy) is a deliberate, approved
setup step, never a build or test step.

#### Graphics, images and icons

- **Photographs and graphics** live in `site/media/`. The originals are kept; the build emits **responsive variants** (a few widths, AVIF and WebP
  with a fallback) under hashed names, and the `Image` atom renders `srcset`. `alt` text is required unless the image is declared
  `decorative`, and `bos-web check` enforces it.
- **Icons** are drawn by an `Icon` atom from a **bundled** icon set (no run-time fetch) or from the app's own SVGs in
  `site/brand/assets/icons/`, by name.
- The image and icon processing engine (a Node library or the Go side) is decided by a short spike; the map and the naming do not depend on it.

#### The head is generated

The framework's root layout writes the icon links (with their `sizes` and `type`), the apple-touch link, the manifest link, the `theme-color`
(from `color.primary`), the font preloads and the default social-card tags **from the asset map**. The app's own HTML template contains only
the framework's placeholder.

#### Development, build, and later uploads

`bos dev` builds the assets on start and watches `site/brand/assets/` and `site/media/`; `bos build` runs the asset build before the web
build; both serve or ship `.bos/build/static/`. In the **dynamic phase** an asset gains a second kind of source, `store` (uploaded through the
admin into object storage): the same derivation (favicon set, apple-touch icon, tinted logos) then runs on upload in `bos-go`, into versioned
paths, and `/favicon.ico` becomes a route that redirects to the current baked favicon, with the static default as its fallback: the same
static-to-dynamic step as content (section 7).

```
.bos/build/static/                 generated; git-ignored; served at the site root
├── favicon.ico  icon.png  icon.svg  apple-touch-icon.png  manifest.webmanifest  og-default.png
├── icons/                         icon-192.png · icon-512.png · icon-maskable-512.png
├── fonts/                         <family>-<weight>.<hash>.woff2
└── img/                           <name>.<width>.<hash>.{avif,webp,jpg}
.bos/build/assets.json             the asset map
```

The framework side is `bos-svelte/src/lib/assets/`: `collect/`, `derive/` (favicon, apple-touch, manifest icons, social card, tints), `fonts/`,
`images/`, `headers/` (the two cache policies), `head/` (the link and meta generator), `map/` (the asset-map types and checks). Its extension
points are the same three as the token pipeline (a transform, a derivation, a validator), found in `site/brand/plugins/`.

## 5. The atomic component structure

Components are organized by Atomic Design and live in the framework, with the app adding its own beside them.

| Level | What it is | Registered as a renderer? | Examples |
|---|---|---|---|
| **atom** | one element, primitives only | no | `Text`, `Heading`, `Button`, `Link`, `Image`, `Icon`, `Logo`, `Badge` |
| **molecule** | a few atoms that work together | no | `Card`, `NavItem`, `Breadcrumb`, `MediaObject`, `FormField` |
| **organism** | a self-contained section; may bind to a content shape | **yes** | `Header`, `Footer`, `Hero`, `Prose`, `FeatureCards`, `Steps`, `Accordion`, `CardGrid`, `ContactBlock` |
| **layout** (template) | slots, regions, widths | **yes** (as a layout preset) | `default`, `landing`, `legal`, `listing`, `article`, `minimal` |
| **page** | a file | — | `site/pages/*.md` |

One component is **one folder**, with its metadata beside it and derived from the same declaration:

```
organisms/Accordion/
├── Accordion.svelte        the component; its props type comes from meta.ts
├── meta.ts                 defineRenderer({id, level, accepts, settings, presets, views})
└── Accordion.test.ts       renders it with a fake source
```

`meta.ts` is the single definition of the renderer's inputs. The component imports its props type from it; the registry, `bos check` and
the generator read the same object. A declaration nothing reads cannot exist, because there is nothing to declare separately.

The first organisms are exactly the ones the old app already uses (a hero, prose, feature cards, steps, an FAQ accordion, card lists, a
header and a footer driven by the site configuration), so the library starts as small as the baseline needs.

## 6. File structures

### 6.1 The frameworks (mirroring agent-framework)

```
platform/go/frameworks/bos-go/                     module github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go
├── core/                     pure logic and interfaces; no net/http, no database; fakes included
│   ├── config/               bos.yaml and environment loading, validation (the running API's loader, its constants made configurable)
│   ├── site/  brand/         the site and the brand, typed
│   ├── content/              content types, entries, the Source interface (the "base")
│   ├── page/                 page resolution over a Source
│   ├── preset/               the preset registry
│   └── registry/             module discovery (folder or explicit)
├── adapters/                 implementations of core interfaces, named for what they are
│   ├── filesource/  memsource/           (later) pgsource/
├── server/                   the HTTP app; depends on core, never the reverse
│   ├── app.go                NewApp(cfg, deps) http.Handler: the fixed middleware chain, then routes
│   ├── middleware/           request id, logger, recoverer (moved from the running API); security headers, CSRF, auth: the ported identity/auth, imported, not copied
│   └── routes/               health, site, pages, content
├── cmd/bos/                  the CLI: new · g · dev · build · check (the Go half) · test · bundle; web-side commands are delegated to bos-web
└── tests/                    added after the core is validated (D26); flat, with fakes; no network

platform/svelte/frameworks/bos-svelte/             package @sbx/bos-svelte
├── src/lib/
│   ├── atoms/  molecules/  organisms/  layouts/   one folder per component (section 5)
│   ├── renderers/            the registry: folder discovery, compatibility checks
│   ├── content/              the loader for site/pages and collections; sources: inline, file, api, fake
│   ├── site/                 site configuration and the content types
│   ├── theme/                the token pipeline and its defaults (below)
│   ├── assets/               the asset pipeline: collect, derive, hash, copy, head wiring (section 4.7)
│   ├── routes/               the catch-all page route and the sitemap, robots and llms.txt endpoints
│   └── server/               hooks: the /api pass-through, silent refresh, security headers
├── bin/bos-web               the web-side CLI: check · tokens build · g renderer (the bos CLI delegates to it)
└── tests/                    added after the core is validated (D26)
```

The token pipeline inside the framework (the counterpart of `site/brand/`):

```
bos-svelte/src/lib/theme/
├── pipeline/       the engine wrapper: read the YAML sources, normalize to the W3C structure, resolve aliases and modes, build
├── transforms/     built-in: hex to color object, rem sizes, font stacks, name with prefix
├── formats/        built-in: CSS variables, TypeScript module, JSON
├── validators/     built-in: schema, alias cycles, undefined or unused roles, contrast pairs
├── presets/        the framework's palette, typography, scale and theme presets: a small neutral set, with the same file shapes as the app's
├── defaults/       the default tokens every site falls back to (fail-open)
├── schema/         schemas of brand.yaml and of the token subset, for `bos-web check` and editors
└── index.ts        definePlugin, the plugin loader (folder discovery)
```

`core` never imports `server`. Every adapter has a `base` interface in `core` and an implementation beside it. Both frameworks are units with
records and rendered READMEs from the start (the unit model), so they are documented like every other package.

**Names and links (D24).** Every module and package path is inside this repository: `bos-go` is the Go module
`github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go` (the repository path plus the directory), and `bos-svelte` is `@sbx/bos-svelte`
(the workspace's own scope, like the other `@sbx/*` packages). Only client projects have repositories of their own. Inside our environment
nothing is fetched: a consumer links the frameworks and packages **by local directory**. Go: `use` lines in the consumer's `go.work` and, in
each of its `go.mod` files, a `require` with a `replace` to a relative path for the framework's **whole in-repo closure** (a dependency's own
`replace` lines do not propagate, and a `require` without one makes the go tool look the module up on the network). Web: the consumer's pnpm
workspace file lists the framework package. The gates run with `GOFLAGS=-mod=readonly` and `GOPROXY=off`, so a missing link fails at once.

**Extracted, not rewritten (D25).** Code that already runs is not written again. The first content of `bos-go` is the running API's own code,
moved into the framework and given configuration where it had constants: the configuration loader, the fixed middleware chain, the JSON
recoverer, the health and root routes, graceful shutdown. Security headers, CSRF and authentication are the ported `identity/auth` package,
imported as it is. New code is written only where nothing runs today (pages, presets, tokens, assets, the registry, the CLI). The extraction
is proven by the baseline's golden files; tests are added after the core is validated (D26); and moved code carries no consumer names: what was named after the
product becomes a configured value.

### 6.2 An app: what `bos new` creates

```
<app>/
├── bos.yaml                  the link between the two halves: name, profile, api {port, cors_origins}, web {port, api_url}, site dir
├── site/                     WHAT THE SITE IS  (data; one folder per kind, section 4.6)
│   ├── site.yaml  nav.yaml  presets/
│   ├── brand/                brand.yaml · tokens/ · presets/ · assets/ · plugins/
│   ├── pages/                the router (section 4.2)
│   ├── content/<type>/       collections as Markdown or YAML files
│   ├── media/                images and files that pages and entries refer to (brand assets are in brand/assets/)
│   ├── types/                content types the app adds
│   └── blocks/presets/       block presets
├── .bos/                     generated output, git-ignored and never committed: build/tokens/, build/static/ (served at the site root), build/assets.json
├── apps/
│   ├── api/main.go           about 15 lines: bos.Run(cfg) · product/ (the app's own Go, behind interfaces the framework defines)
│   └── web/                  svelte.config.js · vite.config.ts · src/{app.html, components/ (the app's own renderers, discovered), routes/ (empty until an override)}
├── deploy/                   generated: docker-compose.yml, .env.example
└── Makefile                  make dev | build | test | check | bundle-up | bundle-down
```

Adding a page is adding a file. Adding a renderer is adding a folder under `apps/web/src/components/`, restarting, and referring to it by id
("add a folder, restart"). Changing the brand is editing `site/brand/`. There is no generated code in the tree to drift.

### 6.3 Generators (the NestJS idea)

`bos new <app>`; `bos g page legal/terms --layout legal`; `bos g collection faq`; `bos g renderer accordion --level organism`;
`bos g preset faq-home --for accordion`. Each generator writes ordinary files a person could have written, and refuses a name that is taken.

### 6.4 What `bos check` validates (in seconds)

It has two halves, because the renderer metadata is TypeScript and the content types are read by both stacks. The **Go half** (`bos check`)
validates `bos.yaml`, `site/site.yaml` and `nav.yaml`, the content types, every collection against its type, and the assets. The **web half**
(`bos-web check`) validates every page's layout, slots and blocks (compatibility, section 4.3), that every renderer id used exists, internal
links, and the brand (the token schema, alias cycles, undefined or unused roles, the contrast pairs). `bos check` runs both, and `make check`
calls `bos check`. Each half reads the same metadata its runtime reads.

## 7. Static first, dynamic later

| Phase | Pages live in | Collections read from | Editing |
|---|---|---|---|
| **Static** | files | files (`file` source) | by editing files |
| **Dynamic reads** | files | the API (`api` source, `bos-go` serving its own source) | by editing files |
| **Dynamic writes** | the database (`db` source for the `page` type) | the database | by an admin; the files become **seed**, applied to a fresh environment only |

The model, the pages and the blocks are identical in all three. The seed rule is the design's (6.6): existing data is never re-seeded, and
editors' changes win after the first boot.

## 8. Dynamic features: kits and modules

A feature that needs a server (authentication, an admin, uploads, a property catalog) is a **kit**, exactly as in the design (section 3): a
family of packages plus its app-facing surface. That surface, derived from the old app and proven by the first dynamic milestone:

- A **Go module** declares route groups (path, authentication, permission, body cap, limiter, and a `Register` function); `bos-go` applies
  the gating in one fixed order and owns the middleware chain. The module also declares its migrations, permissions and jobs.
- A **Svelte manifest** declares routes, admin navigation entries, surfaces (for the sitemap and navigation) and block renderers.
- **Spec keys** and **needs** (services, environment names, jobs) are declared in the kit's manifest record, and `process-cli` validates
  `bos.yaml` against the schema composed from the enabled kits.
- **Configuration levels follow NestJS**: framework root (`bos.yaml`), kit (`kits.<name>`), block (`settings`).
- A setting read by both stacks (token lifetimes) has one key in `bos.yaml`.

## 9. Decisions and open questions

- **D21 (proposed).** The constructor model of sections 4 to 7, with the rule "code declares, data selects".
- **D18 (revised).** Kits expose the surface of section 8; it follows the constructor's milestones and is proven by the first dynamic one.
- **D15.** Where a kit's Go and Svelte wiring lives: unchanged, lean (b), decided by the first dynamic milestone.
- **D22 (proposed).** Brand and tokens (section 4.6): one folder, `site/brand/`, holding the config, W3C-structured YAML token sources in three
  tiers plus modes, presets, assets and plugins; generated output is never committed; every kind follows the same one-folder convention.
- **D23 (proposed).** Assets and static delivery (section 4.7): sources in `site/`, delivery generated into `.bos/build/static/`, an asset map, two URL
  classes (stable and hashed), conventional names win and the rest is derived from the logo mark, fonts self-hosted with a license file each, no
  external host ever contacted, the head generated from the map.
- **D24 (proposed).** Names and links (section 6.1): every module and package path is inside the sdlc-kit repository
  (`github.com/shredbx/sdlc-kit/platform/<lang>/...`, `@sbx/*`); only clients have repositories of their own; consumers link by local directory,
  and nothing is fetched.
- **D25 (proposed).** Reuse before writing (section 6.1): a ported package is imported as it is; running code is moved into the framework and
  configured, never rewritten; new code only where nothing runs today; a dependency such as a cache is a service record and a bundle.
- **D26 (proposed).** Tests come after the core: none are written while the core is built; the gates are the offline build and vet, the
  golden-file comparison and `bos check`; tests are added, as scopes of their own, once the core is built and validated.

| Open | Closed by |
|---|---|
| The layout metadata format (slots, allowed blocks) | the first two layouts (milestone M2) |
| How `meta.ts` becomes a Svelte props type without a build step | the first renderer |
| Whether content types are generated into Go and TypeScript types or validated at run time | the first collection (M3) |
| The neutral token prefix | M1 |
| The pipeline engine: Style Dictionary or a small pipeline of our own with the same formats and plugin interface | the M1 spike |
| The image and icon processing engine (a Node library or the Go side); the exact setting that points the web build's static folder at the generated output | the M1b spike |
| Whether the ported packages' module paths (still `github.com/shredbx/sbx-core/pkg/*`, D7) move into the sdlc-kit paths now | the decision before M0.1 |
