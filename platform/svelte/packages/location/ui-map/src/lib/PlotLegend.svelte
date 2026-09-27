<script lang="ts">
	// PlotLegend — a cadastral survey-instrument panel for a parcel viewer.
	// The hero is a lettered plot diagram (vertices A·B·C…, north-up) that
	// cross-references the A–B / B–C edge list: hovering a row lights the matching
	// edge on the diagram (and, via onhoveredge, on the map). Glass surface, thin
	// blueprint-blue linework, monospace tabular figures. All math comes from
	// parcel.ts (→ @sbx/units); presentation only. Reusable by the canvas (#0303 D7).
	//
	// Styling uses the shared @sbx/ui-map generic token convention
	// (var(--color-*, <fallback>)); --color-plot is the blueprint accent that
	// echoes the map's blue plot. Layout is a flex column capped to the viewport so
	// "Show all" grows the card to full height, then the edge list scrolls.
	import {
		LAND_SIZE_UNITS,
		LAND_SIZE_UNITS_ORDERED,
		type GeoCoordinate,
		type LandSizeUnit
	} from '@sbx/units';
	import { normalizedRingPoints, parcelStats } from './parcel.js';

	let {
		ring,
		unit = $bindable<LandSizeUnit>('sqm'),
		highlightEdge = null,
		onhoveredge,
		startCollapsed = false
	}: {
		ring: GeoCoordinate[];
		unit?: LandSizeUnit;
		highlightEdge?: number | null;
		onhoveredge?: (index: number | null) => void;
		startCollapsed?: boolean;
	} = $props();

	const EDGE_COLLAPSE_THRESHOLD = 8;
	const DIAG = 150; // diagram viewBox units
	const DIAG_PAD = 20; // inset so outward node labels stay in frame

	// Toggle vocabulary = the workspace land-size unit set (same source as the
	// property SizeInput). m² is the human label for sqm; the rest use their
	// display string. No invented "metric/thai/imperial" categories.
	const UNIT_OPTIONS: { id: LandSizeUnit; label: string }[] = LAND_SIZE_UNITS_ORDERED.map((id) => ({
		id,
		label: id === 'sqm' ? 'm²' : LAND_SIZE_UNITS[id].display
	}));

	let edgesExpanded = $state(false);
	// startCollapsed is an initial seed only — the body's open/closed state is
	// user-driven thereafter, so capturing the initial value is intentional.
	// svelte-ignore state_referenced_locally
	let bodyCollapsed = $state(startCollapsed);

	const stats = $derived(parcelStats(ring, unit));
	const hasPlot = $derived(stats.nodeCount >= 3);
	const collapsible = $derived(stats.edges.length > EDGE_COLLAPSE_THRESHOLD);
	const visibleEdges = $derived(
		edgesExpanded || !collapsible ? stats.edges : stats.edges.slice(0, EDGE_COLLAPSE_THRESHOLD)
	);

	// Hero diagram: projected vertices + node labels pushed outward from centroid.
	const diagPts = $derived(normalizedRingPoints(ring, DIAG, DIAG_PAD));
	const diagPoly = $derived(diagPts.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' '));
	const diagNodes = $derived.by(() => {
		const n = diagPts.length;
		if (n === 0) return [];
		let sx = 0;
		let sy = 0;
		for (const p of diagPts) {
			sx += p.x;
			sy += p.y;
		}
		const cx = sx / n;
		const cy = sy / n;
		return diagPts.map((p, i) => {
			const dx = p.x - cx;
			const dy = p.y - cy;
			const len = Math.hypot(dx, dy) || 1;
			return {
				x: p.x,
				y: p.y,
				lx: p.x + (dx / len) * 11,
				ly: p.y + (dy / len) * 11,
				letter: stats.labels[i] ?? ''
			};
		});
	});
	const diagHighlight = $derived.by(() => {
		const n = diagPts.length;
		if (highlightEdge == null || n < 2) return null;
		const a = diagPts[highlightEdge];
		const b = diagPts[(highlightEdge + 1) % n];
		return a && b ? { x1: a.x, y1: a.y, x2: b.x, y2: b.y } : null;
	});

	function hover(index: number | null) {
		onhoveredge?.(index);
	}
</script>

{#if hasPlot}
	<aside class="plot-legend" aria-label="Plot measurements">
		<button
			type="button"
			class="legend__header"
			aria-expanded={!bodyCollapsed}
			onclick={() => (bodyCollapsed = !bodyCollapsed)}
		>
			<span class="legend__glyph" aria-hidden="true"></span>
			<span class="legend__title">Plot</span>
			<span class="legend__count">{stats.nodeCount} nodes</span>
			<svg
				class="legend__chevron"
				class:is-open={!bodyCollapsed}
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
			>
				<polyline points="6 9 12 15 18 9" />
			</svg>
		</button>

		{#if !bodyCollapsed}
			<div class="legend__body">
				<!-- Hero: lettered survey diagram -->
				<div class="legend__diagram">
					<svg
						class="diag"
						viewBox="0 0 {DIAG} {DIAG}"
						role="img"
						aria-label="Plot outline with {stats.nodeCount} vertices"
					>
						<polygon class="diag__poly" points={diagPoly} />
						{#if diagHighlight}
							<line
								class="diag__hl"
								x1={diagHighlight.x1}
								y1={diagHighlight.y1}
								x2={diagHighlight.x2}
								y2={diagHighlight.y2}
							/>
						{/if}
						{#each diagNodes as node, i (i)}
							<circle class="diag__node" cx={node.x} cy={node.y} r="2.4" />
							<text class="diag__label" x={node.lx} y={node.ly}>{node.letter}</text>
						{/each}
					</svg>
					<span class="diag__north" aria-hidden="true">N</span>
				</div>

				<!-- Figures -->
				<div class="legend__figures">
					<div class="fig">
						<span class="fig__label">Area</span>
						<span class="fig__value fig__value--lg">{stats.areaLabel}</span>
					</div>
					<div class="fig fig--row">
						<span class="fig__label">Perimeter</span>
						<span class="fig__value">{stats.perimeterLabel}</span>
					</div>
				</div>

				<!-- Unit switcher -->
				<div class="legend__units" role="group" aria-label="Area unit">
					{#each UNIT_OPTIONS as opt (opt.id)}
						<button
							type="button"
							class="legend__unit"
							class:is-active={unit === opt.id}
							aria-pressed={unit === opt.id}
							onclick={() => (unit = opt.id)}
						>
							{opt.label}
						</button>
					{/each}
				</div>

				<!-- Edge list (flex-grow scroll region) -->
				<ul class="legend__edges">
					{#each visibleEdges as edge, i (edge.name)}
						<li>
							<button
								type="button"
								class="edge"
								class:is-active={highlightEdge === i}
								onpointerenter={() => hover(i)}
								onpointerleave={() => hover(null)}
								onfocus={() => hover(i)}
								onblur={() => hover(null)}
							>
								<span class="edge__name">{edge.name}</span>
								<span class="edge__rule" aria-hidden="true"></span>
								<span class="edge__len">{edge.lengthLabel}</span>
							</button>
						</li>
					{/each}
				</ul>

				{#if collapsible}
					<button
						type="button"
						class="legend__more"
						onclick={() => (edgesExpanded = !edgesExpanded)}
					>
						{edgesExpanded ? 'Show fewer' : `Show all ${stats.edges.length} edges`}
					</button>
				{/if}
			</div>
		{/if}
	</aside>
{/if}

<style>
	.plot-legend {
		display: flex;
		flex-direction: column;
		width: 15.5rem;
		max-width: calc(100vw - 2rem);
		max-height: calc(100dvh - 2.5rem);
		font-family: var(--font-body, 'Inter', system-ui, sans-serif);
		color: var(--color-text, #1c1c1a);
		background: color-mix(in srgb, var(--color-surface, #fff) 76%, transparent);
		-webkit-backdrop-filter: blur(24px) saturate(1.6);
		backdrop-filter: blur(24px) saturate(1.6);
		border: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 50%, transparent);
		border-radius: var(--radius-lg, 0.75rem);
		box-shadow:
			0 18px 44px -14px rgba(15, 23, 30, 0.34),
			inset 0 1px 0 rgba(255, 255, 255, 0.6);
		overflow: hidden;
	}

	/* Header */
	.legend__header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex: none;
		width: 100%;
		padding: 0.6rem 0.8rem;
		background: transparent;
		border: 0;
		border-bottom: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 32%, transparent);
		cursor: pointer;
		text-align: left;
		font: inherit;
		color: inherit;
	}
	.legend__glyph {
		width: 0.5rem;
		height: 0.5rem;
		flex: none;
		transform: rotate(45deg);
		border-radius: 1px;
		background: color-mix(in srgb, var(--color-plot, #2563eb) 16%, transparent);
		border: 1px solid var(--color-plot, #2563eb);
	}
	.legend__title {
		font-size: var(--size-xs, 0.75rem);
		font-weight: 500;
		letter-spacing: 0.18em;
		text-transform: uppercase;
		color: var(--color-muted, #8a8a82);
	}
	.legend__count {
		margin-left: auto;
		font-family: var(--font-mono, ui-monospace, 'SF Mono', Menlo, monospace);
		font-variant-numeric: tabular-nums;
		font-size: 0.7rem;
		letter-spacing: 0.02em;
		color: color-mix(in srgb, var(--color-text-muted, #4b5563) 75%, transparent);
	}
	.legend__chevron {
		width: 13px;
		height: 13px;
		flex: none;
		color: var(--color-muted, #9b9b95);
		transition: transform 180ms ease;
	}
	.legend__chevron.is-open {
		transform: rotate(180deg);
	}

	.legend__body {
		display: flex;
		flex-direction: column;
		min-height: 0;
		flex: 1 1 auto;
		padding: 0.55rem 0.8rem 0.5rem;
	}

	/* Hero diagram */
	.legend__diagram {
		position: relative;
		flex: none;
		padding: 0.3rem 0 0.5rem;
	}
	.diag {
		display: block;
		width: 100%;
		max-width: 9.5rem;
		aspect-ratio: 1 / 1;
		margin: 0 auto;
		overflow: visible;
	}
	.diag__poly {
		fill: color-mix(in srgb, var(--color-plot, #2563eb) 9%, transparent);
		stroke: var(--color-plot, #2563eb);
		stroke-width: 1.5;
		stroke-linejoin: round;
	}
	.diag__hl {
		stroke: var(--color-plot, #2563eb);
		stroke-width: 4;
		stroke-linecap: round;
	}
	.diag__node {
		fill: var(--color-surface, #fff);
		stroke: var(--color-plot, #2563eb);
		stroke-width: 1.3;
	}
	.diag__label {
		font-family: var(--font-mono, ui-monospace, 'SF Mono', Menlo, monospace);
		font-size: 8px;
		font-weight: 600;
		fill: var(--color-text, #1c1c1a);
		text-anchor: middle;
		dominant-baseline: middle;
	}
	.diag__north {
		position: absolute;
		top: 0.3rem;
		right: 0.4rem;
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.6rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		color: var(--color-muted, #b3b3ac);
	}
	.diag__north::before {
		content: '↑';
		display: block;
		text-align: center;
		font-size: 0.7rem;
		line-height: 0.8;
	}

	/* Figures */
	.legend__figures {
		flex: none;
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		padding-top: 0.5rem;
		border-top: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 30%, transparent);
	}
	.fig {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
	}
	.fig--row {
		flex-direction: row;
		align-items: baseline;
		justify-content: space-between;
	}
	.fig__label {
		font-size: 0.65rem;
		font-weight: 500;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--color-muted, #9b9b95);
	}
	.fig__value {
		font-family: var(--font-mono, ui-monospace, 'SF Mono', Menlo, monospace);
		font-variant-numeric: tabular-nums;
		font-weight: 500;
		color: var(--color-text, #1c1c1a);
	}
	.fig__value--lg {
		font-size: 1.0625rem;
		letter-spacing: -0.01em;
		line-height: 1.2;
	}

	/* Unit switcher — one segment per land-size unit (m² · rai · ngan · wah · sqft) */
	.legend__units {
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 1px;
		flex: none;
		margin: 0.65rem 0 0.15rem;
		padding: 2px;
		background: color-mix(in srgb, var(--color-text, #1c1c1a) 4%, transparent);
		border: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 38%, transparent);
		border-radius: var(--radius-md, 0.5rem);
	}
	.legend__unit {
		padding: 0.32rem 0.1rem;
		border: 0;
		border-radius: calc(var(--radius-md, 0.5rem) - 3px);
		background: transparent;
		font: inherit;
		font-size: 0.68rem;
		font-weight: 500;
		text-align: center;
		color: var(--color-text-muted, #6b6b63);
		cursor: pointer;
		transition:
			background 160ms ease,
			color 160ms ease;
	}
	.legend__unit:hover {
		color: var(--color-text, #1c1c1a);
	}
	.legend__unit.is-active {
		background: color-mix(in srgb, var(--color-plot, #2563eb) 14%, transparent);
		color: color-mix(in srgb, var(--color-plot, #2563eb) 92%, #000);
		font-weight: 600;
	}

	/* Edge list — flex-grow scroll region */
	.legend__edges {
		list-style: none;
		margin: 0.35rem 0 0;
		padding: 0;
		min-height: 0;
		flex: 1 1 auto;
		overflow-y: auto;
		overscroll-behavior: contain;
	}
	.edge {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		width: 100%;
		padding: 0.26rem 0.45rem;
		border: 0;
		border-left: 2px solid transparent;
		border-radius: var(--radius-sm, 0.25rem);
		background: transparent;
		font: inherit;
		cursor: default;
		transition: background 130ms ease;
	}
	.edge:hover,
	.edge:focus-visible,
	.edge.is-active {
		background: color-mix(in srgb, var(--color-plot, #2563eb) 8%, transparent);
		border-left-color: var(--color-plot, #2563eb);
		outline: none;
	}
	.edge__name {
		flex: none;
		font-family: var(--font-mono, ui-monospace, 'SF Mono', Menlo, monospace);
		font-size: var(--size-sm, 0.875rem);
		font-weight: 500;
		color: var(--color-text, #1c1c1a);
		letter-spacing: 0.03em;
	}
	.edge__rule {
		flex: 1 1 auto;
		align-self: center;
		height: 0;
		border-bottom: 1px dotted color-mix(in srgb, var(--color-border, #d4d4d8) 65%, transparent);
		transform: translateY(2px);
	}
	.edge__len {
		flex: none;
		font-family: var(--font-mono, ui-monospace, 'SF Mono', Menlo, monospace);
		font-variant-numeric: tabular-nums;
		font-size: var(--size-sm, 0.875rem);
		color: var(--color-text-muted, #6b6b63);
	}

	.legend__more {
		flex: none;
		width: 100%;
		margin-top: 0.4rem;
		padding: 0.42rem;
		border: 0;
		border-top: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 28%, transparent);
		background: transparent;
		font: inherit;
		font-size: 0.72rem;
		font-weight: 600;
		letter-spacing: 0.02em;
		color: var(--color-plot, #2563eb);
		cursor: pointer;
	}
	.legend__more:hover {
		background: color-mix(in srgb, var(--color-plot, #2563eb) 6%, transparent);
	}

	@media (prefers-reduced-motion: reduce) {
		.legend__chevron,
		.legend__unit,
		.edge {
			transition: none;
		}
	}
</style>
