<script lang="ts">
	/**
	 * BreakdownSection — a labelled section that renders ONE visitor-activity
	 * breakdown dimension as a ranked BarList. Generic and reusable: the consumer
	 * loops its display config and renders one of these per dimension, so adding a
	 * backend dimension needs zero new section markup.
	 *
	 * It reuses the existing BarList primitive for the bars (never reinvents them);
	 * this component owns only the section chrome (title + honest empty state). The
	 * data is the raw `DimensionStat[]` the store emits — the section maps it onto
	 * BarList's {label, value} shape using `total_hits` as the magnitude (matching
	 * the "Views" metric used everywhere else in the analytics surface).
	 *
	 * Empty state is honest ("No data yet") — never a "coming soon" placeholder.
	 * Labels never clip/truncate (BarList wraps long labels with overflow-wrap).
	 *
	 * Token contract is inherited from BarList (the `--va-*` custom properties);
	 * this component adds none of its own and carries no bar CSS.
	 *
	 * @example
	 * <BreakdownSection title="Countries" data={stats.breakdowns.country ?? []} />
	 */
	import BarList from './BarList.svelte';
	import type { DimensionStat } from './types';

	interface BreakdownSectionProps {
		/** Human title for the dimension (from the display config). */
		title: string;
		/** Raw breakdown buckets for this dimension. Empty → the empty state. */
		data: DimensionStat[];
		/** Text rendered for a row whose label is the empty string. */
		emptyLabel?: string;
		/** Copy shown when there is no data for this dimension/period. */
		emptyText?: string;
	}

	let {
		title,
		data,
		emptyLabel = 'Unknown',
		emptyText = 'No data yet'
	}: BreakdownSectionProps = $props();

	// Map the store's DimensionStat onto BarList's {label, value}; total_hits is
	// the magnitude (the "Views" metric), consistent with the rest of the surface.
	const rows = $derived(data.map((d) => ({ label: d.label, value: d.total_hits })));
</script>

<section class="va-breakdown">
	<h3 class="va-breakdown__title">{title}</h3>
	{#if rows.length > 0}
		<BarList data={rows} {emptyLabel} ariaLabel={title} />
	{:else}
		<p class="va-breakdown__empty">{emptyText}</p>
	{/if}
</section>

<style>
	.va-breakdown {
		display: flex;
		flex-direction: column;
		gap: var(--va-breakdown-gap, 0.75rem);
	}

	.va-breakdown__title {
		margin: 0;
		font-family: var(--va-title-font, inherit);
		font-size: var(--va-title-size, 1rem);
		font-weight: 700;
		color: var(--va-title-color, var(--va-value-color, currentColor));
	}

	.va-breakdown__empty {
		margin: 0;
		font-size: 0.8125rem;
		color: var(--va-label-color, #6b7280);
	}
</style>
