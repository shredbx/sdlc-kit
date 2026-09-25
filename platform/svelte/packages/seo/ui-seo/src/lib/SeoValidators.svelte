<script lang="ts">
	/**
	 * SeoValidators — deep-links to the public SEO / social validation tools, prefilled with
	 * the page's canonical URL. Zero integration/auth: each opens the tool in a new tab. The
	 * tools FETCH the URL server-side, so they only return real results against a publicly
	 * reachable (deployed) URL — on localhost they'll fail (expected in dev). Reusable by any
	 * admin SEO surface (property SEO facet now; content/source editors later).
	 *
	 * Note: X/Twitter retired its public Card Validator (login-walled), so it is intentionally
	 * omitted rather than shipping a dead link.
	 */
	interface Props {
		/** The canonical / public URL to validate (absolute). */
		url: string;
	}
	let { url }: Props = $props();

	interface Validator {
		label: string;
		checks: string;
		href: string;
	}

	const validators = $derived.by((): Validator[] => {
		const u = encodeURIComponent(url);
		return [
			{
				label: 'Google Rich Results',
				checks: 'JSON-LD / structured data',
				href: `https://search.google.com/test/rich-results?url=${u}`
			},
			{
				label: 'Schema Validator',
				checks: 'schema.org validity',
				href: `https://validator.schema.org/#url=${u}`
			},
			{
				label: 'Facebook Debugger',
				checks: 'Open Graph · FB / IG / WhatsApp',
				href: `https://developers.facebook.com/tools/debug/?q=${u}`
			},
			{
				label: 'LinkedIn Inspector',
				checks: 'Open Graph · LinkedIn',
				href: `https://www.linkedin.com/post-inspector/inspect/${u}`
			},
			{
				label: 'PageSpeed Insights',
				checks: 'performance / SEO',
				href: `https://pagespeed.web.dev/analysis?url=${u}`
			}
		];
	});
</script>

<section class="seo-validators" aria-label="External SEO validation">
	<header class="seo-validators__head">
		<h4 class="seo-validators__title">External validation</h4>
		<p class="seo-validators__note">
			Opens each tool in a new tab. The tools fetch the page, so they only return results once
			it's publicly reachable (a deployed URL).
		</p>
	</header>

	<div class="seo-validators__grid">
		{#each validators as v (v.label)}
			<a class="seo-validators__link" href={v.href} target="_blank" rel="noopener noreferrer">
				<span class="seo-validators__link-label">{v.label}</span>
				<span class="seo-validators__link-checks">{v.checks}</span>
			</a>
		{/each}
	</div>

	<p class="seo-validators__url">Testing: <code>{url}</code></p>
</section>

<style>
	.seo-validators {
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
	}
	.seo-validators__head {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.seo-validators__title {
		font-size: 0.72rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--seo-muted, #6b7280);
		margin: 0;
		font-weight: 600;
	}
	.seo-validators__note {
		font-size: 0.8rem;
		color: var(--seo-muted, #6b7280);
		line-height: 1.5;
		margin: 0;
	}
	.seo-validators__grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: 0.6rem;
	}
	.seo-validators__link {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
		padding: 0.6rem 0.85rem;
		border: 1px solid var(--seo-preview-border, #e5e7eb);
		border-radius: 8px;
		text-decoration: none;
		transition:
			border-color 120ms ease,
			background 120ms ease;
	}
	.seo-validators__link:hover {
		border-color: var(--seo-accent, #2563eb);
		background: var(--seo-accent-wash, rgba(37, 99, 235, 0.05));
	}
	.seo-validators__link-label {
		font-size: 0.9rem;
		font-weight: 600;
		color: var(--seo-link, #111827);
	}
	.seo-validators__link-checks {
		font-size: 0.78rem;
		color: var(--seo-muted, #6b7280);
	}
	.seo-validators__url {
		font-size: 0.8rem;
		color: var(--seo-muted, #6b7280);
		margin: 0;
	}
	.seo-validators__url code {
		font-family: ui-monospace, monospace;
		font-size: 0.85em;
		word-break: break-all;
	}
</style>
