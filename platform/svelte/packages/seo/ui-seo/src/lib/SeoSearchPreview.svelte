<script lang="ts">
	import type { ResolvedSeo } from './types';

	/**
	 * SeoSearchPreview — a live Google-style search-result (SERP) preview. Fed the SAME
	 * ResolvedSeo that SeoHead renders from, so the preview can't drift from the emitted
	 * <head>. Text is NEVER clipped — the SeoEditor's character counters carry the
	 * length guidance instead (a clipped preview would hide content from the editor).
	 */
	interface Props {
		seo: ResolvedSeo;
	}
	let { seo }: Props = $props();

	// Breadcrumb-style URL (host › segment › segment), Google-style. Falls back to the raw
	// canonical when it isn't an absolute URL (e.g. before the origin is known on SSR).
	const breadcrumb = $derived.by(() => {
		try {
			const url = new URL(seo.canonical);
			const segments = url.pathname.split('/').filter(Boolean);
			return [url.host, ...segments].join(' › ');
		} catch {
			return seo.canonical;
		}
	});
</script>

<div class="serp" aria-label="Search result preview">
	<div class="serp__url">{breadcrumb}</div>
	<div class="serp__title">{seo.documentTitle}</div>
	<div class="serp__desc">{seo.finalDesc}</div>
</div>

<style>
	/* Google-SERP simulation — colors are the platform's own (brand-independent) so the
	   preview reads as a real result regardless of the host site's theme. */
	.serp {
		font-family: arial, sans-serif;
		background: #fff;
		border: 1px solid var(--seo-preview-border, #dfe1e5);
		border-radius: 8px;
		padding: 14px 16px;
		max-width: 600px;
	}
	.serp__url {
		font-size: 12px;
		line-height: 1.3;
		color: #4d5156;
		margin-bottom: 3px;
		word-break: break-word;
	}
	.serp__title {
		font-size: 20px;
		line-height: 1.3;
		color: #1a0dab;
		margin-bottom: 3px;
		word-break: break-word;
	}
	.serp__desc {
		font-size: 14px;
		line-height: 1.58;
		color: #4d5156;
		word-break: break-word;
	}
</style>
