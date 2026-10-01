<script lang="ts">
	// MapStage (#0304) — the provider-agnostic, read-only map SCREEN: the live map (via
	// an adapter resolved from `domain`), the red pin + blue parcel, the highlight state,
	// the PlotLegend overlay, and the static-map fallback. It owns NO presentation shell —
	// no fixed positioning, no scrim, no header, no close, no scroll-lock. The HOST decides
	// how the stage is framed:
	//   • full-bleed   → the <MapViewer> wrapper (fixed inset:0 + its own header/close)
	//   • floating card → a consumer's dialog shell (e.g. BR's FullscreenCard) renders
	//                      <MapStage/> in a flush body and the map controls in its header
	// The stage exposes the adapter's base-map state OUTWARD (`mapTypes` / `activeMapType`,
	// both bindable) and a `setMapType` method so whichever shell renders the switcher can
	// drive the map without re-implementing it. The map controls themselves live in the
	// sibling <MapControls/> so the markup exists once across both shells.
	//
	// The stage is a flex CHILD (flex: 1 1 auto) — its host must be a flex column with a
	// definite height so the absolutely-positioned map has a box to fill.
	import { untrack } from 'svelte';
	import type { GeoCoordinate } from '@sbx/units';
	import { sanitizeRing, type LandSizeUnit } from './parcel.js';
	import { createMapViewerAdapter } from './map-viewer.js';
	import type { LngLat, MapViewerAdapter, MapViewerDomain } from './types.js';
	import PlotLegend from './PlotLegend.svelte';

	let {
		ring,
		center,
		domain = 'public',
		approximate = false,
		zoom,
		areaLabel,
		apiKey,
		fallbackSrc,
		title = 'Location & plot',
		mapTypes = $bindable([]),
		activeMapType = $bindable('')
	}: {
		/** Parcel boundary — open or closed ring, [lng,lat]. Empty ⇒ pin-only (no legend). */
		ring: GeoCoordinate[];
		/** Location pin — the exact spot, or the AREA centroid when `approximate` is set. */
		center: LngLat;
		/** Which map provider: 'public' = MapLibre terrain, 'admin' = Google. */
		domain?: MapViewerDomain;
		/** Render the location marker as an APPROXIMATE area halo (not the exact pin) — set when
		 *  the map is framed on an area centroid because the exact location is private/absent. */
		approximate?: boolean;
		/** Initial camera zoom for a pin-only (area) frame; ignored when a plot ring is present. */
		zoom?: number;
		/** Area name shown as the caption on the approximate marker (only when `approximate`). */
		areaLabel?: string;
		/** Provider API key (Google admin domain). */
		apiKey?: string;
		/** Keyless static-map image shown if the live map fails to load. */
		fallbackSrc?: string;
		/** Accessible label for the fallback image / error region. */
		title?: string;
		/** Base-map types the resolved provider offers — published OUT for the shell's
		 *  switcher (empty for the keyless terrain provider → the switcher hides). */
		mapTypes?: readonly string[];
		/** The active base map — published OUT (two-way) so the shell's switcher reflects it. */
		activeMapType?: string;
	} = $props();

	let unit = $state<LandSizeUnit>('sqm');
	let highlightEdge = $state<number | null>(null);
	let failed = $state(false);

	const hasPlot = $derived(sanitizeRing(ring).length >= 3);

	let mapContainer = $state<HTMLDivElement | undefined>(undefined);

	let adapter: MapViewerAdapter | null = null;
	let generation = 0;

	/** Switch the base map — called by the shell's <MapControls/> (the switcher lives in
	 *  the shell, the adapter lives here). Updates the published `activeMapType` so the
	 *  switcher's active state stays in sync. */
	export function setMapType(type: string): void {
		activeMapType = type;
		adapter?.setMapType(type);
	}

	// Mount the adapter ONCE per open — keyed strictly on the map container binding (the
	// only tracked dep). The geo inputs (center/ring/apiKey/domain) are read UNTRACKED: this
	// is a read-only viewer opened fresh each time, so a host that wants to retarget re-keys
	// the {#if} (remount) rather than mutating in place — which keeps a live map from being
	// torn down + re-created (lost camera, tile re-fetch) on an unrelated prop change. The
	// early return until `mapContainer` binds avoids creating a throwaway adapter on the
	// first (container-less) render.
	$effect(() => {
		const container = mapContainer;
		if (!container) return;
		let cleanup: (() => void) | undefined;
		untrack(() => {
			failed = false;
			highlightEdge = null;
			const gen = ++generation;
			const a = createMapViewerAdapter(domain);
			adapter = a;
			mapTypes = a.mapTypes;
			activeMapType = a.defaultMapType;
			a.mount({
				container,
				center,
				ring,
				apiKey,
				approximate,
				zoom,
				areaLabel,
				onNodeHover: (i) => (highlightEdge = i),
				onError: () => (failed = true)
			});
			cleanup = () => {
				a.destroy();
				if (gen === generation) adapter = null;
			};
		});
		return () => cleanup?.();
	});

	// Legend-row hover → highlight that edge on whichever map is mounted.
	$effect(() => {
		highlightEdge;
		adapter?.setHighlightEdge(highlightEdge);
	});
</script>

<div class="ms">
	{#if failed}
		{#if fallbackSrc}
			<img class="ms__fallback" src={fallbackSrc} alt={title} />
		{:else}
			<div class="ms__error">Live map unavailable.</div>
		{/if}
	{:else}
		<div class="ms__map" bind:this={mapContainer}></div>
	{/if}

	{#if hasPlot}
		<div class="ms__legend">
			<PlotLegend {ring} bind:unit {highlightEdge} onhoveredge={(i) => (highlightEdge = i)} />
		</div>
	{/if}
</div>

<style>
	/* The stage is a flex child that fills the remaining space in its host column. */
	.ms {
		position: relative;
		flex: 1 1 auto;
		min-height: 0;
		overflow: hidden;
	}
	.ms__map {
		position: absolute;
		inset: 0;
	}
	.ms__fallback {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: contain;
		background: color-mix(in srgb, var(--color-text, #1c1c1a) 4%, transparent);
	}
	.ms__error {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		font-family: var(--font-body, 'Inter', system-ui, sans-serif);
		color: var(--color-text-muted, #6b6b63);
	}
	.ms__legend {
		position: absolute;
		top: 1rem;
		right: 1rem;
		max-width: calc(100% - 2rem);
		z-index: 2;
	}
</style>
