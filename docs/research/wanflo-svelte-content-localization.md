# Wanflo Svelte content and localization: implementation review

**Reviewed:** 2026-10-09
**Reference:** [`shredbx/shredbx-workspace`](https://github.com/shredbx/shredbx-workspace/tree/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte)
**Revision:** `a96d9d058fb312eb896e2509710905a4acd6c8bb`
**Purpose:** Record what Wanflo's Svelte app actually does with structured content and localization, what may inform the app framework, and what it does not implement. This is research and suggestions, not approval of a framework API.

## Summary

Wanflo keeps its marketing-site content in YAML, with TypeScript interfaces describing the shapes consumed by Svelte components. A single content module imports, parses, validates, and freezes the YAML during the Vite build. Localized prose is kept in a separate copy deck for each language; a keyset check requires every locale to have exactly the English master keys. The SvelteKit app is prerendered, so this is a build-time content path, not a runtime data service.

This is useful prior art for editable structured content, an explicit content contract, and build-time localization checks. It is not a dynamic section system: page files import section components and choose their order and props directly. It also does not provide a general source registry or a section-scoped view context.

## Source evidence

### Content shape and separation

- [`content/site.yml`](https://github.com/shredbx/shredbx-workspace/blob/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte/content/site.yml) holds site identity, contact details, brand assets, media settings, and curated technology groups. It explicitly keeps prose in the locale copy files.
- [`content/packages.yml`](https://github.com/shredbx/shredbx-workspace/blob/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte/content/packages.yml) contains package structure (`id`, `group`, optional free-text `price`) and its ordering. It does not contain package names, summaries, or feature copy.
- [`src/lib/content/types.ts`](https://github.com/shredbx/shredbx-workspace/blob/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte/src/lib/content/types.ts) defines the content shapes used by the app, including optional-field behavior. It also records deliberate distinctions such as free-text prices versus machine-readable data and localized descriptions versus structural project fields.
- The comments in `packages.yml` and `types.ts` explain a concrete localization defect that was corrected: prose initially left in the structural package file rendered in English on other locales while the copy-deck key check remained green. Moving that prose into the copy decks brought it under the locale-key guarantee.

### Loading and localization

- [`src/lib/content/index.ts`](https://github.com/shredbx/shredbx-workspace/blob/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte/src/lib/content/index.ts) is documented in its source as the only reader of `content/*.yml`. It imports the YAML as raw text, parses and validates it, and freezes the resulting data at build time. Its comments explicitly state that the app has no runtime fetch or server for this content.
- [`src/lib/content/keyset.ts`](https://github.com/shredbx/shredbx-workspace/blob/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte/src/lib/content/keyset.ts) compares each locale's keys against the master in both directions. Missing and extra keys throw a `KeysetError` that identifies the locale file and offending keys.
- [`src/lib/content/keyset.test.ts`](https://github.com/shredbx/shredbx-workspace/blob/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte/src/lib/content/keyset.test.ts) checks matching, missing, extra, and absent-master keysets, including that errors explain where to make the correction.
- Locale routing resolves the locale in the optional `[lang=lang]` route layout and passes it down to pages. The root language is English; Thai and Russian have explicit route prefixes.

### Page and section composition

- The [`/website` page](https://github.com/shredbx/shredbx-workspace/blob/a96d9d058fb312eb896e2509710905a4acd6c8bb/clients/wanflo/projects/wanflo/apps/web/svelte/src/routes/%5B%5Blang%3Dlang%5D%5D/website/%2Bpage.svelte) imports `Hero`, `WorkList`, `PackageCards`, `StepLine`, `FaqList`, `CrossLink`, `ContactBlock`, and `Seo`, then renders them in explicit order.
- The page obtains content through named helpers such as `workIn('web')` and `packagesIn('web')`, and passes the selected data and locale to components as props. It does not read content files from components.
- The shared route layout explicitly renders the site header, child page, footer, and contact dock. Site-wide composition is shared, while each page still controls its own section sequence.
- Reuse is deliberate: structurally parallel service pages use the same section components, with different content keys and filters. This keeps components reusable without making the page sequence data-selected.

## Assessment: what Wanflo demonstrates and does not

| Concern | Observed in Wanflo | Relevance and limit |
|---|---|---|
| Content storage | YAML files split between site configuration, structural collections, and locale copy decks | Useful precedent for editable, small content files with different roles. The exact split should follow Hot Potato's real content, not be imposed wholesale. |
| Content contract | TypeScript interfaces describe the values components consume; optional fields have documented rendering expectations | Shows a code-owned type boundary around author-edited data. The inspected interfaces alone should not be mistaken for a generic runtime source protocol. |
| Localization | English is the master keyset; other locale keys must match in both directions | A concrete build-time safeguard against translation drift. The master language, fallback policy, and strictness remain app/framework decisions. |
| Content validation | The single loader parses and validates YAML before freezing data for consumers | Supports centralized validation rather than component-specific parsing. Its build-time lifecycle is tied to Wanflo's prerendering choice. |
| Data source | Vite-imported local YAML, resolved at build time | Does not demonstrate dependency-injected runtime sources, source registration, or switching between files and a service. |
| Page composition | Pages import and render named Svelte components in code | Reusable sections and concise component composition are present, but section placement/component selection is not dynamic. |
| View context | Locale is passed through route data and component props | Does not establish section-level context or descendants reading section data from a provider. Seaside is the closer precedent for that concern. |
| Decorators and event handlers | No general extension pipeline found in the inspected content/page pattern | Does not establish requirements or patterns for these capabilities. |
| Deploy lifecycle | Static prerendering with no runtime content server is intentional | This should remain a valid framework consumer mode, but must not define the only data-loading lifecycle if Hot Potato needs runtime Node behavior. |

## Suggestions for the Svelte app framework discussion

These are evidence-based recommendations, not approved decisions.

1. **Keep content contracts code-owned and validate author data at the boundary.** Wanflo's components consume typed values rather than parsing YAML. A framework source should similarly return validated data; format-specific parsing belongs in the source implementation, not in section components.
2. **Separate structural values from localized prose where the content calls for it.** Wanflo's package records and localized copy illustrate a useful split, and its documented defect shows why translated strings must not sit outside the locale check. Do not force every field into a flat translation dictionary if typed nested content better expresses the actual page.
3. **Treat locale completeness as a testable contract.** A master-key comparison can catch missing and stale translation keys before shipping. Decide per application whether all locales must be complete or whether deliberate fallbacks are allowed; Wanflo chose strict equality for its own three locales.
4. **Keep page composition and content loading as separate dimensions.** Wanflo demonstrates a simple page that explicitly composes reusable sections while content helpers supply data. Seaside demonstrates section identity/context. A future `DynamicSection`-like facility should combine those concerns only where Hot Potato's actual pages benefit, not assume content records alone can safely choose arbitrary UI.
5. **Support both static and runtime loading without putting storage concerns in components.** Wanflo needs build-time loading and Hot Potato may need runtime Node behavior. The shared contract can define validated content reads while its lifecycle is selected by the app/source integration. Wanflo's no-server design should remain possible.
6. **Use a code-registered component map if data selects section renderers.** Wanflo's explicit imports avoid executable content configuration; Seaside's manual composition likewise does not resolve components from data. If configuration-driven choice is required, limit it to stable identifiers mapped to compiled, typed Svelte components.
7. **Defer decorators and handler registries until a concrete section needs them.** Neither reference demonstrates a general-purpose version. A use case should determine what is wrapped, event execution boundary, ordering, and failure behavior before adding those extension points.

## Questions that remain open

- Which Hot Potato sections genuinely need their component identity and ordering to be data-configurable, versus remaining ordinary explicit Svelte composition?
- What is the minimum shared section contract: placement identity, renderer identifier, validated content, settings, and/or ordered child entries?
- Which values belong to localized copy, and which are locale-independent structure or application configuration?
- Does the first Hot Potato content source need to load at build time, request time, or both? How does the same contract remain usable by Wanflo's prerendered deployment?
- Should section data be passed as props, exposed through render-scoped context for nested fields, or use both patterns for different needs?
- What real Hot Potato behavior would require a decorator or event handler, and what would its execution and error policies be?

## Relationship to existing research

- [`seaside-workspace-dynamic-content.md`](./seaside-workspace-dynamic-content.md) records the Next.js reference's section/content records and section-scoped context. Seaside does not provide Wanflo's strict build-time locale keyset pattern.
- Taken together, the references cover different concerns: Seaside is precedent for contextual section identity and content lookup; Wanflo is precedent for typed, build-time content and localization validation. Neither is a complete dynamic section framework.
- [`reference-repos-index.md`](./reference-repos-index.md) and [`README.md`](./README.md) link this note for future research.

No runtime code or app-framework API was changed. Any Svelte contract or implementation still needs its own design discussion and explicit approval.
