<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import type {
		GeocodeResult,
		LngLat,
		LocationPickerProps,
		MapAdapter,
		MapHandle,
		MapProvider
	} from './types.js';
	import { MapConfigError } from './types.js';
	import { clampCoords } from './coords.js';
	import { locationPinSvg } from './markers.js';
	import {
		coordsChanged,
		deepestAdminQuery,
		deepestFilledLevel,
		initialPinState,
		targetZoom
	} from './picker-intent.js';
	import { readBrowserEnv, resolveApiKey, resolveProvider } from './env.js';
	import { createGoogleAdapter } from './adapters/google.js';
	import { createMapboxAdapter } from './adapters/mapbox.js';
	import GeocodeSearch from './GeocodeSearch.svelte';

	const DEFAULT_CENTER: LngLat = { lat: 9.7489, lng: 100.031 }; // Koh Phangan
	const DEFAULT_ZOOM = 11;

	// MAP-03/05 — base map types offered by the in-canvas segmented control
	// (Google map-type ids → short labels). Mirrors PolygonPicker's switcher so
	// the two pickers read as one component in different modes.
	type MapType = 'roadmap' | 'satellite' | 'terrain' | 'hybrid';
	const MAP_TYPES: Array<[MapType, string]> = [
		['roadmap', 'Map'],
		['satellite', 'Sat'],
		['terrain', 'Terr'],
		['hybrid', 'Hyb']
	];

	let {
		lat,
		lng,
		provider: providerOverride,
		surface,
		defaultCenter = DEFAULT_CENTER,
		defaultZoom = DEFAULT_ZOOM,
		height = '360px',
		mapView,
		onchange,
		onmapviewchange,
		onaddressresolve,
		country = 'th',
		reverseGeocodeMinZoom = 10,
		reverseGeocodeDebounceMs = 400,
		adminFrame,
		apiKey: apiKeyProp,
		readonly = false
	}: LocationPickerProps = $props();

	// Skip reverse-geocode when the new center is within this distance of
	// the last reverse-geocoded point. ~50m at typical TH latitude — small
	// enough that any meaningful pan triggers a fresh call, large enough
	// to swallow zoom-in nudges and double-click pans.
	const REVERSE_GEOCODE_THRESHOLD_DEG = 0.0005;

	// MAP-01 — how far the LIVE camera must drift from the committed pin before the
	// "Set this location" / "Cancel" affordances re-appear (~22m at TH latitude).
	// Its OWN constant, NOT REVERSE_GEOCODE_THRESHOLD_DEG (that gates reverse-geocode
	// skips — a different tolerance; sharing them would couple two unrelated semantics).
	const PIN_DRIFT_THRESHOLD_DEG = 0.0002;

	// One-time seed of the initial pin from the lat/lng/defaultCenter props —
	// untrack() keeps it a non-reactive snapshot (state_referenced_locally).
	const { center: initial, dirty: initialDirty } = untrack(() =>
		initialPinState(lat, lng, defaultCenter)
	);

	let mapContainer = $state<HTMLDivElement | null>(null);
	// MAP-03 — the canvas element we fullscreen. It wraps BOTH the map div AND
	// the picker overlays (pin, search, Set/Cancel, controls), so they all stay
	// visible in fullscreen — unlike Google's native button, which fullscreens
	// only the bare map div and drops the overlays.
	let canvasEl = $state<HTMLDivElement | null>(null);
	let isFullscreen = $state(false);
	let provider = $state<MapProvider | null>(null);
	let adapter = $state<MapAdapter | null>(null);
	let handle = $state<MapHandle | null>(null);
	// MAP-03 — current base map type, driven by the in-canvas segmented control.
	// Persisted mapView.map_type wins on mount, else 'hybrid' (satellite + labels,
	// the easiest base for visually confirming a property pin).
	let currentMapType = $state<MapType>(untrack(() => (mapView?.map_type as MapType) ?? 'hybrid'));
	let center = $state<LngLat>(initial);
	let dirty = $state(initialDirty);
	// MAP-01 — live camera center, refreshed on every map move (mount, programmatic
	// setCenter, user pan). Distinct from `center` (the COMMITTED pin); comparing the
	// two drives Set/Cancel visibility. Seeded to the initial pin so an already-pinned
	// map shows neither button until the agent actually moves the camera.
	let cameraCenter = $state<LngLat>(initial);
	// MAP-01 a11y — visually-hidden aria-live text for the Set/Cancel toggle (the
	// clicked button unmounts, so we move focus to the map AND announce here).
	let statusMessage = $state('');
	let errorMessage = $state<string | null>(null);
	let loading = $state(true);
	// Display text pushed into the GeocodeSearch input from outside —
	// updates after every successful reverse-geocode. Empty string clears
	// the input when reverse-geocode returns ZERO_RESULTS or zoom is too
	// low. `undefined` leaves the input untouched.
	let searchDisplay = $state<string | undefined>(undefined);
	// Stale-response guard for in-flight reverse-geocode calls.
	let reverseSeq = 0;
	// Debounce timer + last-reverse-geocoded coords (for the movement
	// threshold check).
	let reverseTimer: ReturnType<typeof setTimeout> | null = null;
	let lastReverseAt: LngLat | null = null;
	// Background indicator: true while a reverse-geocode is in flight, so
	// the UI can show a subtle "Locating…" state under the readout.
	let reverseLoading = $state(false);

	// ── Zoom-ladder framing ──────────────────────────────────────────────
	// Camera priority: pin > admin-frame > persisted. The zoom is a FIXED ladder
	// (Google zoom integers) keyed to the deepest filled level — NOT fitBounds —
	// so the camera lands at a defined, predictable scale (2606-053):
	//   province=9 · district=11 · subDistrict=13 · location/pin=16 (max).
	// All ladder logic is delegated to picker-intent.ts (the contract):
	// `targetZoom` / `deepestFilledLevel`.
	//
	// • NOT pinned → geocode the deepest filled admin level and setCenter() to
	//   its center, then setZoom(ladder[level]).
	// • Pinned → center on the pin coords at the location zoom (16); no geocode
	//   needed (we already have the coords). Runs on commit AND on mount when a
	//   pin pre-exists, so the map always shows a defined area + zoom even when
	//   the location is already set.
	let adminFrameTimer: ReturnType<typeof setTimeout> | null = null;
	let adminFrameSeq = 0;
	// Last admin query we framed to — skips redundant geocodes when the effect
	// re-runs without the assembled string actually changing.
	let lastAdminFramed: string | null = null;

	// Frame directly on a placed pin: center on its coords at the location ladder
	// zoom (16, max). No geocode — the coords are already confirmed.
	function framePin(loc: LngLat) {
		if (!handle) return;
		const zoom = targetZoom(undefined, true);
		handle.setCenter(clampCoords(loc));
		if (zoom != null) handle.setZoom(zoom);
	}

	function scheduleAdminFrame(query: string) {
		if (adminFrameTimer) {
			clearTimeout(adminFrameTimer);
			adminFrameTimer = null;
		}
		adminFrameTimer = setTimeout(() => {
			void runAdminFrame(query);
		}, reverseGeocodeDebounceMs);
	}

	async function runAdminFrame(query: string) {
		if (!adapter || !handle) return;
		// A pin was confirmed between schedule and fire — abandon framing.
		if (dirty) return;
		const level = deepestFilledLevel(adminFrame);
		if (!level) return;
		const zoom = targetZoom(adminFrame, false);
		if (zoom == null) return;
		const seq = ++adminFrameSeq;
		try {
			const results = await adapter.search({ q: query, country, limit: 1 });
			if (seq !== adminFrameSeq) return; // a newer cascade change superseded this
			if (dirty) return; // pin landed while geocoding — pin wins
			const top = results[0];
			if (top) {
				// Center on the geocoded admin region + the FIXED ladder zoom for
				// the deepest filled level (replaces fitBounds — defined scale, not
				// a bounds-fit guess).
				handle.setCenter(clampCoords({ lat: top.lat, lng: top.lng }));
				handle.setZoom(zoom);
				lastAdminFramed = query;
			}
		} catch {
			// Geocode transport error — leave the camera where it is. Framing is
			// a best-effort convenience, never a hard failure.
		}
	}

	function selectAdapter(p: MapProvider, key: string): MapAdapter {
		if (p === 'google') return createGoogleAdapter(key);
		return createMapboxAdapter(key);
	}

	function emitChange(loc: LngLat) {
		const clamped = clampCoords(loc);
		if (!coordsChanged(center, clamped)) return;
		center = clamped;
		dirty = true;
		onchange?.(clamped);
	}

	// MAP-01 — Set/Cancel smart visibility. Compare the live camera to the committed pin:
	//  • no pin yet (!dirty) → show "Set this location" (place the first pin); no Cancel.
	//  • pin set + camera drifted off it → show "Set" (re-confirm here) + "Cancel" (snap back).
	//  • pin set + camera still resting on it → hide both (nothing to commit or revert).
	const cameraDrifted = $derived(
		Math.abs(cameraCenter.lat - center.lat) > PIN_DRIFT_THRESHOLD_DEG ||
			Math.abs(cameraCenter.lng - center.lng) > PIN_DRIFT_THRESHOLD_DEG
	);
	const showSet = $derived(!dirty || cameraDrifted);
	const showCancel = $derived(dirty && cameraDrifted);

	function farEnoughForReverseGeocode(next: LngLat): boolean {
		if (!lastReverseAt) return true;
		return (
			Math.abs(next.lat - lastReverseAt.lat) > REVERSE_GEOCODE_THRESHOLD_DEG ||
			Math.abs(next.lng - lastReverseAt.lng) > REVERSE_GEOCODE_THRESHOLD_DEG
		);
	}

	// `emitOnResolve` is true ONLY for the explicit "Set this location" commit
	// path — when the reverse-geocode resolves, the picker fires
	// `onaddressresolve(result)` so the consumer can auto-fill its empty address
	// fields. Drag/search PREVIEW calls pass it false: they update the search
	// readout (`searchDisplay`) but never push into the form.
	function scheduleReverseGeocode(
		viewport: { lat: number; lng: number; zoom: number },
		emitOnResolve = false
	) {
		if (reverseTimer) {
			clearTimeout(reverseTimer);
			reverseTimer = null;
		}
		// Below the threshold zoom Google returns continent / country labels —
		// useless for a PASSIVE preview, so we clear the search input and bail.
		// A COMMIT (emitOnResolve) bypasses this: the agent explicitly placed
		// the point, so we always resolve + auto-fill regardless of how far the
		// camera is zoomed out (the pin is then framed to ladder zoom 16 anyway).
		if (!emitOnResolve && viewport.zoom < reverseGeocodeMinZoom) {
			searchDisplay = '';
			lastReverseAt = null;
			return;
		}
		const next: LngLat = { lat: viewport.lat, lng: viewport.lng };
		// A commit always reverse-geocodes (the agent's explicit action), even
		// when the point hasn't moved far enough for a passive preview refresh —
		// otherwise pressing Set on an already-previewed spot wouldn't auto-fill.
		if (!emitOnResolve && !farEnoughForReverseGeocode(next)) return;
		reverseTimer = setTimeout(() => {
			void runReverseGeocode(next, emitOnResolve);
		}, reverseGeocodeDebounceMs);
	}

	async function runReverseGeocode(loc: LngLat, emitOnResolve = false) {
		if (!adapter) return;
		const seq = ++reverseSeq;
		reverseLoading = true;
		try {
			const result = await adapter.reverseGeocode({ lat: loc.lat, lng: loc.lng, country });
			if (seq !== reverseSeq) return; // newer drag superseded this call
			lastReverseAt = loc;
			if (result) {
				searchDisplay = result.display_name;
				// Commit path only: hand the resolved structured address to the
				// consumer for empty-only field fill. The pin (onchange) already
				// fired; this is the address layer settling behind it.
				if (emitOnResolve) onaddressresolve?.(result);
			} else {
				// ZERO_RESULTS — keep the pin but show nothing in the search.
				searchDisplay = '';
			}
		} catch {
			if (seq !== reverseSeq) return;
			// Transport errors leave the prior readout in place; just stop the
			// spinner. A failed commit reverse-geocode silently fills nothing —
			// the pin still committed via onchange.
		} finally {
			if (seq === reverseSeq) reverseLoading = false;
		}
	}

	onMount(() => {
		let cancelled = false;
		let resolvedHandle: MapHandle | null = null;
		let resolvedAdapter: MapAdapter | null = null;

		// MAP-03 — track fullscreen enter/exit (Esc also exits, firing this).
		// Sync the flag (drives the toggle icon/label) and resize the map, or
		// Google keeps painting at the pre-fullscreen container size.
		const onFullscreenChange = () => {
			isFullscreen = document.fullscreenElement === canvasEl;
			handle?.resize();
		};
		document.addEventListener('fullscreenchange', onFullscreenChange);

		(async () => {
			try {
				const env = readBrowserEnv();
				const p = resolveProvider({
					provider: providerOverride,
					surface,
					envProvider: env.provider
				});
				// Prop wins — consumer's $env/dynamic/public read flows through
				// here at runtime, which is the only path that works in
				// production Node builds. Vite dev mode also populates env.*,
				// kept as a fallback so the picker doesn't need a prop wiring
				// to be smoke-testable in isolation.
				const key =
					apiKeyProp ||
					resolveApiKey({
						provider: p,
						googleKey: env.googleKey,
						mapboxToken: env.mapboxToken
					});
				if (!key) {
					throw new MapConfigError(
						p === 'google'
							? 'PUBLIC_GOOGLE_MAPS_API_KEY is empty'
							: 'PUBLIC_MAPBOX_TOKEN is empty'
					);
				}
				if (!mapContainer) {
					throw new MapConfigError('Map container element not ready');
				}

				// Initial camera prefers persisted mapView center/zoom when no
				// lat/lng is pinned yet. Once a pin exists, lat/lng wins for the
				// camera center but mapView.zoom still applies — agent's saved
				// zoom is independent of where the pin sits.
				const initialCameraCenter =
					!dirty && mapView?.center
						? { lat: mapView.center[1], lng: mapView.center[0] }
						: center;
				const initialCameraZoom =
					typeof mapView?.zoom === 'number' ? mapView.zoom : defaultZoom;

				const a = selectAdapter(p, key);
				const h = await a.mount({
					container: mapContainer,
					center: initialCameraCenter,
					zoom: initialCameraZoom,
					apiKey: key,
					// Default to hybrid (satellite + labels) — easiest base for
					// visually confirming a property pin. Persisted map_type wins;
					// the in-canvas map-type control drives `currentMapType` after.
					mapType: currentMapType,
					// Read-only → non-interactive static pin preview (no gestures /
					// no Google control chrome). The picker's own overlays are
					// suppressed in the markup below.
					readonly,
					// Point picker → zoom about the centered pin, not the cursor, so
					// zooming in/out keeps the pin over the same spot (2606-066).
					lockCenterOnZoom: true
				});

				if (cancelled) {
					a.destroy(h);
					return;
				}

				h.onmove = (viewport) => {
					// MAP-01 — keep the live camera center in sync so Set/Cancel can compare it
					// against the committed pin. This is the CAMERA, not the pin (the pin only
					// moves on an explicit Set — 2606-053).
					cameraCenter = { lat: viewport.lat, lng: viewport.lng };
					// Viewport tracking only — fires on initial mount, programmatic setCenter,
					// and user pan. Camera state may be persisted by the consumer; the pin is
					// NEVER updated here. The fix for Task #6 (2605-144) lives in this split:
					// previously this line emitted onchange on the first idle event of every
					// mount, baking the map's default-center coords into the consumer's state.
					onmapviewchange?.({
						zoom: viewport.zoom,
						center: [viewport.lng, viewport.lat]
					});
				};
				h.ondragend = (viewport) => {
					// User-initiated pan stop. Drag NO LONGER commits the pin
					// (2606-053): the "Set this location" button is the single commit
					// affordance. Dragging only refreshes the live PREVIEW readout
					// (searchDisplay) via a reverse-geocode — no emitChange, no fill.
					// setCenter() does NOT trigger ondragend per Google Maps Events docs:
					// https://developers.google.com/maps/documentation/javascript/events
					// Programmatic moves (search-pick → setCenter) flow through onmove only.
					scheduleReverseGeocode(viewport, false);
				};

				provider = p;
				adapter = a;
				handle = h;
				resolvedHandle = h;
				resolvedAdapter = a;
				loading = false;
			} catch (e) {
				if (cancelled) return;
				errorMessage =
					e instanceof MapConfigError
						? e.message
						: e instanceof Error
							? e.message
							: 'Map could not be loaded';
				loading = false;
			}
		})();

		return () => {
			cancelled = true;
			document.removeEventListener('fullscreenchange', onFullscreenChange);
			if (reverseTimer) {
				clearTimeout(reverseTimer);
				reverseTimer = null;
			}
			if (adminFrameTimer) {
				clearTimeout(adminFrameTimer);
				adminFrameTimer = null;
			}
			if (resolvedHandle && resolvedAdapter) {
				resolvedAdapter.destroy(resolvedHandle);
			}
		};
	});

	// Zoom-ladder framing effect. Two modes, both delegating the zoom to
	// `targetZoom` (the contract):
	//  • NOT pinned → re-frame the empty-state camera whenever the assembled
	//    admin query changes, tightening as the agent fills deeper cascade
	//    levels (province → district → subDistrict ladder zooms).
	//  • Pinned → frame the pin ONCE (center + location zoom 16) when the map
	//    first mounts with a pre-existing pin. The one-shot `pinFramed` guard
	//    keeps the effect from yanking the camera back to 16 every time the
	//    agent zooms out to inspect, while still guaranteeing "the map shows a
	//    defined area + zoom even when the location is already set" on load.
	//    On a fresh commit the pin is framed directly in setHere(), not here.
	// Reactive on `adminFrame` + `handle` + `dirty`.
	let pinFramed = false;
	$effect(() => {
		const query = deepestAdminQuery(adminFrame);
		// Read these so the effect re-runs when the map mounts or a pin lands.
		const ready = !!handle && !loading;
		const pinned = dirty;
		if (!ready) return;
		if (pinned) {
			// Pin already on the map at mount → frame it once. Fresh commits are
			// framed in setHere(); this branch only covers the pre-existing pin.
			if (!pinFramed) {
				pinFramed = true;
				untrack(() => framePin(center));
			}
			return;
		}
		if (!query) return;
		if (query === lastAdminFramed) return;
		untrack(() => scheduleAdminFrame(query));
	});

	function handleSelect(result: GeocodeResult) {
		// Search PANS ONLY (2606-053) — it never commits the pin. Set the live
		// preview (search text) and pan the camera; the agent confirms with
		// "Set this location". No emitChange, no field fill.
		searchDisplay = result.display_name;
		// Remember this point so a follow-up dragend/idle doesn't trigger a
		// redundant reverse-geocode for the same coordinate.
		lastReverseAt = { lat: result.lat, lng: result.lng };
		handle?.setCenter(clampCoords({ lat: result.lat, lng: result.lng }));
	}

	// Explicit "Set this location" — the SINGLE commit affordance (2606-053).
	// Reads the live camera center, commits it as the pin (emitChange → onchange),
	// frames the pin at the location ladder zoom (16), then reverse-geocodes the
	// committed point and — on resolve — fires onaddressresolve so the consumer
	// auto-fills its empty address fields.
	function setHere() {
		if (!handle) return;
		const v = handle.getViewport();
		const loc: LngLat = { lat: v.lat, lng: v.lng };
		emitChange(loc);
		framePin(loc);
		// Force the reverse-geocode + emit even if the point hasn't moved far
		// since the last preview (the commit is an explicit user action).
		scheduleReverseGeocode({ ...v, ...loc }, true);
		announceAndPark('Location set.');
	}

	// MAP-01 "Cancel" — snap the camera back onto the committed pin WITHOUT touching
	// zoom (the user asked to restore the LOCATION, not the zoom; framePin would force
	// zoom 16). setCenter → onmove → cameraCenter re-syncs to the pin → cameraDrifted
	// clears → Set/Cancel auto-hide. The pin (`center`) is left untouched.
	function recenterPin() {
		if (!handle) return;
		handle.setCenter(clampCoords(center));
		announceAndPark('Returned to the set location.');
	}

	// MAP-01 a11y — the Set/Cancel button just clicked is about to unmount (its
	// visibility clears once the camera rests on the pin), which would drop focus to
	// <body>. Move focus to the map (a stable sibling) and announce via the aria-live
	// region so the toggle isn't silent for AT users. NFR: no focus loss on toggle.
	function announceAndPark(message: string) {
		statusMessage = message;
		mapContainer?.focus();
	}

	function handleManualLat(e: Event) {
		const v = parseFloat((e.target as HTMLInputElement).value);
		if (!Number.isFinite(v)) return;
		const next = { lat: v, lng: center.lng };
		emitChange(next);
		// MAP-01 — keep the camera on the manually-entered pin so it doesn't read as
		// "drifted" (which would wrongly offer Set to re-commit the stale camera).
		handle?.setCenter(clampCoords(next));
	}
	function handleManualLng(e: Event) {
		const v = parseFloat((e.target as HTMLInputElement).value);
		if (!Number.isFinite(v)) return;
		const next = { lat: center.lat, lng: v };
		emitChange(next);
		handle?.setCenter(clampCoords(next));
	}

	// MAP-03 — switch the base map type via the in-canvas segmented control.
	// Persists alongside the live camera so map_type round-trips with zoom/center.
	function setMapType(t: MapType) {
		if (!handle) return;
		currentMapType = t;
		handle.setMapType(t);
		const v = handle.getViewport();
		onmapviewchange?.({ zoom: v.zoom, center: [v.lng, v.lat], map_type: t });
	}

	// MAP-03 — custom fullscreen. Fullscreens the WHOLE canvas (map + overlays)
	// so the pin / search / Set / controls stay visible — the native button only
	// covered the bare map div. Esc-to-exit is native to the Fullscreen API; the
	// toggle button is always rendered, so focus is never dropped to <body>.
	function toggleFullscreen() {
		if (!canvasEl) return;
		if (document.fullscreenElement) {
			void document.exitFullscreen();
		} else {
			void canvasEl.requestFullscreen?.();
		}
	}
</script>

<div class="ui-map-picker">
	<div
		bind:this={canvasEl}
		class="ui-map-picker__canvas"
		class:is-fullscreen={isFullscreen}
		style:height
		class:is-loading={loading}
		class:is-error={!!errorMessage}
	>
		<div
			bind:this={mapContainer}
			class="ui-map-picker__map"
			role={readonly ? 'img' : 'application'}
			tabindex="-1"
			aria-label={readonly
				? 'Map showing the property location.'
				: 'Map. Drag or search to position the camera, then press Set this location to place the pin. Or expand the manual coordinates section below.'}
			aria-hidden={!!errorMessage}
		></div>

		{#if loading && !errorMessage}
			<div class="ui-map-picker__overlay ui-map-picker__overlay--loading">Loading map…</div>
		{/if}

		{#if errorMessage}
			<div class="ui-map-picker__overlay ui-map-picker__overlay--error" role="alert">
				<strong>Map unavailable</strong>
				<span>{errorMessage}</span>
				<span class="ui-map-picker__overlay-hint">Enter coordinates manually below.</span>
			</div>
		{:else}
			{#if !loading && !readonly}
				<!-- MAP-03 — search INSIDE the canvas (top-left) so it AND its
				     dropdown (a DOM descendant of this canvas) survive fullscreen;
				     Google's native fullscreen left them out, so they vanished.
				     Read-only previews render none of this chrome. -->
				{#if adapter}
					<div class="ui-map-picker__search">
						<GeocodeSearch
							geocode={adapter}
							onselect={handleSelect}
							{country}
							externalValue={searchDisplay}
						/>
						{#if reverseLoading}
							<span class="ui-map-picker__search-status" aria-live="polite">Locating…</span>
						{/if}
					</div>
				{/if}

				<!-- MAP-03/05 — top-right control cluster: base map-type segmented
				     control + custom fullscreen toggle. Mirrors PolygonPicker's
				     chrome so the two pickers read as one component. -->
				<div class="ui-map-picker__controls">
					<div class="ui-map-picker__maptype" role="group" aria-label="Map type">
						{#each MAP_TYPES as [code, label] (code)}
							<button
								type="button"
								class="ui-map-picker__maptype-btn"
								class:is-active={currentMapType === code}
								aria-pressed={currentMapType === code}
								onclick={() => setMapType(code)}
								title={label}
							>
								{label}
							</button>
						{/each}
					</div>
					<button
						type="button"
						class="ui-map-picker__fullscreen"
						onclick={toggleFullscreen}
						aria-label={isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'}
						aria-pressed={isFullscreen}
						title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
					>
						{#if isFullscreen}
							<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
								<path d="M9 4v3a2 2 0 0 1-2 2H4" />
								<path d="M15 4v3a2 2 0 0 0 2 2h3" />
								<path d="M9 20v-3a2 2 0 0 0-2-2H4" />
								<path d="M15 20v-3a2 2 0 0 1 2-2h3" />
							</svg>
						{:else}
							<svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
								<path d="M4 9V6a2 2 0 0 1 2-2h3" />
								<path d="M20 9V6a2 2 0 0 0-2-2h-3" />
								<path d="M4 15v3a2 2 0 0 0 2 2h3" />
								<path d="M20 15v3a2 2 0 0 1-2 2h-3" />
							</svg>
						{/if}
					</button>
				</div>
			{/if}

			<!-- Center pin. Solid brand teardrop once the location is set; a
			     translucent hollow (stroked) ghost while UNSET, so a reference /
			     default center never reads as a placed pin. -->
			<div class="ui-map-picker__pin" class:is-unset={!dirty} aria-hidden="true">
				<!-- ONE canonical property pin (red teardrop + white circle), shared markers.ts.
				     The tip sits on the map centre via the translate(-50%,-100%) below. -->
				{@html locationPinSvg(30, 40)}
			</div>
			<!-- MAP-01 — Set/Cancel smart visibility (the 2606-053 single-commit
			     affordance, now conditional). "Set this location" shows when there's no
			     pin yet OR the camera drifted off the committed pin; "Cancel" (snap back
			     to the pin, zoom preserved) shows only when a pin exists AND the camera
			     drifted. Both hide once the camera rests on the pin. -->
			{#if !loading && !readonly && (showSet || showCancel)}
				<div class="ui-map-picker__actions">
					{#if showCancel}
						<button type="button" class="ui-map-picker__cancel" onclick={recenterPin}>
							Cancel
						</button>
					{/if}
					{#if showSet}
						<button type="button" class="ui-map-picker__set" onclick={setHere}>
							Set this location
						</button>
					{/if}
				</div>
			{/if}
		{/if}
	</div>
	<span class="ui-map-picker__sr-status" aria-live="polite">{statusMessage}</span>

	{#if !readonly}
	<div class="ui-map-picker__readout">
		{#if errorMessage}
			<label class="ui-map-picker__field">
				<span>Latitude</span>
				<input
					type="number"
					step="0.00001"
					inputmode="decimal"
					name="ui-map-fallback-latitude"
					value={dirty ? center.lat : ''}
					oninput={handleManualLat}
				/>
			</label>
			<label class="ui-map-picker__field">
				<span>Longitude</span>
				<input
					type="number"
					step="0.00001"
					inputmode="decimal"
					name="ui-map-fallback-longitude"
					value={dirty ? center.lng : ''}
					oninput={handleManualLng}
				/>
			</label>
		{:else if dirty}
			<output>
				{center.lat.toFixed(5)}, {center.lng.toFixed(5)}
			</output>
		{:else}
			<output class="ui-map-picker__readout--placeholder">
				Drag the map or search above to position it, then press “Set this location” to place the pin
			</output>
		{/if}
	</div>
	{/if}

	{#if !errorMessage && !readonly}
		<details class="ui-map-picker__manual">
			<summary>Enter coordinates manually</summary>
			<label class="ui-map-picker__field">
				<span>Latitude</span>
				<input
					type="number"
					step="any"
					inputmode="decimal"
					name="ui-map-manual-latitude"
					value={dirty ? center.lat : ''}
					oninput={handleManualLat}
				/>
			</label>
			<label class="ui-map-picker__field">
				<span>Longitude</span>
				<input
					type="number"
					step="any"
					inputmode="decimal"
					name="ui-map-manual-longitude"
					value={dirty ? center.lng : ''}
					oninput={handleManualLng}
				/>
			</label>
		</details>
	{/if}
</div>

<style>
	.ui-map-picker {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		width: 100%;
	}
	.ui-map-picker__canvas {
		position: relative;
		width: 100%;
		height: 100%;
		min-height: 240px;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.5rem;
		overflow: hidden;
		background: var(--color-surface-muted, #f4f4f5);
	}
	.ui-map-picker__map {
		position: absolute;
		inset: 0;
	}
	/* MAP-03 — this Google Maps build (v3.65.3b) ignores both `fullscreenControl:
	   false` AND `disableDefaultUI: true` for the fullscreen control (verified at
	   runtime): the native "Toggle fullscreen view" button persists. Its
	   fullscreen covers only this bare map div, dropping the picker's pin / search
	   / Set overlays — the exact bug MAP-03 fixes. Hide it so only our custom
	   fullscreen (which fullscreens the whole canvas) remains. `.gm-fullscreen-control`
	   is Google's stable control class; scoped to this map so other Google maps
	   on the page are untouched. */
	.ui-map-picker__map :global(.gm-fullscreen-control) {
		display: none !important;
	}
	.ui-map-picker__pin {
		position: absolute;
		top: 50%;
		left: 50%;
		/* Tip on the map centre: the element's bottom-centre sits on the point. */
		transform: translate(-50%, -100%);
		pointer-events: none;
		z-index: 5;
		filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.35));
	}
	/* {@html} content isn't Svelte-scoped, so target the injected canonical pin via :global. */
	.ui-map-picker__pin :global(svg) {
		display: block;
	}
	/* Unset — the location isn't confirmed yet; GHOST the SAME canonical pin (never a
	   different glyph) so a reference / default centre reads as not-yet-placed. */
	.ui-map-picker__pin.is-unset {
		opacity: 0.5;
		filter: grayscale(0.55);
	}
	/* MAP-01 — the commit actions float together at the canvas bottom-center.
	   Conditionally rendered (see the Set/Cancel visibility derivation): "Set this
	   location" places/re-confirms the pin; "Cancel" snaps the camera back to it. */
	.ui-map-picker__actions {
		position: absolute;
		bottom: 0.75rem;
		left: 50%;
		transform: translateX(-50%);
		z-index: 6;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	/* Brand-filled primary pill — the single "confirm this spot" commit action. */
	.ui-map-picker__set {
		padding: 0.4rem 0.9rem;
		border: 1px solid transparent;
		border-radius: 999px;
		background: var(--color-brand, #e5392b);
		color: #fff;
		font: inherit;
		font-weight: 600;
		font-size: 0.8125rem;
		cursor: pointer;
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
		transition: background 0.12s ease;
	}
	.ui-map-picker__set:hover {
		background: #c92e21;
	}
	/* Secondary "revert" pill — neutral surface, sits LEFT of Set (plan MAP-01.6). */
	.ui-map-picker__cancel {
		padding: 0.4rem 0.9rem;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 999px;
		background: var(--color-surface, #fff);
		color: var(--color-text, #1f2937);
		font: inherit;
		font-weight: 600;
		font-size: 0.8125rem;
		cursor: pointer;
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.2);
		transition: background 0.12s ease;
	}
	.ui-map-picker__cancel:hover {
		background: var(--color-surface-muted, #f4f4f5);
	}
	/* MAP-01 a11y — visually-hidden live region announcing Set/Cancel results. */
	.ui-map-picker__sr-status {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border: 0;
	}
	.ui-map-picker__overlay {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.25rem;
		padding: 1rem;
		text-align: center;
		background: rgba(244, 244, 245, 0.92);
		z-index: 6;
	}
	.ui-map-picker__overlay--loading {
		color: var(--color-muted, #71717a);
	}
	.ui-map-picker__overlay--error {
		color: var(--color-error, #b91c1c);
	}
	.ui-map-picker__overlay-hint {
		color: var(--color-muted, #71717a);
		font-size: 0.875rem;
	}
	.ui-map-picker__readout {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.875rem;
		color: var(--color-muted, #71717a);
	}
	.ui-map-picker__readout--placeholder {
		font-style: italic;
	}
	.ui-map-picker__field {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		flex: 1 1 12rem;
	}
	.ui-map-picker__field span {
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	.ui-map-picker__field input {
		padding: 0.4rem 0.5rem;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.375rem;
		font: inherit;
	}
	.ui-map-picker__manual {
		margin-top: 0.5rem;
	}
	.ui-map-picker__manual summary {
		cursor: pointer;
		font-size: 0.875rem;
		color: var(--color-muted, #71717a);
		user-select: none;
		padding: 0.25rem 0;
	}
	.ui-map-picker__manual summary:hover {
		color: var(--color-text, #1f2937);
	}
	.ui-map-picker__manual[open] {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	/* MAP-03 — search overlay, top-left INSIDE the canvas (was an external row
	   above the map). Constrained width leaves the top-right corner clear for
	   the map-type + fullscreen cluster. Its dropdown is a DOM descendant, so
	   it survives fullscreen. */
	.ui-map-picker__search {
		position: absolute;
		top: 0.625rem;
		left: 0.625rem;
		z-index: 7;
		width: min(20rem, calc(100% - 15rem));
	}
	.ui-map-picker__search-status {
		position: absolute;
		right: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		font-size: 0.75rem;
		color: var(--color-muted, #71717a);
		pointer-events: none;
		background: var(--color-surface, #fff);
		padding-left: 0.5rem;
	}

	/* MAP-03 — fullscreen: fill the screen, overriding the inline height. The
	   `is-fullscreen` class (driven by the fullscreenchange listener) drives
	   sizing — more reliable cross-browser than the :fullscreen pseudo. */
	.ui-map-picker__canvas.is-fullscreen {
		width: 100%;
		height: 100% !important;
		border: 0;
		border-radius: 0;
	}

	/* MAP-03/05 — top-right control cluster: map-type switcher + fullscreen. */
	.ui-map-picker__controls {
		position: absolute;
		top: 0.625rem;
		right: 0.625rem;
		z-index: 7;
		display: flex;
		align-items: flex-start;
		gap: 0.375rem;
	}
	/* Map-type segmented control — mirrors PolygonPicker's switcher (MAP-05). */
	.ui-map-picker__maptype {
		display: inline-flex;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.375rem;
		overflow: hidden;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
	}
	.ui-map-picker__maptype-btn {
		padding: 0.3rem 0.55rem;
		font-size: 0.75rem;
		font-weight: 500;
		color: var(--color-text-muted, #4b5563);
		background: var(--color-surface, #fff);
		border: 0;
		border-right: 1px solid var(--color-border, #d4d4d8);
		cursor: pointer;
		transition: background 120ms ease, color 120ms ease;
	}
	.ui-map-picker__maptype-btn:last-child {
		border-right: 0;
	}
	.ui-map-picker__maptype-btn:hover {
		background: var(--color-surface-muted, #f4f4f5);
		color: var(--color-text, #18181b);
	}
	.ui-map-picker__maptype-btn.is-active {
		background: var(--color-accent, #18181b);
		color: var(--color-on-accent, #fff);
	}
	/* Custom fullscreen toggle — compact square icon button (replaces Google's
	   native button, which fullscreened only the bare map div). */
	.ui-map-picker__fullscreen {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		padding: 0;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.375rem;
		background: var(--color-surface, #fff);
		color: var(--color-text, #111827);
		cursor: pointer;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
		transition: background 120ms ease;
	}
	.ui-map-picker__fullscreen:hover {
		background: var(--color-surface-muted, #f4f4f5);
	}
	.ui-map-picker__fullscreen svg {
		display: block;
		fill: none;
		stroke: currentColor;
		stroke-width: 2;
		stroke-linecap: round;
		stroke-linejoin: round;
	}
</style>
