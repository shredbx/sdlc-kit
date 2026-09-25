<script lang="ts">
	// MapControls (#0304) — the map's chrome controls: the base-map switcher (admin Google:
	// roadmap · satellite · terrain · hybrid) + the "Get directions" link. Extracted from the
	// MapViewer header so the SAME markup serves every shell that frames a <MapStage/> — the
	// full-bleed <MapViewer> header AND a consumer's dialog header (e.g. BR's FullscreenCard
	// header-actions slot). It owns no map state: the host binds the stage's `mapTypes` /
	// `activeMapType` in, and `onpick` calls back into the stage's `setMapType`. The switcher
	// hides itself when the provider offers fewer than two base maps (the keyless terrain
	// provider). Close is the SHELL's responsibility, never the controls'.
	let {
		mapTypes,
		activeMapType,
		directionsHref,
		onpick
	}: {
		/** Base-map types the provider offers; < 2 ⇒ the switcher is hidden. */
		mapTypes: readonly string[];
		/** The currently-active base map (drives the pressed state). */
		activeMapType: string;
		/** "Get directions" destination (already resolved by the host). */
		directionsHref: string;
		/** Fired with the chosen base-map type — the host drives the stage's adapter. */
		onpick: (type: string) => void;
	} = $props();
</script>

<div class="mvc">
	{#if mapTypes.length > 1}
		<div class="mvc__types" role="group" aria-label="Base map">
			{#each mapTypes as t (t)}
				<button
					type="button"
					class="mvc__type"
					class:is-active={activeMapType === t}
					aria-pressed={activeMapType === t}
					onclick={() => onpick(t)}
				>
					{t}
				</button>
			{/each}
		</div>
	{/if}
	<!-- Directions only when the host resolved a destination — an approximate AREA frame
	     (private/absent exact pin) passes an empty href, so the link is hidden (a route to a
	     coarse area is meaningless + privacy-leaky). -->
	{#if directionsHref}
		<a class="mvc__dir" href={directionsHref} target="_blank" rel="noopener noreferrer">
			Get directions
			<svg
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
			>
				<path d="M7 17 17 7" /><path d="M7 7h10v10" />
			</svg>
		</a>
	{/if}
</div>

<style>
	.mvc {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
	}
	/* Base-map switcher (admin Google: roadmap·satellite·terrain·hybrid). */
	.mvc__types {
		display: inline-flex;
		gap: 1px;
		padding: 2px;
		background: color-mix(in srgb, var(--color-text, #1c1c1a) 5%, transparent);
		border: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 50%, transparent);
		border-radius: var(--radius-md, 0.5rem);
	}
	.mvc__type {
		padding: 0.3rem 0.55rem;
		border: 0;
		border-radius: calc(var(--radius-md, 0.5rem) - 3px);
		background: transparent;
		font: inherit;
		font-size: var(--size-xs, 0.75rem);
		font-weight: 500;
		text-transform: capitalize;
		color: var(--color-text-muted, #6b6b63);
		cursor: pointer;
		transition:
			background 150ms ease,
			color 150ms ease;
	}
	.mvc__type:hover {
		color: var(--color-text, #1c1c1a);
	}
	.mvc__type.is-active {
		background: color-mix(in srgb, var(--color-plot, #2563eb) 14%, transparent);
		color: color-mix(in srgb, var(--color-plot, #2563eb) 92%, #000);
		font-weight: 600;
	}
	.mvc__dir {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.4rem 0.7rem;
		border: 1px solid color-mix(in srgb, var(--color-border, #d4d4d8) 70%, transparent);
		border-radius: var(--radius-md, 0.5rem);
		font-family: var(--font-body, 'Inter', system-ui, sans-serif);
		font-size: var(--size-sm, 0.875rem);
		font-weight: 500;
		text-decoration: none;
		color: var(--color-text, #1c1c1a);
		transition:
			background 150ms ease,
			border-color 150ms ease;
	}
	.mvc__dir:hover {
		background: color-mix(in srgb, var(--color-plot, #2563eb) 8%, transparent);
		border-color: var(--color-plot, #2563eb);
	}
	.mvc__dir svg {
		width: 14px;
		height: 14px;
	}
	@media (prefers-reduced-motion: reduce) {
		.mvc__type,
		.mvc__dir {
			transition: none;
		}
	}
</style>
