// @sbx/ui-seo · resolveSeo: the single override-or-default resolution.
//
// Lifted verbatim from the SeoHead $derived block so there is ONE place that decides
// "override wins, else derived default". SeoHead renders from this; the SeoEditor's
// live SERP/social previews render from this — so the preview can never drift from the
// real <head>. Pure (no $app/state import): request context is passed in via SeoContext.

import type { SeoMeta, SeoDefaults, SeoContext, ResolvedSeo } from './types';

/** Whitespace-collapse for clean meta output (mirrors the original SeoHead behavior). */
function collapse(value: string): string {
	return value.replace(/\s+/g, ' ').trim();
}

/** Absolute-URL helper. An already-absolute path is returned unchanged; otherwise it is
 *  resolved against canonicalBase (brand-canonical, SSR-stable) or the live origin. */
function absolute(pathOrUrl: string, ctx: SeoContext, canonicalBase?: string): string {
	if (/^https?:\/\//.test(pathOrUrl)) return pathOrUrl;
	const base = (canonicalBase ?? ctx.origin).replace(/\/$/, '');
	return `${base}${pathOrUrl}`;
}

/**
 * Resolve the final head values from a (possibly null) override + the entity-derived
 * defaults + request context. Override wins when present and non-empty, else the
 * derived default. Description/OG-description are whitespace-collapsed.
 */
export function resolveSeo(
	meta: SeoMeta | null | undefined,
	defaults: SeoDefaults,
	ctx: SeoContext
): ResolvedSeo {
	const pageType = defaults.pageType ?? 'website';
	const finalTitle = meta?.meta_title || defaults.title;
	const finalDesc = collapse(meta?.meta_description || defaults.description || '');
	const finalImage = meta?.og_image || defaults.image || defaults.defaultImage;
	const canonical = meta?.canonical_url || absolute(ctx.pathname, ctx, defaults.canonicalBase);
	const ogTitle = meta?.og_title || finalTitle;
	const ogDesc = collapse(meta?.og_description || finalDesc || '');

	return {
		documentTitle: `${finalTitle} — ${defaults.siteName}`,
		finalTitle,
		finalDesc,
		finalImage,
		canonical,
		ogTitle,
		ogDesc,
		keywords: meta?.meta_keywords,
		noindex: meta?.noindex ?? false,
		pageType,
		siteName: defaults.siteName
	};
}
