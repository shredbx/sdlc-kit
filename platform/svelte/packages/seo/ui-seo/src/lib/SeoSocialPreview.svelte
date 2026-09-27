<script lang="ts">
	import type { ResolvedSeo } from './types';

	/**
	 * SeoSocialPreview — a live social-share (Open Graph) card preview: the 1200×630 image,
	 * domain, title and description as Facebook / LinkedIn / Slack render them. Fed the same
	 * ResolvedSeo as SeoHead. The image is cover-cropped to the 1.91:1 share ratio (shows
	 * how it WILL be cropped); text is never clipped.
	 */
	interface Props {
		seo: ResolvedSeo;
	}
	let { seo }: Props = $props();

	const domain = $derived.by(() => {
		try {
			return new URL(seo.canonical).host;
		} catch {
			return seo.canonical;
		}
	});
</script>

<div class="og" aria-label="Social share preview">
	{#if seo.finalImage}
		<div class="og__media">
			<img src={seo.finalImage} alt="" />
		</div>
	{/if}
	<div class="og__body">
		<div class="og__domain">{domain}</div>
		<div class="og__title">{seo.ogTitle}</div>
		<div class="og__desc">{seo.ogDesc}</div>
	</div>
</div>

<style>
	/* Open-Graph card simulation — neutral social-card chrome (FB/LinkedIn/Slack look),
	   brand-independent. 1200×630 = 1.91:1 media slot. */
	.og {
		max-width: 100%;
		border: 1px solid var(--seo-preview-border, #dadde1);
		border-radius: 8px;
		overflow: hidden;
		background: #fff;
		font-family: Helvetica, Arial, sans-serif;
	}
	.og__media {
		aspect-ratio: 1.91 / 1;
		background: #e9ebee;
	}
	.og__media img {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.og__body {
		padding: 10px 12px;
		border-top: 1px solid #dadde1;
	}
	.og__domain {
		font-size: 12px;
		text-transform: uppercase;
		color: #606770;
		letter-spacing: 0.02em;
		margin-bottom: 3px;
		word-break: break-word;
	}
	.og__title {
		font-size: 16px;
		font-weight: 700;
		line-height: 1.3;
		color: #1d2129;
		margin-bottom: 4px;
		word-break: break-word;
	}
	.og__desc {
		font-size: 14px;
		line-height: 1.4;
		color: #606770;
		word-break: break-word;
	}
</style>
