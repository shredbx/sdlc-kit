# shredbx: bestierealestate domain map, the "13 capabilities" search, and the bestays-web Next.js environment

Last updated: 2026-09-23
tags: shredbx, bestierealestate, capabilities, research

> Scope note: this is a deep-dive follow-up to the earlier `docs/research/shredbx-codebase-inventory.md`
> and `docs/research/shredbx-sdlc-system.md` passes. It answers three specific questions the user
> raised: (1) what in bestierealestate is genuinely real-estate-specific vs. already generic and
> reusable, (2) whether shredbx has an established "13 capabilities" list, and (3) what a Next.js
> bootstrap matching bestays-web's environment would need to replicate. Research only — nothing
> built or ported.

---

## Part 1 — bestierealestate domain inventory

Target: `/Users/solo/Projects/workspaces/shredbx/clients/bestie/projects/bestierealestate`

### What the product says about itself

- **`CLAUDE.md`**: "BR = the Real-Estate Configuration of SBX Fabric." Enabled modules: `product(→property)
  · content · inquiry · identity · analytics · seo · media · automation · canvas · backup · faq`.
  Architecture is stated explicitly as a **thin Chi consumer (99% rule)** — `apps/api-chi/` is wiring;
  domain logic lives in shared packages (`pkg/property`, `pkg/auth`, `pkg/rbac`, `pkg/image`, `pkg/cms`,
  `pkg/faq`, …). Role-based access only — **BR is NOT SaaS, no ownership model** (this matters: it means
  the RBAC/auth model here is simpler than a true multi-tenant SaaS product would need).
- Data-model invariants worth carrying forward as generic patterns regardless of vertical: money stored
  as integer minor-unit (satang) never float; dictionaries as VARCHAR(50) code tables, not DB ENUMs;
  units-of-measure kept in code, not DB; media URLs UUID-based (never slug); one `images` table per
  schema with joins for galleries, no JSONB blobs, no width/height columns in the images table itself.
- **`ROADMAP.md`** and **`HANDOFF.md`** show the product mid-build: gallery/media work, `pkg/property`
  enrichment, title-deed normalization, Supabase→BR data migration pipeline, a scheduler subsystem,
  the "smart collection" catalog feature, and public listing pages (`/properties`, `/properties/[id]`,
  `/for-sale`, `/for-lease`). Deal/transaction modeling: `for_sale`/`for_lease` are booleans (offering),
  while deal kind (sale | lease) is a separate code enum in `pkg/transaction` — dual tenure is handled
  as two separate deals, not a combined state.
- **`DEPLOY.md`**: confirms the mirror-only deploy model (BR ships from its own mirror repo, not the
  monorepo), a backup→migrate→start one-shot chain gating every deploy, and a DB-driven universal
  scheduler (no in-process ticker; external Dokploy Schedule Jobs tick a `scheduler_jobs`/`scheduler_runs`
  is-due gate) — this scheduler design is itself a generic, reusable pattern, not real-estate-specific.

### Go API (`apps/api-chi`) — shared packages actually imported

Grepping every `.go` file under `apps/api-chi` for `github.com/shredbx/sbx-core/pkg/*` imports gives the
full, verified list of shared packages BR actually consumes (not just what main.go wires at startup):

```
address · auth · calendar (+ical) · cms · collection · contact (+vcard) · database · dictionary ·
faq · feed · geocoordinate · httputil · image · inquiry · language · money · notify · phonenumber ·
property · rbac · repository (+postgres) · rss · scheduler (+schedcli) · seo · socialnetwork ·
transaction · user · video · visitoractivity
```

This is meaningfully different from the candidate list the prior research pass flagged as plausible.
Notably:
- **`pkg/person` is NOT used** — BR uses `pkg/user` (account/auth identity) + `pkg/contact` (CRM contact
  record) instead of a unified person concept. This is a signal for the generic-BOS design: "person" as
  imagined upstream may not be the right shape; user/contact as two separate concerns is what's actually
  proven in production.
- **`pkg/i18n` is NOT used** — BR rolls its own `internal/localization` package for dynamic-content
  translation validation (keyed by ISO-639-1 locale, using `golang.org/x/text` via `pkg/language`, which
  *is* shared). `pkg/i18n` as a package name may be aspirational/unused; `pkg/language` is the proven
  piece.
- **`pkg/lease` and `pkg/leaseparticipant` exist in core but are NOT used** — BR modeled lease as a value
  of `pkg/transaction`'s deal-kind enum (sale | lease) rather than a separate lease entity/package. This
  is worth flagging directly to whoever owns `pkg/lease` upstream: it may be speculative, unconsumed by
  the one production-realistic vertical that would need it.
- **`pkg/completeness`, `pkg/csvimport`, `pkg/storage` all exist in core but are NOT imported anywhere
  in BR** — same caveat: these are unproven-in-production candidates, not confirmed-generic packages.
- **`pkg/faq`, `pkg/inquiry`, `pkg/scheduler`, `pkg/seo`, `pkg/socialnetwork`, `pkg/calendar` ARE used**
  — these are the prior pass's generic candidates that BR actually confirms in production.

### Go API — BR-local (non-shared) domain packages

Everything under `apps/api-chi/internal/` that is NOT a thin pass-through to a shared package. Each was
sampled by reading its package doc-comment:

| Package | What it is | Domain-specific or generic-but-not-yet-extracted? |
|---|---|---|
| `internal/agent` | The property Agent/broker entity — card on listing detail + agent directory. Explicitly documented as "BR-project-scoped, not a shared sbx-core primitive." | **Real-estate-specific** (though the underlying shape — a staff member with a public-facing card, distinct from the login/RBAC user — generalizes to "staff directory" for many verticals) |
| `internal/areageo` | Geocoded centroid cache for province/district/sub-district, used to show an *approximate* public map pin without exposing exact property location. | **Real-estate-specific** (privacy-preserving geo display is the real estate use case; the underlying "coarse geocode cache keyed to an admin area" pattern is generic) |
| `internal/deal` / `pkg/transaction` | Party-role policy engine for a "deal" (buyer/seller/lessor/lessee roles per deal kind/facet). | **Domain-specific object (real-estate transaction), generic mechanism** (facet→role policy is a reusable pattern for any multi-party transaction type) |
| `internal/canvas` | Design-canvas document editor + proxy/auth for BR agents authoring watermark designs. | **Generic tool wearing a real-estate skin** — canvas document editing itself is not real-estate-specific; it's currently scoped to one use (watermark design). |
| `internal/propertyfilter` | The canonical catalog filter language — ONE filter expression/predicate builder shared by public listing search, smart-collection resolution, and admin list management. | **Generic pattern, real-estate-named** — "catalog filter language with lenient public parsing + strict admin parsing, feeding one predicate builder" is a reusable e-commerce/catalog pattern; only the field vocabulary (property_type, price, etc.) is domain-specific. |
| `internal/newsfeed` | BR-specific wiring of the shared `pkg/rss`/`pkg/feed` engine for a news module. | **Generic (news/content feed), thin local wiring only** |
| `internal/watermark` | Publishes a canvas design as a registry entry the media pipeline stamps onto property images. | **Real-estate-adjacent use case (protecting listing photos), generic mechanism** (image watermarking is any-vertical-applicable) |
| `internal/registry` | BR-specific catalogue/dictionary wiring — "the shared pkg/dictionary doesn't know which BR tables reference which dictionaries; that knowledge belongs here." | **Structurally generic, necessarily per-project** — every vertical needs this same kind of table↔dictionary binding registry. |
| `internal/siteconfig` | Singleton branding/header-nav config backing admin Appearance editors + public header/footer render. | **Fully generic** — applies to any site, no real-estate content. |
| `internal/page` | Published district/property-type filter option builders for public location browsing (province▸district). | **Real-estate-specific presentation logic** over generic dictionary/location data. |
| `internal/qr` | Server-side QR code generation for agent contact links (wa.me etc.), with a strict URL-scheme allowlist for security. | **Fully generic** utility. |
| `internal/document` | System document CRUD + anyone-with-link share/revoke. | **Fully generic** (document sharing, not real-estate-specific). |
| `internal/export` | Pure serializer: entity records → CSV/Markdown export tables, explicitly no DB/HTTP deps, hard invariant that no internal UUIDs leak into exports. | **Fully generic** admin-export utility. |
| `internal/media` (youtube_metadata.go) | BR-specific, key-gated YouTube Data API enrichment (duration/view-count) layered on top of the shared keyless `pkg/video` oEmbed resolver. | **Generic mechanism (video metadata enrichment), narrowly BR-keyed today** |
| `internal/autotag` | Turns dictionary values into display hashtags (`"Koh Phangan"` → `"#KohPhangan"`) for SEO/marketing; explicitly NOT the match/search key. | **Fully generic** text-formatting utility. |
| `internal/imageimport` | Server-to-server remote image fetch with SSRF hardening (scheme allowlist + per-redirect-hop IP re-validation). | **Fully generic**, valuable security pattern worth lifting into shared core regardless of vertical. |
| `internal/feedcache` | Redis adapter for the shared `pkg/rss.Cache` interface, with in-memory fallback when Redis isn't configured. | **Fully generic** infra adapter. |
| `internal/assistantchat`, `internal/aiassistant`, `internal/assistantcapabilities` | Public AI-chat session tokens, an AI-assistant settings singleton (system prompt, model list, enabled tools, knowledge sources), and a registry-driven tool-discovery descriptor surface consumed by an agent. | **Fully generic** — an AI-assistant subsystem with a governance-YAML-mirrored capability descriptor. Notable as a whole subsystem that looks like a candidate for its own shared package, not BR-only. |
| `internal/localization` | Dynamic per-row content translation validation (ISO-639-1 gate on `property.Translations`-style JSONB fields). | **Structurally generic (any translatable-content model), currently coupled to the `property` table name in its examples** |
| `internal/config` (rbac.go) | Project-owned role constants (`manager`, `super-admin` — BR narrowed from the framework's illustrative defaults). Explicit design note: every project should declare its own role names locally. | **This is itself a documented generic PATTERN** ("project-owns-role-names") more than a real-estate concept. |

### Svelte web app (`apps/web-svelte`) — product surface

Full route-group structure under `src/routes/`:

- **`(public)`** — the customer-facing site: `/`, `/properties` (list), `/properties/[slug]` (detail),
  `/properties/for-sale`, `/properties/for-lease`, `/properties/type/[type]`, `/properties/area/[province]`
  and `/properties/area/[province]/[district]` (geo drill-down), `/properties/collections/[slug]`
  (smart/curated collections), `/search`, `/sell` (seller lead-gen), `/services` + `/services/[slug]`,
  `/guides` + `/guides/[slug]`, `/faq`, `/about`, `/contact`, `/p/[id]` (short-link redirect), plus
  `/privacy`, `/terms`, `/cookies`.
- **`(minimal)`** — `/login`, `/auth/callback` (magic-link / token auth flow), no site chrome.
- **`(admin)/admin`** — platform-operator surface: `/admin` dashboard, `/admin/users` + `/admin/users/[id]`,
  `/admin/audit`, `/admin/backup`, `/admin/reset-requests`, `/admin/schedules`.
- **`(admin)/manage`** — the content/business-operator surface, the largest route tree by far:
  - `manage/properties` (+ `[id]/{edit,overview,dashboard,media,seo,units,transactions,inquiries,
    activity,analytics,information}`), `manage/properties/new`
  - `manage/content/properties/{agents,analytics,collections,dashboard,listing,locations,media,
    settings,settings/tags,transactions}` — a second, content-oriented property admin surface layered
    over the same domain
  - `manage/agents`, `manage/contacts` (+ `list`, `new`, `[id]/edit`, `settings`) — CRM
  - `manage/inquiries` (+ `[id]`, `page`) — inbound lead/CRM inquiry handling
  - `manage/transactions` (+ `[id]`, `new`, `batch`) — deal/transaction admin
  - `manage/calendar` + `[id]`
  - `manage/faq` (+ `analytics`, `content`, `design`, `seo`)
  - `manage/guides` (+ `analytics`, `design`, `listing`, `new`, `seo`, `settings`, `[id]/edit`) —
    content marketing
  - `manage/services` (+ `analytics`, `design`, `new`, `page`, `seo`, `[id]/edit`) — a second content type
    parallel to guides
  - `manage/news` (+ `categories`, `items`, `sources`)
  - `manage/pages` (+ `[surface]/{information,seo,analytics}`, `appearance/{header,footer,theme}` each
    with their own `analytics`/`seo`) — generic CMS page + site-appearance editor
  - `manage/marketing/watermarks` (+ `[id]`)
  - `manage/tools/media-canvas`
  - `manage/ai-assistant` (+ `chat`, `models`, `sources`, `tools`, `usage`, `welcome`)
  - `manage/analytics/website`, `manage/documents`, `manage/messages`, `manage/social-media`,
    `manage/tags`, `manage/seo`, `manage/account`, `manage/content/dictionaries/[name]`
- Root-level SEO/infra routes: `llms.txt`, `robots.txt`, `sitemap.xml`, `favicon.ico`.

### Domain-specific vs. already-generic split (the deliverable value)

**Real-estate-only concepts** (would need genuine domain-specific redesign for another vertical):
- Property/listing model itself (bedrooms, land units, title deeds, property types, transaction offering
  booleans)
- `internal/agent` (listing agent/broker card — though the underlying "staff directory" shape
  generalizes)
- `internal/areageo` (coarse public geocode privacy pattern) and the area/province/district drill-down
  browsing surface
- Smart collections *as a catalog concept* (the resolution mechanism in `internal/propertyfilter` is
  generic; the notion of a curated real-estate collection like "Beachfront Villas" is domain content)
- Similar/related-properties ranking rules (`property_similar.go`, `property_related.go`) — explicitly
  documented as BR PRODUCT decisions, not shared-package logic
- Lease/sale deal-kind vocabulary in `pkg/transaction`
- Title-deed normalization, land-area unit conversions

**Already-generic concepts** (proven in production, would work for almost any business vertical
essentially as-is):
- `pkg/auth`, `pkg/rbac`, `pkg/user` (+ BR's "project owns its role names" pattern)
- `pkg/contact` (CRM contact, distinct from login user) + `pkg/inquiry` (lead/CRM inbox)
- `pkg/money` (integer minor-unit pattern), `pkg/address`, `pkg/geocoordinate`, `pkg/phonenumber`
- `pkg/dictionary` (code-table catalogue engine) + BR's local `internal/registry` binding layer
- `pkg/faq`, `pkg/seo`, `pkg/socialnetwork`, `pkg/calendar` (+ ical export)
- `pkg/scheduler` (DB-driven, externally-ticked, is-due-gated cron — a strong generic pattern)
- `pkg/rss`/`pkg/feed` (+ `internal/feedcache` Redis adapter) — generic content-feed engine
- `pkg/image`/`pkg/video` (+ the watermark registry mechanism, + SSRF-hardened `internal/imageimport`)
- `internal/siteconfig` (branding/nav singleton), `internal/document` (share-by-link), `internal/export`
  (CSV/Markdown table serializer), `internal/qr`, `internal/autotag`
- The AI-assistant subsystem (`aiassistant`/`assistantchat`/`assistantcapabilities`) — a full,
  governance-YAML-driven tool-discovery + chat config surface with zero real-estate coupling
- The CMS surface: `manage/pages`, `manage/guides`, `manage/services` (two parallel generic
  "content type with analytics+SEO+design tabs" admin patterns) and `manage/news`
- The scheduler/backup admin UI (`/admin/schedules`, `/admin/backup`) and its universal cron-control
  design

This split is the actionable map: everything in the second list is a strong candidate to lift into a
generic BOS core as-is or near-as-is; everything in the first list needs a real domain redesign pass
before it can serve outside real estate, though several (propertyfilter's catalog-filter language,
deal's facet→role policy, canvas, areageo's coarse-geocode pattern) are *generic mechanisms* that just
happen to be named/scoped for one vertical today and would generalize cheaply.

---

## Part 2 — the "13 capabilities" search

**Verdict: found verbatim, exact count of 13, single authoritative source.**

Source: `/Users/solo/Projects/workspaces/shredbx/.sbx/workspace/capability-roadmap.yml`

Line 3: `purpose: Master reference for all 13 capabilities with governance locations, deliverable standards, and implementation steps`

The file organizes the 13 capabilities into 5 tiers (lines 32–61):

```yaml
tiers:
  - tier: 1  name: Foundational  capabilities: [modeling, architecture]                      # 2
  - tier: 2  name: Build         capabilities: [implementation, infrastructure, configuration] # 3
  - tier: 3  name: Quality       capabilities: [testing, documentation, security]              # 3
  - tier: 4  name: Delivery      capabilities: [automation, deployment]                        # 2
  - tier: 5  name: Operational   capabilities: [observability, operations, continuity]         # 3
```

2 + 3 + 3 + 2 + 3 = **13**, and each of the 13 names has its own fully-specified section further down
in the same file (governance locations, required/optional deliverables, process steps, entry/exit
criteria) — lines 67–688. This is corroborated independently by
`.sbx/workspace/packages/core/go/pkg/sdlc/package.yml:1036`, whose `purpose` field lists the identical
13 names in the identical tier order:

> "One of the 13 capabilities: modeling, architecture, implementation, infrastructure, configuration,
> testing, documentation, security, automation, deployment, observability, operations, continuity"

The full ordered list of 13:

1. modeling
2. architecture
3. implementation
4. infrastructure
5. configuration
6. testing
7. documentation
8. security
9. automation
10. deployment
11. observability
12. operations
13. continuity

### Other candidate locations checked, and why they are NOT the list

- **`docs/plans/2026-01-24-sdlc-factory-vision.md` "Platform Consolidation" table** — this is a
  *different* concept entirely: 12 rows mapping external SaaS tools (Figma, Canva, Xcode IB, VS Code,
  Vercel, Clerk, Supabase, Salesforce, Contentful, Jira, GitHub, Stripe) to planned in-house "Hub"
  replacements. Not a capability taxonomy — a tool-replacement roadmap. The 12-vs-13 near-miss the
  prompt anticipated is real but coincidental; this table is unrelated to the 13-capabilities list.
- **`.sbx/.framework/knowledge/capabilities/`** — contains exactly 3 capability files (`run.yml`,
  `deploy.yml`, `infra.yml`) plus a README, confirming the prior pass's finding. This is a *different,
  much narrower* "knowledge capability" concept (agent-runnable operational capabilities), not the
  13-capability SDLC taxonomy.
- **`docs/user-input/SBX-Technical-Architecture-v0.0.4.md`** — "capability" here (section 6.2 "SBX
  Capabilities", `capability.schema.yml`) refers to yet a *third* sense: composable "knowledge units"
  (tech-stack, patterns, tools) that get attached to an agent/role via a `capabilities:` list (e.g.
  `sbx/capabilities/tech-stack/golang`), used for LLM prompt/context composition — not the 13-item SDLC
  process taxonomy either. No "13" count appears anywhere in this file.
- **`.sbx/workspace/projects/sbx/applications/sbx-web/capability-matrix.yml:6`** — a fourth confirming
  reference: `"# METHOD: 13 capabilities x system-unit levels = feature wishlist."` — this file uses the
  same 13-capability taxonomy as an axis of a feature-planning matrix, reinforcing that
  `capability-roadmap.yml`'s 13 is the one canonical, repo-wide meaning of "13 capabilities" in shredbx.

So: there are (at least) three distinct, unrelated uses of the word "capability" living in shredbx
simultaneously (SDLC-process capability, agent-knowledge capability, agent-runnable operational
capability). Only the SDLC-process one is the "13 capabilities" the user meant, and it is unambiguous
and fully documented at `.sbx/workspace/capability-roadmap.yml`.

---

## Part 3 — the bestays-web Next.js environment

Target (old/superseded, environment-reference only):
`/Users/solo/Projects/workspaces/shredbx/clients/bestie/projects/bestie-workspace/src/apps/bestays-web`

`package.json`: **Next.js `^15.5.7`**, **React `^19.2.1`** / React DOM `^19.2.1`, dev script runs
`next dev --turbopack` (Turbopack dev server, not webpack). **App Router** — there is a top-level `app/`
directory (route groups `(public)`, `(protected)`, plus `api/`, `auth/`, `locations/`, `maintenance/`,
`opengraph-image.tsx`, `robots.ts`, `sitemap.ts`, `globals.css`), no `pages/` directory. UI stack:
**Tailwind CSS v4** (via `@tailwindcss/postcss` in `postcss.config.mjs`, no separate `tailwind.config.js`
— confirms Tailwind v4's CSS-native config), **shadcn/ui** (`components.json` present, `style: "new-york"`,
RSC enabled, `baseColor: "neutral"`, icon library `lucide`) built on **Radix UI** primitives (both the
`radix-ui` umbrella package and individual `@radix-ui/react-*` packages — dialog, alert-dialog, select,
tabs, navigation-menu, scroll-area, label, slot), plus `lucide-react` and `@remixicon/react` for icons,
`next-themes` for light/dark. Forms/validation: `react-hook-form` + `@hookform/resolvers` + `zod`. Other
notable dependencies: `date-fns` + `moment` (both present), `react-big-calendar`, drag-and-drop via both
`@dnd-kit/*` and `@hello-pangea/dnd`, `react-dropzone` + `browser-image-compression` + `heic2any` for
image upload handling, `sonner` for toasts, `vaul` for drawers. Testing via Jest (`jest.config.e2e.js`,
config lives one level up at the monorepo's `jest.config.js`) plus a Python `pytest`/e2e setup
(`requirements-e2e.txt`, `pytest.ini`) alongside Playwright-style `e2e/`. Three **workspace-internal**
packages are pulled in via `workspace:*`: `@reactbook/playground`, `@reactbook/web-services`,
`@reactbook/web-ui` — these are shared UI/services libraries from the bestie-workspace monorepo, not
public npm packages, so a standalone bootstrap cannot `npm install` them; it would need either its own
component library or a deliberate substitute (likely `projects/sbx/packages/core/svelte`-equivalent for
a TS/React core-ui package, if one gets built).

`next.config.ts`: minimal — `transpilePackages` for the three `@reactbook/*` workspace packages,
`turbopack.resolveAlias` mapping `@` → `./src` (note: path alias points at `src/`, but the actual route
tree lives in the top-level `app/`, and `src/` only holds a `components/booking` subfolder — an
apparent structural leftover/inconsistency worth noting, not a convention to copy blindly),
`allowedDevOrigins` for LAN mobile testing, and `images.unoptimized: true` with `remotePatterns` for
Supabase Storage signed URLs and a QR-code service — i.e. this app talks directly to Supabase Storage
for images, unlike bestierealestate's Go-API-mediated `pkg/image` pipeline (consistent with
`GOLANG_API_MIGRATION_REVIEW.md` present in the same directory, suggesting this app was mid-migration
away from direct-Supabase toward the Go API bestierealestate now uses).

**What matching this environment would require**: Next.js 15 + React 19 on the App Router with
Turbopack dev; Tailwind v4 (CSS-based config, no JS config file) + shadcn/ui "new-york" style scaffolded
via `components.json` + Radix UI primitives + lucide icons; react-hook-form + zod for forms; next-themes
for theming; a decision on what replaces the `@reactbook/*` workspace packages (component library +
shared services) since those won't exist outside the old bestie-workspace monorepo; and a call on
whether to standardize on `date-fns` only (drop the redundant `moment`) and pick one drag-and-drop
library rather than carrying both `@dnd-kit/*` and `@hello-pangea/dnd`.

---

## Sources referenced

- `clients/bestie/projects/bestierealestate/{CLAUDE.md, ROADMAP.md, DEPLOY.md, HANDOFF.md}`
- `clients/bestie/projects/bestierealestate/apps/api-chi/` (main.go imports, `internal/*` package docs,
  root-level `smart_collection.go`, `property_similar.go`, `property_related.go`)
- `projects/sbx/packages/core/go/pkg/` (directory listing, cross-checked against BR's actual imports)
- `clients/bestie/projects/bestierealestate/apps/web-svelte/src/routes/` (full route tree)
- `.sbx/workspace/capability-roadmap.yml` (full read)
- `.sbx/workspace/packages/core/go/pkg/sdlc/package.yml:1036` (corroborating line)
- `.sbx/workspace/projects/sbx/applications/sbx-web/capability-matrix.yml:6` (corroborating line)
- `docs/plans/2026-01-24-sdlc-factory-vision.md` (Platform Consolidation table, ruled out)
- `.sbx/.framework/knowledge/capabilities/` (directory listing, ruled out)
- `docs/user-input/SBX-Technical-Architecture-v0.0.4.md` (grepped for "capabilit", ruled out)
- `clients/bestie/projects/bestie-workspace/src/apps/bestays-web/{package.json, next.config.ts,
  components.json, postcss.config.mjs}`
