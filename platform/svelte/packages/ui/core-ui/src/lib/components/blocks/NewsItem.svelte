<script lang="ts" module>
	/**
	 * NewsFeedItem — the presentation shape for one aggregated news entry.
	 *
	 * This is the wire-projection of `pkg/feed.FeedItem` ENRICHED for display:
	 * the consumer's loader maps the raw API item into this shape (adding the
	 * human-readable `sourceName`, `categoryLabel`, `languageLabel`, and the
	 * relative `timeAgo` string) so the component does ZERO transforms in markup.
	 *
	 * `link` is always the original source article URL (link-out model) — the
	 * "Read more" anchor opens it in a new tab. `imageUrl` may be empty (e.g.
	 * Bangkok Post's bare RSS), in which case a branded placeholder renders.
	 */
	export interface NewsFeedItem {
		/** Stable key — the source GUID or the link. */
		id: string;
		/** Article headline. */
		title: string;
		/** Original source article URL (absolute http/https) — the link-out target. */
		link: string;
		/** Short plain-text excerpt (HTML already stripped server-side). May be empty. */
		excerpt: string;
		/** Hotlinked image URL, or '' when the source provides none. */
		imageUrl: string;
		/** Human-readable source name (e.g. "Bangkok Post — Property"). */
		sourceName: string;
		/** Normalized category display label (e.g. "Property"). */
		categoryLabel: string;
		/** Language display label (e.g. "EN" / "TH"). */
		languageLabel: string;
		/** Relative published time, pre-formatted by the loader (e.g. "3h ago"). */
		timeAgo: string;
		/** ISO-8601 published timestamp for the <time datetime> attribute (may be ''). */
		publishedAt: string;
		/** Raw source <category> values, rendered as tag chips. */
		sourceTags: string[];
		/** Curator-promoted lead article — the consumer renders it as a featured
		 *  lead card (larger media, prominent placement). Optional; defaults false. */
		featured?: boolean;
	}
</script>

<script lang="ts">
	/**
	 * NewsItem — one feed entry, in either a compact LIST row or a richer CARD.
	 *
	 * A single component renders BOTH layouts via the `variant` prop — no
	 * duplication. The "Read more" CTA is always right-aligned and link-outs to
	 * the source in a new tab (target=_blank rel="noopener noreferrer"). Text is
	 * never clipped or truncated (no line-clamp / ellipsis) per workspace UI
	 * rules — the layout accommodates any length.
	 *
	 * Image degrades gracefully: when `item.imageUrl` is empty (Bangkok Post) or
	 * the hotlinked image fails to load, a branded placeholder renders instead —
	 * never a broken-image icon.
	 *
	 * @example
	 * <NewsItem {item} variant="list" />
	 * <NewsItem {item} variant="card" />
	 */
	import Badge from '../primitives/Badge.svelte';
	import Icon from '../primitives/Icon.svelte';

	let {
		item,
		variant = 'list',
		featured = false
	}: {
		item: NewsFeedItem;
		variant?: 'list' | 'card';
		/** Render as the prominent featured lead (larger media + headline). The
		 *  consumer (NewsFeed) sets this for the single curator-promoted item. */
		featured?: boolean;
	} = $props();

	// Image fallback: trust the source URL until it errors at runtime. `failedSrc`
	// records the URL that failed so a recycled DOM node re-attempts a NEW src
	// (the placeholder only shows for the exact URL that errored, or when there
	// is no URL at all). No initial-value capture — derived from current `item`.
	let failedSrc = $state('');
	let showImage = $derived(Boolean(item.imageUrl) && item.imageUrl !== failedSrc);
</script>

<article
	class="news-item news-item--{variant}"
	class:news-item--featured={featured}
	data-news-item
	data-featured={featured ? 'true' : undefined}
>
	{#if featured}
		<span class="news-item__featured-flag">Featured</span>
	{/if}
	<div class="news-item__media">
		{#if showImage}
			<img
				class="news-item__image"
				src={item.imageUrl}
				alt={item.title}
				loading="lazy"
				referrerpolicy="no-referrer"
				onerror={() => (failedSrc = item.imageUrl)}
			/>
		{:else}
			<div class="news-item__placeholder" aria-hidden="true">
				<Icon name="file-text" size="xl" />
			</div>
		{/if}
	</div>

	<div class="news-item__body">
		<div class="news-item__meta">
			<Badge label={item.sourceName} variant="primary" size="sm" />
			{#if item.categoryLabel}
				<Badge label={item.categoryLabel} variant="info" size="sm" />
			{/if}
			{#if item.languageLabel}
				<span class="news-item__lang">{item.languageLabel}</span>
			{/if}
			{#if item.timeAgo}
				<time class="news-item__time" datetime={item.publishedAt}>{item.timeAgo}</time>
			{/if}
		</div>

		<h3 class="news-item__title">{item.title}</h3>

		{#if item.excerpt}
			<p class="news-item__excerpt">{item.excerpt}</p>
		{/if}

		{#if item.sourceTags.length > 0}
			<ul class="news-item__tags">
				{#each item.sourceTags as tag}
					<li class="news-item__tag">{tag}</li>
				{/each}
			</ul>
		{/if}

		<div class="news-item__actions">
			<a
				class="news-item__readmore"
				href={item.link}
				target="_blank"
				rel="noopener noreferrer"
			>
				Read more
				<Icon name="external-link" size="xs" />
			</a>
		</div>
	</div>
</article>

<style>
	/* The component carries its own layout + visual language; consumers theme it
	   through the --news-* token contract (chained over generic --color-* tokens
	   so a brand like BR maps --color-accent -> --br-color-primary and the item
	   adopts the brand automatically). No consumer-side CSS for shared patterns. */
	.news-item {
		position: relative;
		display: flex;
		background: var(--news-item-bg, var(--color-surface, #fff));
		border: 1px solid var(--news-item-border, var(--color-border, #e5e7eb));
		border-radius: var(--news-item-radius, 0.75rem);
		overflow: hidden;
		transition: box-shadow 0.15s ease, border-color 0.15s ease;
	}

	/* ---- FEATURED lead: a prominent hero card. On wide viewports it lays the
	   media beside a larger body; it stacks under 720px. The accent border + the
	   "Featured" flag mark it as the curator-promoted lead. ---- */
	.news-item--featured {
		flex-direction: row;
		align-items: stretch;
		border-color: var(--news-accent, var(--color-accent, #6366f1));
		box-shadow: var(--news-item-shadow-hover, 0 4px 16px rgba(0, 0, 0, 0.08));
	}

	.news-item--featured .news-item__media {
		flex: 0 0 45%;
		min-height: 16rem;
		aspect-ratio: auto;
	}

	.news-item--featured .news-item__body {
		gap: 0.875rem;
		padding: var(--news-item-pad-featured, 1.75rem 2rem);
	}

	.news-item--featured .news-item__title {
		font-size: var(--news-title-size-featured, 1.625rem);
	}

	.news-item__featured-flag {
		position: absolute;
		top: 0.875rem;
		left: 0.875rem;
		z-index: 1;
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--news-accent-contrast, #fff);
		background: var(--news-accent, var(--color-accent, #6366f1));
		border-radius: 999px;
		padding: 0.25rem 0.75rem;
	}

	@media (max-width: 720px) {
		.news-item--featured {
			flex-direction: column;
		}

		.news-item--featured .news-item__media {
			flex-basis: auto;
			width: 100%;
			aspect-ratio: 16 / 9;
			min-height: 0;
		}
	}

	.news-item:hover {
		border-color: var(--news-item-border-hover, var(--color-accent, #6366f1));
		box-shadow: var(--news-item-shadow-hover, 0 4px 16px rgba(0, 0, 0, 0.08));
	}

	/* ---- LIST variant: horizontal row, fixed thumb, dense ---- */
	.news-item--list {
		flex-direction: row;
		align-items: stretch;
	}

	.news-item--list .news-item__media {
		flex: 0 0 var(--news-thumb-list, 12rem);
		min-height: 8rem;
	}

	/* ---- CARD variant: vertical, image on top ---- */
	.news-item--card {
		flex-direction: column;
	}

	.news-item--card .news-item__media {
		width: 100%;
		aspect-ratio: 16 / 9;
	}

	/* ---- Media (image or placeholder) ---- */
	.news-item__media {
		position: relative;
		background: var(--news-placeholder-bg, var(--color-bg-secondary, #f3f4f6));
	}

	.news-item__image {
		display: block;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.news-item__placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 100%;
		color: var(--news-placeholder-fg, var(--color-text-muted, #9ca3af));
		background: linear-gradient(
			135deg,
			var(--news-placeholder-bg, var(--color-bg-secondary, #f3f4f6)),
			var(--news-placeholder-bg-2, var(--color-bg-tertiary, #e5e7eb))
		);
	}

	/* ---- Body ---- */
	.news-item__body {
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
		flex: 1;
		min-width: 0;
		padding: var(--news-item-pad, 1rem 1.25rem);
	}

	.news-item__meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem;
	}

	.news-item__lang {
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--color-text-muted, #6b7280);
		border: 1px solid var(--color-border, #e5e7eb);
		border-radius: 999px;
		padding: 0.0625rem 0.5rem;
	}

	.news-item__time {
		font-size: 0.75rem;
		color: var(--color-text-muted, #9ca3af);
		margin-left: auto;
	}

	.news-item__title {
		font-family: var(--news-title-font, var(--font-heading, inherit));
		font-size: var(--news-title-size, 1.125rem);
		font-weight: 700;
		line-height: 1.35;
		color: var(--color-text, #111827);
		margin: 0;
	}

	.news-item__excerpt {
		font-size: 0.9375rem;
		line-height: 1.55;
		color: var(--color-text-muted, #4b5563);
		margin: 0;
	}

	.news-item__tags {
		display: flex;
		flex-wrap: wrap;
		gap: 0.375rem;
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.news-item__tag {
		font-size: 0.6875rem;
		color: var(--color-text-muted, #6b7280);
		background: var(--color-bg-secondary, #f3f4f6);
		border-radius: 0.375rem;
		padding: 0.125rem 0.5rem;
	}

	/* CTA — always right-aligned (workspace rule, zero exceptions). */
	.news-item__actions {
		display: flex;
		justify-content: flex-end;
		margin-top: auto;
		padding-top: 0.25rem;
	}

	.news-item__readmore {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.8125rem;
		font-weight: 600;
		letter-spacing: 0.02em;
		text-transform: uppercase;
		color: var(--news-cta-color, var(--color-accent, #6366f1));
		text-decoration: none;
		transition: color 0.15s ease;
	}

	.news-item__readmore:hover {
		color: var(--news-cta-color-hover, var(--color-accent-hover, #4f46e5));
		text-decoration: underline;
	}

	.news-item__readmore:focus-visible {
		outline: 2px solid var(--color-accent, #6366f1);
		outline-offset: 3px;
		border-radius: 2px;
	}

	/* On narrow viewports the list row stacks like a card so the thumb never
	   crushes the body (and text is never clipped to fit). */
	@media (max-width: 640px) {
		.news-item--list {
			flex-direction: column;
		}

		.news-item--list .news-item__media {
			flex-basis: auto;
			width: 100%;
			aspect-ratio: 16 / 9;
		}
	}
</style>
