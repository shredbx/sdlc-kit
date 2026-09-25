// @sbx/ui-seo · types — the SEO value types shared across the SEO surface.
//
// SeoMeta is the per-entity override object; its snake_case keys mirror the backend
// `seo_meta` JSON contract EXACTLY (Go pkg/seo.SeoMeta, the Bindable JSONB column).
// This is the SINGLE TS declaration of that shape — SeoHead, the SeoEditor, and every
// consumer import it from here (previously it was redeclared in SeoHead.svelte and again
// in each app's api.ts; FF3b dedup).

/** og:type discriminator. */
export type SeoPageType = 'website' | 'article' | 'profile';

/**
 * Per-entity SEO + Open Graph override. Every field optional: an unset/empty field
 * falls back to the consumer's derived default (see SeoDefaults) at render time, so a
 * page renders a correct head set even before the backend ships the field. Override
 * wins only when present and non-empty.
 */
export interface SeoMeta {
	meta_title?: string;
	meta_description?: string;
	meta_keywords?: string[];
	og_title?: string;
	og_description?: string;
	og_image?: string;
	canonical_url?: string;
	/** When true, emit `<meta name="robots" content="noindex">` — hides the page from
	 *  search engines. The one genuinely-missing v1 control (SEO/GEO 2025-26 verdict).
	 *  meta_keywords is retained for backend back-compat but no longer rendered/edited
	 *  (dead since 2009, a Bing spam signal). */
	noindex?: boolean;
}

/**
 * Entity-derived fallbacks the consumer computes (property title/cover, article
 * title/hero, …) and hands to resolveSeo. Generic across offering/content/static-page
 * — the consumer owns the derivation, the resolver owns the override-or-default logic.
 */
export interface SeoDefaults {
	/** Base title (e.g. the property/article title). meta_title overrides. */
	title: string;
	/** Base description. meta_description overrides. */
	description: string;
	/** Brand/site name — appended after the title: `{finalTitle} — {siteName}`. */
	siteName: string;
	/** Absolute fallback OG/twitter image when neither override nor page image is set. */
	defaultImage: string;
	/** The page's natural image (e.g. a cover). Used when no og_image override. */
	image?: string;
	/** og:type — website (default) · article · profile. */
	pageType?: SeoPageType;
	/** Origin for absolute canonical/og:url. Defaults to the live origin (see SeoContext). */
	canonicalBase?: string;
}

/** Request-time context the resolver needs to build absolute URLs (kept out of the
 *  pure resolver's globals so it stays testable). */
export interface SeoContext {
	pathname: string;
	origin: string;
}

/** Fully-resolved head values — the single output the renderer AND the live previews
 *  consume, guaranteeing the editor preview is byte-identical to the emitted <head>. */
export interface ResolvedSeo {
	documentTitle: string;
	finalTitle: string;
	finalDesc: string;
	finalImage: string;
	canonical: string;
	ogTitle: string;
	ogDesc: string;
	keywords?: string[];
	noindex: boolean;
	pageType: SeoPageType;
	siteName: string;
}

/** A JSON-LD node (schema.org structured data). The vocabulary is open, so arbitrary
 *  keys carry `unknown` values — type-safe, never `any`. Build these with the schema.ts
 *  helpers and pass to `<SeoHead jsonLd={...}>` to emit a
 *  `<script type="application/ld+json">` (the #1 per-entity GEO lever). */
export interface JsonLd {
	'@context'?: string;
	'@type': string | string[];
	[key: string]: unknown;
}
