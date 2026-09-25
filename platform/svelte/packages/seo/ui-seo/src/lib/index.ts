// @sbx/ui-seo — the reusable SEO surface (the Bindable SEO concern), extracted from
// core-ui so every SEO component lives in one feature package (mirrors @sbx/ui-contact /
// @sbx/ui-calendar). Depends on @sbx/core-ui for form primitives (the SeoEditor, next).
//
// Today: the SeoHead <head> renderer · the resolveSeo SSOT · reusable schema.org
// (JSON-LD) builders. Next F3 steps add the editor + live previews here:
//   SeoEditor.svelte · SeoSearchPreview.svelte · SeoSocialPreview.svelte

export { default as SeoHead } from './SeoHead.svelte';
export { default as JsonLdScript } from './JsonLdScript.svelte';
export { default as SeoEditor } from './SeoEditor.svelte';
export { default as SeoSearchPreview } from './SeoSearchPreview.svelte';
export { default as SeoSocialPreview } from './SeoSocialPreview.svelte';
export { default as SeoValidators } from './SeoValidators.svelte';
export { resolveSeo } from './resolve';
export { toSeoDraft, fromSeoDraft } from './draft';
export {
	serializeJsonLd,
	buildRealEstateListingSchema,
	buildArticleSchema,
	buildItemListSchema,
	buildBreadcrumbListSchema,
	buildFAQPageSchema,
	buildOrganizationSchema
} from './schema';

export type {
	SeoMeta,
	SeoDefaults,
	ResolvedSeo,
	SeoContext,
	SeoPageType,
	JsonLd
} from './types';
export type { SeoDraft } from './draft';
export type {
	RealEstateListingInput,
	ArticleInput,
	PostalAddressInput,
	ItemListInput,
	ItemListEntryInput,
	BreadcrumbCrumbInput,
	FaqPageInput,
	FaqPageEntryInput,
	OrganizationInput
} from './schema';
