<script lang="ts" module>
	import type { NewsFeedItem } from './NewsItem.svelte';

	/** A single dropdown option for the filter bar (source/category/language). */
	export interface NewsFilterOption {
		value: string;
		label: string;
	}

	/** Active filter selections (each '' means "All"). Mirrors the URL query. */
	export interface NewsFilterState {
		source: string;
		category: string;
		language: string;
	}

	/** Layout the feed renders items in. Bound to the section's data-view. */
	export type NewsView = 'list' | 'card';
</script>

<script lang="ts">
	/**
	 * NewsFeed — orchestrates the aggregated news section.
	 *
	 * Owns: the filter bar (source / category / language Selects), the list⇄card
	 * VIEW toggle, the item collection, and the loading / empty states. The root
	 * is `section[aria-label="News"]` with a `data-view` attribute reflecting the
	 * current layout — the e2e contract (news.spec.ts) keys off both.
	 *
	 * Framework-agnostic: filter changes are emitted via `onFilter` so the
	 * consumer reflects them to the URL query and triggers an SSR reload (the
	 * feed itself never fetches). The view toggle is purely client-side — it
	 * relayouts the SAME item set with no reload, so the item count is invariant
	 * across the toggle (the toggle assertion depends on this).
	 *
	 * @example
	 * <NewsFeed
	 *   {items}
	 *   {sourceOptions} {categoryOptions} {languageOptions}
	 *   filters={{ source, category, language }}
	 *   onFilter={(next) => goto(buildUrl(next))}
	 * />
	 */
	import { untrack } from 'svelte';
	import NewsItem from './NewsItem.svelte';
	import Select from '../primitives/Select.svelte';
	import Skeleton from '../utilities/Skeleton.svelte';

	let {
		items,
		sourceOptions = [],
		categoryOptions = [],
		languageOptions = [],
		filters,
		initialView = 'list',
		loading = false,
		newerCount = 0,
		showFilters = true,
		onFilter,
		onReveal
	}: {
		/** Enriched, display-ready feed items (already sorted newest-first). */
		items: NewsFeedItem[];
		/** Source dropdown options (excluding the "All" entry — added internally). */
		sourceOptions?: NewsFilterOption[];
		/** Category dropdown options (excluding "All"). */
		categoryOptions?: NewsFilterOption[];
		/** Language dropdown options (excluding "All"). */
		languageOptions?: NewsFilterOption[];
		/** Current filter selections (drive the Select values; '' = All). */
		filters: NewsFilterState;
		/** Initial view layout. The toggle is internal client state thereafter —
		 *  the component is the single source of truth for `view`, so a parent
		 *  never two-way-binds it (that caused a hydration race where the click
		 *  didn't reliably flip data-view). */
		initialView?: NewsView;
		/** Loading state — renders Skeletons instead of items. */
		loading?: boolean;
		/** Count of newer items available since the visitor loaded (incremental
		 *  reveal). >0 renders the reveal control; the consumer polls the count. */
		newerCount?: number;
		/** When false the filter-bar dropdowns are hidden (e.g. when the consumer
		 *  renders its own sidebar filters). The view toggle is always visible.
		 *  Defaults true. */
		showFilters?: boolean;
		/** Emitted when any filter changes — consumer reflects to URL + reloads. */
		onFilter?: (next: NewsFilterState) => void;
		/** Emitted when the reveal control is clicked — the consumer fetches the
		 *  newer items and prepends them (keeping scroll position stable). */
		onReveal?: () => void;
	} = $props();

	// VIEW is owned here as the single source of truth — internal client state,
	// seeded ONCE from `initialView` (untrack — the toggle owns it thereafter, so
	// the binding race is gone). The toggle relayouts the SAME item set with no
	// reload, so the item count is invariant across the toggle.
	let view = $state<NewsView>(untrack(() => initialView));

	// Prepend an "All" option so a bare feed (no filter) is selectable.
	const ALL = '';
	let sourceChoices = $derived([{ value: ALL, label: 'All sources' }, ...sourceOptions]);
	let categoryChoices = $derived([{ value: ALL, label: 'All categories' }, ...categoryOptions]);
	let languageChoices = $derived([{ value: ALL, label: 'All languages' }, ...languageOptions]);

	let isEmpty = $derived(!loading && items.length === 0);

	// Featured-lead split: the (single) curator-promoted item renders as a
	// prominent lead card ahead of the rest, in BOTH view layouts. Falls back to a
	// flat list when nothing is featured.
	let leadItem = $derived(items.find((it) => it.featured) ?? null);
	let restItems = $derived(leadItem ? items.filter((it) => it !== leadItem) : items);

	// Count-aware reveal label — press-room voice, correct singular/plural.
	let revealLabel = $derived(
		newerCount === 1
			? 'Show 1 newer article'
			: `Show ${newerCount} newer articles`
	);

	function update(patch: Partial<NewsFilterState>) {
		onFilter?.({ ...filters, ...patch });
	}
</script>

<section class="news-feed" aria-label="News" data-view={view}>
	<div class="news-feed__controls">
		{#if showFilters}
			<div class="news-feed__filters" role="group" aria-label="Filters">
				<Select
					options={sourceChoices}
					value={filters.source}
					size="sm"
					onchange={(v) => update({ source: v })}
					data-view-id="news-filter-source"
				/>
				<Select
					options={categoryChoices}
					value={filters.category}
					size="sm"
					onchange={(v) => update({ category: v })}
					data-view-id="news-filter-category"
				/>
				<Select
					options={languageChoices}
					value={filters.language}
					size="sm"
					onchange={(v) => update({ language: v })}
					data-view-id="news-filter-language"
				/>
			</div>
		{/if}

		<div class="news-feed__view" role="group" aria-label="View">
			<button
				type="button"
				class="news-feed__view-btn"
				class:news-feed__view-btn--active={view === 'list'}
				aria-pressed={view === 'list'}
				onclick={() => (view = 'list')}
			>
				List
			</button>
			<button
				type="button"
				class="news-feed__view-btn"
				class:news-feed__view-btn--active={view === 'card'}
				aria-pressed={view === 'card'}
				onclick={() => (view = 'card')}
			>
				Card
			</button>
		</div>
	</div>

	{#if newerCount > 0}
		<!-- Incremental reveal — CTA right-aligned (workspace rule). Clicking it
		     prepends the newer items the background job imported since load. -->
		<div class="news-feed__reveal">
			<button type="button" class="news-feed__reveal-btn" onclick={() => onReveal?.()}>
				{revealLabel}
			</button>
		</div>
	{/if}

	{#if loading}
		<div class="news-feed__items" aria-busy="true">
			{#each Array(6) as _, i (i)}
				<div class="news-feed__skeleton">
					<Skeleton variant="card" />
				</div>
			{/each}
		</div>
	{:else if isEmpty}
		<div class="news-feed__empty">
			<p class="news-feed__empty-title">No news right now</p>
			<p class="news-feed__empty-hint">
				No articles match the current filters. Try clearing a filter to see more.
			</p>
		</div>
	{:else}
		{#if leadItem}
			<div class="news-feed__lead">
				<NewsItem item={leadItem} variant="card" featured />
			</div>
		{/if}
		<div class="news-feed__items">
			{#each restItems as item (item.id)}
				<NewsItem {item} variant={view} />
			{/each}
		</div>
	{/if}
</section>

<style>
	.news-feed {
		display: flex;
		flex-direction: column;
		gap: var(--news-feed-gap, 1.5rem);
		width: 100%;
	}

	/* Controls row: filters on the left, view toggle on the right. */
	.news-feed__controls {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.news-feed__filters {
		display: flex;
		flex-wrap: wrap;
		gap: 0.625rem;
		flex: 1;
		min-width: 0;
	}

	.news-feed__filters :global(.select-wrapper) {
		width: auto;
		min-width: 9rem;
	}

	/* Segmented view toggle — two text buttons (List / Card).
	   margin-inline-start:auto pins it to the trailing (right) edge of the
	   controls row — so it's right-aligned whether the filter bar is present
	   (showFilters) or absent (showFilters={false}, BR /news), without relying
	   on the row's space-between having a sibling to push against. */
	.news-feed__view {
		display: inline-flex;
		border: 1px solid var(--color-border, #e5e7eb);
		border-radius: 999px;
		padding: 0.1875rem;
		background: var(--color-bg-secondary, #f3f4f6);
		flex-shrink: 0;
		margin-inline-start: auto;
	}

	.news-feed__view-btn {
		appearance: none;
		border: none;
		background: transparent;
		cursor: pointer;
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-muted, #6b7280);
		padding: 0.375rem 0.875rem;
		border-radius: 999px;
		transition: background 0.15s ease, color 0.15s ease;
	}

	.news-feed__view-btn:hover {
		color: var(--color-text, #111827);
	}

	.news-feed__view-btn--active {
		/* --news-accent lets a consumer brand the active toggle (BR → teal);
		   falls back to the core accent, then a neutral default. */
		background: var(--news-accent, var(--color-accent, #6366f1));
		color: var(--news-accent-contrast, #fff);
	}

	.news-feed__view-btn:focus-visible {
		outline: 2px solid var(--news-accent, var(--color-accent, #6366f1));
		outline-offset: 2px;
	}

	/* Incremental reveal control — full-width row, CTA right-aligned. */
	.news-feed__reveal {
		display: flex;
		justify-content: flex-end;
	}

	.news-feed__reveal-btn {
		appearance: none;
		cursor: pointer;
		font-size: 0.8125rem;
		font-weight: 600;
		letter-spacing: 0.02em;
		color: var(--news-accent-contrast, #fff);
		background: var(--news-accent, var(--color-accent, #6366f1));
		border: 1px solid var(--news-accent, var(--color-accent, #6366f1));
		border-radius: 999px;
		padding: 0.5rem 1.125rem;
		transition: opacity 0.15s ease;
	}

	.news-feed__reveal-btn:hover {
		opacity: 0.9;
	}

	.news-feed__reveal-btn:focus-visible {
		outline: 2px solid var(--news-accent, var(--color-accent, #6366f1));
		outline-offset: 2px;
	}

	/* Featured lead — the curator-promoted item, full-width above the grid. */
	.news-feed__lead {
		width: 100%;
	}

	/* Item collection.
	   - LIST view: single column of horizontal rows.
	   - CARD view: responsive grid of vertical cards. */
	.news-feed[data-view='list'] .news-feed__items {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.news-feed[data-view='card'] .news-feed__items {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(20rem, 1fr));
		gap: 1.25rem;
	}

	.news-feed__skeleton {
		width: 100%;
	}

	.news-feed[data-view='list'] .news-feed__items[aria-busy='true'] {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(20rem, 1fr));
		gap: 1.25rem;
	}

	/* Empty state. */
	.news-feed__empty {
		text-align: center;
		padding: 3rem 1rem;
		color: var(--color-text-muted, #6b7280);
	}

	.news-feed__empty-title {
		font-size: 1.125rem;
		font-weight: 700;
		color: var(--color-text, #111827);
		margin: 0 0 0.375rem;
	}

	.news-feed__empty-hint {
		font-size: 0.9375rem;
		margin: 0;
	}

	@media (max-width: 640px) {
		.news-feed__controls {
			align-items: stretch;
		}

		.news-feed__view {
			align-self: flex-end;
		}
	}
</style>
