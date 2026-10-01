<script lang="ts">
	/**
	 * TrendChart — hand-rolled inline-SVG trend: vertical bars for views/day with
	 * an overlaid line for daily visits. Zero charting dependency (the dataset is
	 * small — ≤90 day points). A hover tooltip surfaces day · views · visits.
	 *
	 * Token contract (sensible fallbacks):
	 *   --va-bar-color      bar fill                (fallback: rgba(0,0,0,0.18))
	 *   --va-bar-hover      hovered bar fill        (fallback: --va-accent)
	 *   --va-line-color     visits line stroke      (fallback: --va-accent)
	 *   --va-accent         shared accent           (fallback: currentColor)
	 *   --va-axis-color     baseline + tick labels  (fallback: #9ca3af)
	 *   --va-grid-color     gridline                (fallback: rgba(0,0,0,0.06))
	 *
	 * @example
	 * <TrendChart data={byDay.map(d => ({ day: d.day, views: d.total_hits, visits: d.unique_views }))} />
	 */
	import type { TrendPoint } from './types';

	interface TrendChartProps {
		/** The day series. Empty → renders nothing (caller owns the empty state). */
		data: TrendPoint[];
		/** Accessible description of the chart. */
		ariaLabel?: string;
		/** Label for the bar series in the tooltip/legend. */
		viewsLabel?: string;
		/** Label for the line series in the tooltip/legend. */
		visitsLabel?: string;
	}

	let {
		data,
		ariaLabel = 'Views and daily visits over time',
		viewsLabel = 'Views',
		visitsLabel = 'Daily visits'
	}: TrendChartProps = $props();

	// Fixed viewBox — the SVG scales responsively to its container width.
	const W = 720;
	const H = 200;
	const PAD = { top: 12, right: 8, bottom: 24, left: 8 };
	const innerW = W - PAD.left - PAD.right;
	const innerH = H - PAD.top - PAD.bottom;

	const maxValue = $derived(
		Math.max(1, ...data.map((d) => Math.max(d.views, d.visits)))
	);
	const n = $derived(data.length);
	// Slot width per day; bars sit centered with a small gutter.
	const slotW = $derived(n > 0 ? innerW / n : innerW);
	const barW = $derived(Math.max(2, Math.min(slotW * 0.6, 28)));

	function x(i: number): number {
		return PAD.left + i * slotW + slotW / 2;
	}
	function yFor(v: number): number {
		return PAD.top + innerH - (v / maxValue) * innerH;
	}

	// Overlaid visits line as an SVG polyline points string.
	const linePoints = $derived(
		data.map((d, i) => `${x(i)},${yFor(d.visits)}`).join(' ')
	);

	function dayLabel(day: string | Date): string {
		const dt = typeof day === 'string' ? new Date(day) : day;
		if (Number.isNaN(dt.getTime())) return String(day);
		return dt.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}

	// Tick thinning: aim for ~6 visible labels regardless of 7/30/90-day spans so
	// labels never overlap (never clipped — we simply show fewer ticks).
	const tickStep = $derived(Math.max(1, Math.ceil(n / 6)));
	function isTick(i: number): boolean {
		return i % tickStep === 0 || i === n - 1;
	}

	let hover = $state<number | null>(null);
</script>

{#if n > 0}
	<figure class="va-trend" aria-label={ariaLabel}>
		<svg
			class="va-trend__svg"
			viewBox="0 0 {W} {H}"
			preserveAspectRatio="none"
			role="img"
			aria-label={ariaLabel}
		>
			<!-- baseline -->
			<line
				class="va-trend__axis"
				x1={PAD.left}
				y1={PAD.top + innerH}
				x2={W - PAD.right}
				y2={PAD.top + innerH}
			/>
			<!-- bars: views/day -->
			{#each data as d, i (i)}
				<rect
					class="va-trend__bar"
					class:va-trend__bar--hover={hover === i}
					x={x(i) - barW / 2}
					y={yFor(d.views)}
					width={barW}
					height={Math.max(0, PAD.top + innerH - yFor(d.views))}
					rx="1.5"
				/>
				<!-- transparent hover hit-area spanning the full slot height -->
				<rect
					class="va-trend__hit"
					x={x(i) - slotW / 2}
					y={PAD.top}
					width={slotW}
					height={innerH}
					role="presentation"
					onmouseenter={() => (hover = i)}
					onmouseleave={() => (hover = null)}
				/>
			{/each}
			<!-- overlaid line: daily visits -->
			<polyline class="va-trend__line" points={linePoints} fill="none" />
			{#each data as d, i (`pt-${i}`)}
				<circle
					class="va-trend__dot"
					class:va-trend__dot--hover={hover === i}
					cx={x(i)}
					cy={yFor(d.visits)}
					r={hover === i ? 3.5 : 2}
				/>
			{/each}
			<!-- x-axis day labels (thinned) -->
			{#each data as d, i (`lbl-${i}`)}
				{#if isTick(i)}
					<text class="va-trend__tick" x={x(i)} y={H - 6} text-anchor="middle">
						{dayLabel(d.day)}
					</text>
				{/if}
			{/each}
		</svg>

		{#if hover !== null}
			{@const d = data[hover]}
			<div
				class="va-trend__tooltip"
				style="left: {(x(hover) / W) * 100}%;"
				role="status"
			>
				<span class="va-trend__tooltip-day">{dayLabel(d.day)}</span>
				<span class="va-trend__tooltip-row">
					<span class="va-trend__swatch va-trend__swatch--bar"></span>{viewsLabel}: {d.views}
				</span>
				<span class="va-trend__tooltip-row">
					<span class="va-trend__swatch va-trend__swatch--line"></span>{visitsLabel}: {d.visits}
				</span>
			</div>
		{/if}

		<figcaption class="va-trend__legend">
			<span class="va-trend__legend-item">
				<span class="va-trend__swatch va-trend__swatch--bar"></span>{viewsLabel}
			</span>
			<span class="va-trend__legend-item">
				<span class="va-trend__swatch va-trend__swatch--line"></span>{visitsLabel}
			</span>
		</figcaption>
	</figure>
{/if}

<style>
	.va-trend {
		position: relative;
		margin: 0;
		width: 100%;
	}

	.va-trend__svg {
		display: block;
		width: 100%;
		height: auto;
		overflow: visible;
	}

	.va-trend__axis {
		stroke: var(--va-axis-color, #9ca3af);
		stroke-width: 1;
		vector-effect: non-scaling-stroke;
	}

	.va-trend__bar {
		fill: var(--va-bar-color, rgba(0, 0, 0, 0.18));
		transition: fill 120ms ease;
	}

	.va-trend__bar--hover {
		fill: var(--va-bar-hover, var(--va-accent, currentColor));
	}

	.va-trend__hit {
		fill: transparent;
		cursor: pointer;
	}

	.va-trend__line {
		stroke: var(--va-line-color, var(--va-accent, currentColor));
		stroke-width: 2;
		stroke-linejoin: round;
		stroke-linecap: round;
		vector-effect: non-scaling-stroke;
	}

	.va-trend__dot {
		fill: var(--va-line-color, var(--va-accent, currentColor));
		transition: r 120ms ease;
	}

	.va-trend__tick {
		fill: var(--va-axis-color, #9ca3af);
		font-size: 11px;
	}

	.va-trend__tooltip {
		position: absolute;
		top: 0;
		transform: translateX(-50%);
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 0.4rem 0.6rem;
		background: var(--va-tooltip-bg, #111827);
		color: var(--va-tooltip-text, #fff);
		border-radius: 0.375rem;
		font-size: 0.75rem;
		line-height: 1.4;
		white-space: nowrap;
		pointer-events: none;
		z-index: 2;
	}

	.va-trend__tooltip-day {
		font-weight: 600;
	}

	.va-trend__tooltip-row {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.va-trend__legend {
		display: flex;
		gap: 1rem;
		margin-top: 0.5rem;
		font-size: 0.75rem;
		color: var(--va-label-color, #6b7280);
	}

	.va-trend__legend-item {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.va-trend__swatch {
		display: inline-block;
		width: 10px;
		height: 10px;
		border-radius: 2px;
	}

	.va-trend__swatch--bar {
		background: var(--va-bar-color, rgba(0, 0, 0, 0.18));
	}

	.va-trend__swatch--line {
		background: var(--va-line-color, var(--va-accent, currentColor));
		border-radius: 9999px;
	}
</style>
