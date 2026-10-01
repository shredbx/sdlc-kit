<script lang="ts">
	// MapViewer (#0304) — the FULL-BLEED presentation of a read-only property/contact map.
	// A thin shell (fixed inset:0, its own header + ✕ + Escape + scroll-lock) wrapped around
	// the provider-agnostic <MapStage/> (the actual map) and <MapControls/> (the switcher +
	// directions). This is the immersive, edge-to-edge framing; a consumer that wants the
	// map inside a floating dialog renders <MapStage/> + <MapControls/> in its OWN shell
	// instead (e.g. BR's FullscreenCard). All three share ONE MapStage / MapControls so the
	// map + controls can never drift between framings.
	import { directionsUrl } from './maplibre.js';
	import type { GeoCoordinate } from '@sbx/units';
	import type { LngLat, MapViewerDomain } from './types.js';
	import MapStage from './MapStage.svelte';
	import MapControls from './MapControls.svelte';

	let {
		ring,
		center,
		open = $bindable(false),
		domain = 'public',
		approximate = false,
		zoom,
		areaLabel,
		title = 'Location & plot',
		directionsHref,
		fallbackSrc,
		apiKey
	}: {
		/** Parcel boundary — open or closed ring, [lng,lat]. Empty ⇒ pin-only (no legend). */
		ring: GeoCoordinate[];
		/** Location pin. */
		center: LngLat;
		/** Dialog visibility (two-way). */
		open?: boolean;
		/** Which map provider: 'public' = MapLibre terrain, 'admin' = Google. */
		domain?: MapViewerDomain;
		/** Header label. */
		title?: string;
		/** Override the "Get directions" destination (default: directions to `center`). */
		directionsHref?: string;
		/** Keyless static-map image shown if the live map fails to load. */
		fallbackSrc?: string;
		/** Provider API key (Google admin domain). */
		apiKey?: string;
		/** Render the location marker as an APPROXIMATE area halo (not the exact pin) — set when
		 *  the map frames on an area centroid because the exact location is private/absent. */
		approximate?: boolean;
		/** Initial camera zoom for a pin-only (area) frame; ignored when a plot ring is present. */
		zoom?: number;
		/** Area name shown as the caption on the approximate marker (only when `approximate`). */
		areaLabel?: string;
	} = $props();

	// The stage publishes the adapter's base-map state up here so the header switcher can
	// render + drive it through the stage's `setMapType`.
	let stage = $state<ReturnType<typeof MapStage>>();
	let closeBtn = $state<HTMLButtonElement>();
	let mapTypes = $state<readonly string[]>([]);
	let activeMapType = $state('');

	const dirHref = $derived(directionsHref ?? directionsUrl(center));

	// On open: move focus into the dialog (the ✕, so the keyboard lands inside the viewer,
	// not on the trigger behind it) + lock background scroll; restore the scroll on close.
	$effect(() => {
		if (!open) return;
		closeBtn?.focus();
		const prevOverflow = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = prevOverflow;
		};
	});
</script>

<svelte:window
	onkeydown={(e) => {
		if (open && e.key === 'Escape') open = false;
	}}
/>

{#if open}
	<div class="mv" role="dialog" aria-modal="true" aria-label={title}>
		<header class="mv__bar">
			<h2 class="mv__title">{title}</h2>
			<div class="mv__actions">
				<MapControls
					{mapTypes}
					{activeMapType}
					directionsHref={dirHref}
					onpick={(t) => stage?.setMapType(t)}
				/>
				<button
					bind:this={closeBtn}
					class="mv__close"
					type="button"
					onclick={() => (open = false)}
					aria-label="Close"
				>
					<svg
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<path d="M18 6 6 18" /><path d="m6 6 12 12" />
					</svg>
				</button>
			</div>
		</header>

		<MapStage
			bind:this={stage}
			{ring}
			{center}
			{domain}
			{approximate}
			{zoom}
			{areaLabel}
			{apiKey}
			{fallbackSrc}
			{title}
			bind:mapTypes
			bind:activeMapType
		/>
	</div>
{/if}

<style>
	.mv {
		position: fixed;
		inset: 0;
		z-index: 1000;
		display: flex;
		flex-direction: column;
		background: var(--color-surface, #fff);
	}
	.mv__bar {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex: none;
		padding: 0.7rem 1rem;
		border-bottom: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 60%, transparent);
		background: var(--color-surface, #fff);
	}
	.mv__title {
		margin: 0;
		font-family: var(--font-body, 'Inter', system-ui, sans-serif);
		font-size: var(--size-md, 1rem);
		font-weight: 600;
		color: var(--color-text, #1c1c1a);
	}
	.mv__actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-left: auto;
	}
	.mv__close {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2.1rem;
		height: 2.1rem;
		padding: 0;
		border: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 70%, transparent);
		border-radius: var(--radius-md, 0.5rem);
		background: transparent;
		color: var(--color-text, #1c1c1a);
		cursor: pointer;
		transition: background 150ms ease;
	}
	.mv__close:hover {
		background: color-mix(in srgb, var(--color-text, #1c1c1a) 6%, transparent);
	}
	.mv__close svg {
		width: 18px;
		height: 18px;
	}
	@media (prefers-reduced-motion: reduce) {
		.mv__close {
			transition: none;
		}
	}
</style>
