<script lang="ts">
	import { page } from '$app/state';
	import { resolveSeo } from './resolve';
	import { serializeJsonLd } from './schema';
	import type { SeoMeta, SeoPageType, JsonLd } from './types';

	/**
	 * SeoHead — emits the per-page <head> SEO/OG/Twitter set (+ optional JSON-LD).
	 * SeoMeta is the shared shape (./types); all override-or-default resolution lives in
	 * resolveSeo (./resolve), shared with the SeoEditor previews so the live editor
	 * preview is byte-identical to what this renders. An absent/null seoMeta degrades to
	 * the derived defaults. Pass structured data via `jsonLd` (built with the ./schema
	 * builders) to additionally emit a <script type="application/ld+json"> for SEO/GEO.
	 */
	interface Props {
		/** Page base title (e.g. the property/article title). seoMeta.meta_title overrides. */
		title: string;
		/** Page base description. seoMeta.meta_description overrides. */
		description: string;
		/** Per-page SEO overrides from the API (or null/absent until the backend lands). */
		seoMeta?: SeoMeta | null;
		/** Brand/site name — appended after the title (`{finalTitle} — {siteName}`). */
		siteName: string;
		/** Absolute fallback OG/twitter image when neither the override nor the page image is set. */
		defaultImage: string;
		/** The page's natural image (e.g. a cover). Used when no seoMeta.og_image override. */
		image?: string;
		/** og:type — website (default) · article · profile. */
		pageType?: SeoPageType;
		/** Origin for absolute canonical/og:url (e.g. https://bestierealestate.com). Defaults to the live origin. */
		canonicalBase?: string;
		/** Optional schema.org structured data (or a list) — build via the ./schema builders.
		 *  Rendered as a <script type="application/ld+json"> for SEO/GEO. Omit → nothing emitted. */
		jsonLd?: JsonLd | JsonLd[];
	}

	let {
		title,
		description,
		seoMeta = null,
		siteName,
		defaultImage,
		image,
		pageType = 'website',
		canonicalBase,
		jsonLd
	}: Props = $props();

	// Single resolution pass — see ./resolve. Request context (origin/pathname) is
	// passed in so the resolver stays pure and unit-testable.
	const r = $derived(
		resolveSeo(
			seoMeta,
			{ title, description, siteName, defaultImage, image, pageType, canonicalBase },
			{ pathname: page.url.pathname, origin: page.url.origin }
		)
	);
</script>

<svelte:head>
	<title>{r.documentTitle}</title>
	<meta name="description" content={r.finalDesc} />
	<link rel="canonical" href={r.canonical} />
	{#if r.noindex}
		<meta name="robots" content="noindex" />
	{/if}

	<meta property="og:type" content={r.pageType} />
	<meta property="og:site_name" content={r.siteName} />
	<meta property="og:title" content={r.ogTitle} />
	<meta property="og:description" content={r.ogDesc} />
	<meta property="og:image" content={r.finalImage} />
	<meta property="og:image:width" content="1200" />
	<meta property="og:image:height" content="630" />
	<meta property="og:url" content={r.canonical} />

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={r.ogTitle} />
	<meta name="twitter:description" content={r.ogDesc} />
	<meta name="twitter:image" content={r.finalImage} />

	<!-- JSON-LD: serializeJsonLd escapes `<` so user-derived text is breakout-safe; the
	     closing tag is concatenated so no literal </script> token appears in source. -->
	{#if jsonLd}
		{@html '<script type="application/ld+json">' + serializeJsonLd(jsonLd) + '</' + 'script>'}
	{/if}
</svelte:head>
