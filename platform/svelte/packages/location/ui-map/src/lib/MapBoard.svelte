<script lang="ts">
	// MapBoard (task 2607-133) — the provider-agnostic, read-only map SCREEN for a SET of
	// properties. Sibling of MapStage: same contract, different job. MapStage frames ONE
	// location and its plot; MapBoard plots MANY and reports which one was chosen.
	//
	// Like MapStage it owns NO presentation shell — no fixed positioning, no scrim, no header,
	// no chrome. The HOST decides the frame and renders the preview card / count line / toggles
	// itself, so the same board serves the catalog's split pane and the homepage's band.
	//
	// The board is a flex CHILD (flex: 1 1 auto) — its host must be a flex column with a
	// definite height so the absolutely-positioned map has a box to fill.
	import { untrack } from 'svelte';
	import { createMapBoardAdapter } from './map-board.js';
	import type { LngLat, MapBoardAdapter, MapBoardFrame, MapMarkerPoint, MapViewerDomain } from './types.js';

	let {
		points,
		domain = 'public',
		activeKey = null,
		fitToPoints = true,
		frame = null,
		interactive = true,
		activePanPadding,
		center,
		zoom,
		fitPadding,
		fitSignal = 0,
		emptyMessage = 'No properties to show on the map.',
		onselect,
		onclusterselect,
		ondeselect
	}: {
		/** Everything to plot. Reconciled BY KEY on change, so a filter change mutates the
		 *  surviving markers instead of rebuilding the board. */
		points: readonly MapMarkerPoint[];
		/** Which map provider: 'public' = keyless MapLibre. */
		domain?: MapViewerDomain;
		/** The selected marker, owned by the HOST (the URL is usually the real source). */
		activeKey?: string | null;
		/** Frame the camera to the points on mount. */
		fitToPoints?: boolean;
		/** Fit THIS frame instead of every point (null = all points). The host computes it —
		 *  e.g. the result set's density core with outliers excluded — and swaps it live
		 *  ("Show all" ↔ focused) without remounting. */
		frame?: MapBoardFrame | null;
		/** False = ambient board: no zoom chrome, no camera gestures; marker clicks still work.
		 *  For decorative surfaces (homepage band). */
		interactive?: boolean;
		/** Keeps an activated marker clear of host chrome docked over the map's bottom edge. */
		activePanPadding?: { bottom: number };
		/** Fallback camera used when there are no points. */
		center?: LngLat;
		zoom?: number;
		/** Inset kept between the fitted points and the edges — pass asymmetric padding when
		 *  host chrome (a rail, a floating card) overlaps the map, so no marker hides under it. */
		fitPadding?: { top: number; right: number; bottom: number; left: number };
		/** Re-fit nudge: increment to re-run the current camera fit (frame when set, else all
		 *  points) even though no other input changed — the host's "Fit all results" recovery
		 *  from a visitor's manual pan/zoom. 0 = never fired. */
		fitSignal?: number;
		/** Shown over the basemap when there is nothing to plot; '' suppresses the note (the
		 *  host layers its own zero-state overlay). */
		emptyMessage?: string;
		/** A marker was activated (click / Enter / Space). */
		onselect?: (key: string) => void;
		/** A merged cluster was activated that zooming cannot split (members share a spot) —
		 *  the host reveals the members' listings. Without it the adapter falls back to a
		 *  zoom nudge. */
		onclusterselect?: (clusterKey: string, memberKeys: readonly string[]) => void;
		/** The map background was clicked — the host clears its selection. */
		ondeselect?: () => void;
	} = $props();

	let failed = $state(false);
	let mapContainer = $state<HTMLDivElement | undefined>(undefined);

	let adapter: MapBoardAdapter | null = null;
	let generation = 0;

	// Mount ONCE per container binding. The marker set and the selection are pushed in through
	// the adapter's own methods by the effects below — NOT by remounting — because rebuilding
	// the map on every filter change would refetch tiles and throw away the camera the visitor
	// just set. Geo inputs are read untracked here for that reason.
	$effect(() => {
		const container = mapContainer;
		if (!container) return;
		let cleanup: (() => void) | undefined;
		untrack(() => {
			failed = false;
			const gen = ++generation;
			const a = createMapBoardAdapter(domain);
			adapter = a;
			a.mount({
				container,
				points,
				fitToPoints,
				frame,
				interactive,
				activePanPadding,
				center,
				zoom,
				fitPadding,
				onSelect: (key) => onselect?.(key),
				onClusterSelect: (clusterKey, memberKeys) => onclusterselect?.(clusterKey, memberKeys),
				onDeselect: () => ondeselect?.(),
				onError: () => (failed = true)
			});
			cleanup = () => {
				a.destroy();
				if (gen === generation) adapter = null;
			};
		});
		return () => cleanup?.();
	});

	// Push marker-set changes into the live map.
	$effect(() => {
		const next = points;
		untrack(() => adapter?.setPoints(next));
	});

	// Push selection changes into the live map.
	$effect(() => {
		const key = activeKey;
		untrack(() => adapter?.setActive(key));
	});

	// Push camera-frame changes (focused core ↔ show all) into the live map.
	$effect(() => {
		const f = frame;
		untrack(() => adapter?.setFrame(f));
	});

	// Host-requested re-fit (fitSignal counter): re-runs the CURRENT fit — frame when
	// set, else all points — even though no input changed. The host's recovery from a
	// visitor's manual pan/zoom ("Fit all results").
	$effect(() => {
		const s = fitSignal;
		if (s > 0) untrack(() => adapter?.fit());
	});
</script>

<div class="mb">
	{#if failed}
		<div class="mb__error">Live map unavailable.</div>
	{:else}
		<div class="mb__map" bind:this={mapContainer}></div>
		{#if points.length === 0 && emptyMessage}
			<!-- Empty result: the basemap stays visible and the camera does NOT move. Blanking or
			     re-framing the map at the moment a filter empties would strip the visitor of the
			     one thing still orienting them. An empty-string emptyMessage suppresses this note
			     entirely — for hosts that layer their own richer zero-state overlay instead. -->
			<p class="mb__empty">{emptyMessage}</p>
		{/if}
	{/if}
</div>

<style>
	/* The board is a flex child that fills the remaining space in its host column.
	   isolation: the markers' z-indexes (resting 1 / hover 10 / active 20) must stay INSIDE the
	   board's own stacking context — without it an active pill out-stacks host chrome layered
	   over the map (the rail), and a selected marker floats over the cards it just opened. */
	.mb {
		position: relative;
		isolation: isolate;
		flex: 1 1 auto;
		min-height: 0;
		overflow: hidden;
	}
	.mb__map {
		position: absolute;
		inset: 0;
	}
	.mb__error,
	.mb__empty {
		position: absolute;
		font-family: var(--font-body, 'Inter', system-ui, sans-serif);
		color: var(--color-text-muted, #6b6b63);
	}
	.mb__error {
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.mb__empty {
		left: 50%;
		top: 1rem;
		transform: translateX(-50%);
		z-index: 2;
		margin: 0;
		padding: 0.5rem 0.875rem;
		font-size: 0.8125rem;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.94);
		box-shadow: 0 1px 2px rgba(13, 79, 79, 0.1), 0 2px 6px rgba(13, 79, 79, 0.1);
	}
</style>
