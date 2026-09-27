<script lang="ts">
	/**
	 * BarList — a ranked list of `{label, value}` rows, each with an inline
	 * proportion bar drawn BEHIND the label/value so the row doubles as its own
	 * magnitude indicator. Rows are rendered in the order given (the caller sorts).
	 * An empty `label` renders as the configurable `emptyLabel` (default "Unknown",
	 * the neutral fallback correct for any dimension — country, language, os, …).
	 * Dimensions where empty has a specific meaning (e.g. referer = "Direct") pass
	 * their own `emptyLabel`.
	 *
	 * Token contract (sensible fallbacks):
	 *   --va-barlist-fill   proportion bar tint   (fallback: --va-accent-soft)
	 *   --va-accent-soft    soft accent            (fallback: rgba(0,0,0,0.06))
	 *   --va-label-color    label color            (fallback: currentColor)
	 *   --va-value-color    value color            (fallback: #6b7280)
	 *
	 * @example
	 * <BarList data={byReferer.map(r => ({ label: r.label, value: r.total_hits }))} />
	 */
	import type { LabeledValue } from './types';

	interface BarListProps {
		/** Rows in display order. Empty → nothing (caller owns the empty state). */
		data: LabeledValue[];
		/** Text rendered for a row whose label is the empty string. */
		emptyLabel?: string;
		/** Accessible label for the list. */
		ariaLabel?: string;
	}

	let { data, emptyLabel = 'Unknown', ariaLabel = 'Ranked list' }: BarListProps = $props();

	const maxValue = $derived(Math.max(1, ...data.map((d) => d.value)));
	function display(label: string): string {
		return label === '' ? emptyLabel : label;
	}
</script>

{#if data.length > 0}
	<ul class="va-barlist" aria-label={ariaLabel}>
		{#each data as d (d.label)}
			<li class="va-barlist__row">
				<span class="va-barlist__fill" style="width: {(d.value / maxValue) * 100}%;" aria-hidden="true"></span>
				<span class="va-barlist__label">{display(d.label)}</span>
				<span class="va-barlist__value">{d.value}</span>
			</li>
		{/each}
	</ul>
{/if}

<style>
	.va-barlist {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.va-barlist__row {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 0.5rem 0.625rem;
		border-radius: var(--va-barlist-radius, 0.375rem);
		overflow: hidden;
	}

	.va-barlist__fill {
		position: absolute;
		inset: 0 auto 0 0;
		background: var(--va-barlist-fill, var(--va-accent-soft, rgba(0, 0, 0, 0.06)));
		border-radius: var(--va-barlist-radius, 0.375rem);
		z-index: 0;
	}

	.va-barlist__label {
		position: relative;
		z-index: 1;
		font-size: 0.8125rem;
		color: var(--va-label-color, currentColor);
		/* Never clip: long origins wrap rather than truncate. */
		overflow-wrap: anywhere;
	}

	.va-barlist__value {
		position: relative;
		z-index: 1;
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--va-value-color, #6b7280);
		flex-shrink: 0;
	}
</style>
