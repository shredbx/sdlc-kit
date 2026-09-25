<script lang="ts">
	/**
	 * DeviceBars — a horizontal split of a labelled dataset (e.g. mobile / tablet
	 * / desktop), each row showing the raw count and its share of the total.
	 *
	 * Token contract (sensible fallbacks):
	 *   --va-bar-color    bar fill              (fallback: --va-accent)
	 *   --va-track-color  bar track             (fallback: rgba(0,0,0,0.06))
	 *   --va-accent       shared accent         (fallback: currentColor)
	 *   --va-label-color  label/percent color   (fallback: #6b7280)
	 *   --va-value-color  count color           (fallback: currentColor)
	 *
	 * @example
	 * <DeviceBars data={[{label:'mobile',value:62},{label:'desktop',value:30}]} />
	 */
	import type { LabeledValue } from './types';

	interface DeviceBarsProps {
		/** Rows to render. Empty → nothing (caller owns the empty state). */
		data: LabeledValue[];
		/** Accessible label for the list. */
		ariaLabel?: string;
	}

	let { data, ariaLabel = 'Device split' }: DeviceBarsProps = $props();

	const total = $derived(data.reduce((sum, d) => sum + d.value, 0));
	function pct(v: number): number {
		return total > 0 ? Math.round((v / total) * 100) : 0;
	}
	// Bar width is share-of-max so the largest device fills the track — easier to
	// compare proportions at a glance than share-of-total on a short list.
	const maxValue = $derived(Math.max(1, ...data.map((d) => d.value)));
</script>

{#if data.length > 0}
	<ul class="va-devices" aria-label={ariaLabel}>
		{#each data as d (d.label)}
			<li class="va-devices__row">
				<span class="va-devices__label">{d.label}</span>
				<span class="va-devices__track">
					<span class="va-devices__fill" style="width: {(d.value / maxValue) * 100}%;"></span>
				</span>
				<span class="va-devices__value">{d.value}</span>
				<span class="va-devices__pct">{pct(d.value)}%</span>
			</li>
		{/each}
	</ul>
{/if}

<style>
	.va-devices {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
	}

	.va-devices__row {
		display: grid;
		grid-template-columns: minmax(5rem, auto) 1fr auto auto;
		align-items: center;
		gap: 0.75rem;
	}

	.va-devices__label {
		font-size: 0.8125rem;
		color: var(--va-value-color, currentColor);
		text-transform: capitalize;
	}

	.va-devices__track {
		position: relative;
		height: 8px;
		border-radius: 9999px;
		background: var(--va-track-color, rgba(0, 0, 0, 0.06));
		overflow: hidden;
	}

	.va-devices__fill {
		position: absolute;
		inset: 0 auto 0 0;
		border-radius: 9999px;
		background: var(--va-bar-color, var(--va-accent, currentColor));
	}

	.va-devices__value {
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--va-value-color, currentColor);
		text-align: right;
		min-width: 2ch;
	}

	.va-devices__pct {
		font-size: 0.75rem;
		color: var(--va-label-color, #6b7280);
		text-align: right;
		min-width: 3ch;
	}
</style>
