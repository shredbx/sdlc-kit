<script lang="ts">
	/**
	 * SectionHeader — Presentational header for page sections.
	 *
	 * Renders title + optional subtitle + optional eyebrow + optional "View all →"
	 * link with count. A pure presentational component — no wrapper <section>,
	 * no children slot for content, no geometry. Lives inside a <Section>.
	 *
	 * Use when a section needs the standard title/subtitle/view-all triad.
	 * Skip entirely for sections whose child carries its own header (LayerMap,
	 * PlatformChips).
	 *
	 * @example
	 * <Section id="databases">
	 *   <SectionHeader
	 *     title="Data & Persistence"
	 *     subtitle="Storage adapters and data transformers"
	 *     viewAllHref="/packages/data-persistence"
	 *     viewAllCount={12}
	 *   />
	 *   <div class="category-grid">…</div>
	 * </Section>
	 */
	import type { Snippet } from 'svelte';

	let {
		title,
		subtitle,
		eyebrow,
		viewAllHref,
		viewAllLabel = 'View All',
		viewAllCount,
		align = 'left',
		extra,
		class: className = '',
	}: {
		title: string;
		subtitle?: string;
		eyebrow?: string;
		viewAllHref?: string;
		viewAllLabel?: string;
		viewAllCount?: number;
		align?: 'left' | 'center';
		extra?: Snippet;
		class?: string;
	} = $props();
</script>

<header class="section-header section-header--{align} {className}">
	<div class="section-header__text">
		{#if eyebrow}<span class="section-header__eyebrow">{eyebrow}</span>{/if}
		<h2 class="section-header__title">{title}</h2>
		{#if subtitle}<p class="section-header__subtitle">{subtitle}</p>{/if}
	</div>

	<div class="section-header__actions">
		{#if extra}{@render extra()}{/if}
		{#if viewAllHref}
			<a href={viewAllHref} class="section-header__view-all">
				{viewAllLabel}
				{#if viewAllCount != null}
					<span class="section-header__count">{viewAllCount}</span>
				{/if}
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
					 stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<line x1="5" y1="12" x2="19" y2="12" />
					<polyline points="12 5 19 12 12 19" />
				</svg>
			</a>
		{/if}
	</div>
</header>

<style>
	.section-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.section-header--center {
		justify-content: center;
		text-align: center;
	}

	.section-header__text {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}

	.section-header__eyebrow {
		font-family: var(--font-mono, monospace);
		font-size: 0.75rem;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--color-text-muted, #6b7280);
		margin-bottom: 0.375rem;
	}

	.section-header__title {
		font-size: 1.25rem;
		font-weight: 700;
		color: var(--color-text, #eeeff1);
		margin: 0;
		line-height: 1.3;
	}

	.section-header__subtitle {
		font-size: 0.875rem;
		color: var(--color-text-muted, #6b7280);
		margin: 0.375rem 0 0;
		line-height: 1.5;
	}

	.section-header__actions {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-shrink: 0;
	}

	.section-header__view-all {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		font-family: var(--font-mono, monospace);
		font-size: 0.75rem;
		font-weight: 500;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--color-text-muted, #6b7280);
		text-decoration: none;
		padding: 0.375rem 0.75rem;
		border: 1px solid var(--color-border, #2a2a2d);
		border-radius: 999px;
		transition: color 0.15s ease, border-color 0.15s ease;
		white-space: nowrap;
	}

	.section-header__view-all:hover {
		color: var(--color-accent, #e94560);
		border-color: var(--color-accent, #e94560);
	}

	.section-header__count {
		font-size: 0.6875rem;
		padding: 0.0625rem 0.375rem;
		border-radius: 0.25rem;
		background: rgba(255, 255, 255, 0.06);
		color: var(--color-text-disabled, #55555a);
	}

	@media (max-width: 768px) {
		.section-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.75rem;
		}
	}
</style>
