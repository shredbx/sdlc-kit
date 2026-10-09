# Seaside dynamic content: implementation review and Svelte design notes

**Reviewed:** 2026-10-09
**Reference:** [`shredbx/seaside-workspace`](https://github.com/shredbx/seaside-workspace/tree/2477824876ffcd04724b67befcb0b9eee6803a62)
**Revision:** `2477824876ffcd04724b67befcb0b9eee6803a62`
**Purpose:** Record what the Next.js implementation actually does, what is reusable as prior art, and what remains a design question for the Svelte app framework. This is research, not approval of a Svelte API or constructor implementation.

## Summary

Seaside models editable page content as **sections** and **content entries**. A section is identified by `(domain, section)` and owns flexible `info`; content entries use the same identity, have their own flexible `info`, and include a `sorting` value. The page fetches both collections, provides them to descendant component trees through resolvers, and each section layout scopes its children with the current domain and section. Descendant field components/hooks read that context to access the selected section's data.

That is useful precedent for the user's intended composition: establish the identity and data context at a section boundary, make data available to ordinary child components, and keep the page's domain data separate from its visual markup.

It is **not** a dynamic component resolver. The page imports and chooses every section layout in code. The source is Supabase-specific, and the client-side resolvers use module-level Zustand stores. No `DynamicSection`, section-to-component registry, selectable source adapter, decorator pipeline, or general event-handler registry was found in the inspected source. Those are not features to claim as already implemented.

## Source evidence

### Data model

- [`SectionSchema.ts`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b67befcb0b9eee6803a62/src/packages/cms/src/sections/schemas/SectionSchema.ts) defines a section with optional `id`, `domain`, `section`, and `info`. `info` has common localized title/subtitle fields and permits additional fields.
- [`ContentSchema.ts`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b67befcb0b9eee6803a62/src/packages/cms/src/content/schemas/ContentSchema.ts) defines content with optional `id`, `domain`, `section`, flexible `info`, and positive integer `sorting`.
- The model therefore distinguishes section-level metadata from zero or more ordered content records; it does not encode a Svelte/React component identity in those records.

### Loading and providing data

- [`getSections.ts`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b67befcb0b9eee6803a62/src/packages/cms/src/ssr/sections/libs/getSections.ts) queries the Supabase `seaside_sections` table for one or more domains.
- [`getContent.ts`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b67befcb0b9eee6803a62/src/packages/cms/src/ssr/content/libs/getContent.ts) queries `seaside_content`, optionally filtering by section.
- The fetch functions create a Supabase client when one is not supplied. Query errors and exceptions are logged, then represented as empty arrays. This is a source-specific implementation and its success-shaped error fallback should not be copied into a framework source contract without an explicit error policy.
- The [`/3d` route](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b67befcb0b9eee6803a62/src/apps/seaside-web/src/app/%28domains%29/3d/page.tsx) starts section and content reads, wraps its children in `SectionDataResolver` and `ContentDataResolver`, and explicitly renders six named section layouts.
- [`SectionDataResolver`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b67befcb0b9eee6803a62/src/packages/cms/src/sections/components/SectionDataResolver.tsx) and [`ContentDataResolver`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b9eee6803a62/src/packages/cms/src/content/components/ContentDataResolver.tsx) expose fetched records to client components using Zustand stores.

### View context and component composition

- [`Section.tsx`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b9eee6803a62/src/packages/cms/src/sections/components/Section.tsx) receives `domain`, `section`, `className`, and `children`. It renders the HTML section and wraps descendants in `SectionContextProvider`.
- [`SectionContextProvider.tsx`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b9eee6803a62/src/packages/cms/src/sections/components/SectionContextProvider.tsx) places the current identity in React context. `useSectionInfoField` reads that identity and connects it to the section store.
- A concrete [`HeroSectionLayout`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b9eee6803a62/src/apps/seaside-web/src/domains/3d/hero/components/HeroSectionLayout.tsx) uses `<Section domain="seaside3d" section="hero">` and composes explicit hero fields and controls as children.
- The [`HeroTitle`](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b9eee6803a62/src/apps/seaside-web/src/domains/3d/hero/components/HeroTitle.tsx) component gets its title by calling the CMS hook; it does not receive the section record as a direct prop.
- The [`home` route](https://github.com/shredbx/seaside-workspace/blob/2477824876ffcd04724b67befcb0b9eee6803a62/src/apps/seaside-web/src/app/%28domains%29/page.tsx) is also manually composed from `Header`, `HeroSection`, and child components. In both examples, component choice and order are in source code, not selected dynamically from section data.

## Assessment: what the reference proves and does not prove

| Concern | Observed in Seaside | Relevance and limit |
|---|---|---|
| Section identity | `(domain, section)` passed into a section wrapper and made available to descendants | A concrete precedent for contextual section identity and context-aware content fields. |
| Section vs. content | Separate schemas, tables, and resolvers | Useful conceptual separation: singleton section configuration vs. repeatable/ordered content entries. It does not establish that Hot Potato needs editable content or a database. |
| UI composition | Explicit page route imports and renders section-layout components | Shows concise composition around reusable section layouts, but not a data-selected component resolver. |
| Data source | Supabase-specific server fetch functions | Does not demonstrate interchangeable file/API/database adapters. |
| Request isolation | Module-level Zustand stores are created once and populated from resolver effects | Risky as a pattern for concurrent SSR: mutable state is shared at module scope, and mutation occurs in effects. Do not carry this state lifecycle into the Svelte server framework. |
| Missing context | React context begins with an empty identity and the hook falls back to an empty identity when absent | This can hide a missing provider. A new Svelte API should decide explicitly whether context is required and fail clearly if it is absent. |
| Component selection | Chosen by direct imports in route/layout code | No dynamic registry or `<DynamicSection>` mechanism was found. |
| Decorators and handlers | Editing controls and callbacks exist in CMS-specific components | No general-purpose decorator or event-handler extension pipeline was found. These concepts still need to be specified for Svelte. |
| Error behavior | Query failure/exception logs and returns `[]` | Can render as “no content” on a failed read; this conflates missing data and infrastructure failure. The Svelte contract should distinguish them unless the user explicitly chooses otherwise. |

## Suggestions for the Svelte architecture discussion

These are recommendations derived from the inspected code and existing sdlc-kit proposals. They are not approved decisions.

1. **Separate three identities/contracts.** Keep the page/layout's ordered placements, the section record's configuration, and its repeatable content entries distinct. Seaside supports the section/content distinction; `docs/proposals/bos-constructor.md` additionally describes page → layout → slots → blocks, typed renderer/content contracts, and `code declares, data selects`. Reconcile those vocabularies before choosing Svelte types or file names.
2. **Compose with ordinary Svelte components.** Prefer normal Svelte props and slots for explicit composition. Where many nested fields need the current section identity/data, establish a render-scoped context at the `Section` boundary and expose narrow typed accessors to descendants. This takes the useful idea from Seaside without copying its React hooks/stores. Define when context is mandatory and what missing context does.
3. **Inject sources at the server/render boundary.** The route or framework page loader should receive registered source adapters and load the records required for that render. A file-backed adapter can serve the current static content; later adapters can implement the same declared contract. Components should receive serializable, validated data (as props or scoped context), not open a connection or select a storage provider themselves.
4. **Keep visual component selection explicit and safe.** If the user wants data-configured sections, page/layout data should select a stable identifier from a typed, application-registered component map. Only compiled, explicitly registered components are selectable; content files must never name import paths or contain executable component/handler code. Keep a route/layout override possible for sections that need hand-written composition.
5. **Treat decorators and event handlers as separate extension points.** A decorator can be a typed, ordered wrapper/transform around resolved section data or the rendered component; an event handler should be an explicitly registered function associated with a declared event and validated payload. Avoid one generic plugin mechanism for both. Before designing either contract, enumerate a real Hot Potato use case and state whether the handler runs on the server or browser.
6. **Avoid module-global request state.** Make registries immutable after app setup; construct source clients and request-specific view context per render/request. Do not use a mutable module singleton to hold user-, request-, or page-specific records.
7. **Make source errors observable.** Keep not-found, invalid content, and source failure distinguishable. Do not silently turn a failed file/API read into an empty section; the caller should be able to apply a deliberate required/optional section policy.
8. **Use Hot Potato as the first proof, not as a reason to generalize everything.** Map its actual page regions/content into the shared model and use only component primitives/packages that improve this concrete port. Add decorator/handler APIs only after a real section needs them.

## Design questions to resolve before implementation

- Does a page declare an ordered list of named sections, or does a layout supply section placements and let page data override them?
- Is `domain` the content namespace, the feature/module name, or both? How is it distinct from a page route and a section instance id?
- Does a section select its component by an explicit renderer id, or does code bind a component to `(domain, section)` and data only select the placement/settings?
- Which section data is singleton metadata, which data is an ordered content collection, and which Hot Potato values are ordinary app configuration rather than dynamic content?
- Is context a convenience for nested display fields while the section component itself receives explicit data props, or should content components access more of the view context?
- What concrete Hot Potato requirement calls for a decorator or event handler, and what execution boundary, ordering, error handling, and data shape does it need?
- What are the source adapter's operations for this first use case: read one record, read the section list, read ordered content, or another minimal contract?
- Which existing `platform/svelte/packages` components genuinely fit the first page after comparison with Hot Potato's current components?

## Relationship to existing sdlc-kit plans

- [`bos-constructor.md`](../proposals/bos-constructor.md) proposes pages, layouts/slots, blocks, renderers, content types, and source adapters with the rule “code declares, data selects.” It is marked **proposal, not yet approved** and describes a constructor/BOS path, not a settled Svelte app-framework API.
- [`bos-app-roadmap.md`](../proposals/bos-app-roadmap.md) places the renderer registry and layouts in M2, then file/API content sources in M3. It too is a proposal and does not authorize implementing those milestones as part of the current Hot Potato scope.
- [`bos-bootstrap-and-app-framework-architecture.md`](../proposals/bos-bootstrap-and-app-framework-architecture.md) distinguishes the base app framework from optional BOS behavior, but it does not specify a `DynamicSection` API or approve a Svelte-specific context contract.
- [`reference-repos-index.md`](reference-repos-index.md) is updated to point here so this reference is discoverable without rescanning the repo.

No runtime code was changed while preparing this note. This document records evidence and suggestions only; architectural choices and implementation scopes still require user review and approval.
