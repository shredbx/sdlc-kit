import type { Snippet } from 'svelte';
import type { GeoCoordinate } from '@sbx/units';

// Coordinates ===============================================================

export interface LngLat {
	lat: number;
	lng: number;
}

export interface MapViewport {
	lat: number;
	lng: number;
	zoom: number;
}

export interface BoundingBox {
	south: number;
	west: number;
	north: number;
	east: number;
}

// Geocoding =================================================================

export interface GeocodeQuery {
	q: string;
	limit?: number;
	bias?: BoundingBox;
	country?: string;
}

// Structured address parsed from a geocode response. Field semantics map to
// Google address_components types so other adapters (Mapbox, etc.) must
// normalize into this shape. All fields optional — providers vary.
export interface StructuredAddress {
	street_number?: string;
	route?: string;
	sub_locality?: string;
	locality?: string;
	admin_area_2?: string;
	admin_area_1?: string;
	postal_code?: string;
	country?: string;
	country_code?: string;
}

export interface GeocodeResult {
	display_name: string;
	lat: number;
	lng: number;
	components?: StructuredAddress;
	// Recommended camera frame for this result. Google returns
	// `geometry.viewport` (always) / `geometry.bounds` (optional) as a
	// LatLngBounds; the adapter normalizes it into this south/west/north/east
	// box so consumers can fitBounds() to frame a place (a district, a
	// province) without a center+zoom guess. Absent when the provider gives
	// no viewport.
	bounds?: BoundingBox;
}

export interface ReverseGeocodeQuery {
	lat: number;
	lng: number;
	// Bias the result to a region (ccTLD hint, e.g. 'th'). Optional.
	country?: string;
}

export interface GeocodeService {
	search(query: GeocodeQuery): Promise<GeocodeResult[]>;
	// Reverse: coordinate → structured address. Returns null when the
	// provider returns ZERO_RESULTS (point in the ocean, etc.). Throws
	// on transport / API errors so the caller can surface them.
	reverseGeocode(query: ReverseGeocodeQuery): Promise<GeocodeResult | null>;
}

// Diff confirmation =========================================================

// One row in the AddressDiffConfirm modal. `isChange` is computed by the
// consumer (current vs. next normalized comparison) so the modal stays
// presentation-only.
export interface AddressDiffField {
	key: string;
	label: string;
	current: string;
	next: string;
	isChange: boolean;
}

// Provider selection ========================================================

export type MapProvider = 'google' | 'mapbox';
export type MapSurface = 'admin' | 'public';

// Map view state ============================================================

// Persisted camera state for a map (LocationPicker / PolygonPicker). Mirrors
// the wire shape of pkg/address.MapView so the JSONB round-trip through
// property_locations.map_view is lossless.
//
// Coordinate ordering on `center` is [lng, lat] for GeoJSON parity; UI
// layers that use {lat, lng} object literals swap at the boundary.
export interface MapViewState {
	zoom?: number;
	center?: [number, number]; // [longitude, latitude]
	map_type?: string; // roadmap | satellite | terrain | hybrid (Google) / streets-v12 etc. (Mapbox)
	heading?: number; // 0-360 (Google satellite only)
	tilt?: number; // 0-45 (Google satellite zoom ≥ 18 only)
}

// Adapter contract ==========================================================

export interface MountOpts {
	container: HTMLElement;
	center: LngLat;
	zoom: number;
	apiKey: string;
	style?: string;
	// Initial base map type (Google: roadmap | satellite | terrain | hybrid;
	// Mapbox style id). Optional — adapters fall back to their provider default
	// when unset. Admin pickers pass 'hybrid' (satellite + labels reads best
	// for placing a property pin).
	mapType?: string;
	// Read-only display. When true the adapter mounts a NON-interactive map:
	// no zoom/pan gestures, no control chrome (zoom, keyboard, etc.) — a static
	// pin preview. The picker also suppresses its own overlays (search, map-type,
	// fullscreen, Set/Cancel). Default false = the full interactive editor.
	readonly?: boolean;
	// Anchor zoom at the map CENTER instead of the cursor. Google's default
	// scroll-wheel / double-click zoom recenters toward the pointer, which drags
	// the geographic center — and a center pin — off the spot being placed. Point
	// pickers (LocationPicker) set this so zooming in/out keeps the pin over the
	// same location. Default false = Google's native cursor-anchored zoom.
	lockCenterOnZoom?: boolean;
}

export interface MapHandle {
	provider: MapProvider;
	destroy(): void;
	setCenter(loc: LngLat): void;
	setZoom(zoom: number): void;
	// Switch the base map type (Google ids: roadmap | satellite | terrain |
	// hybrid). Drives the in-canvas map-type segmented control shared by the
	// pickers (MAP-03/05).
	setMapType(mapType: string): void;
	// Re-read the container size and repaint. Call after the map's container
	// changes dimensions outside the normal layout flow — e.g. entering /
	// exiting the Fullscreen API — so the provider doesn't keep painting tiles
	// at the stale size (blank / clipped map). Implementations restore the
	// center (a raw resize pins the top-left corner, drifting the center).
	resize(): void;
	getViewport(): MapViewport;
	// Frame the camera to a south/west/north/east box. `padding` is the inset
	// (px) kept between the box and the map edge. Used to frame an admin region
	// (district / province) from a geocode result's `bounds`, mirroring the
	// PolygonPicker's fitBounds-over-LatLngBounds behaviour but exposed through
	// the handle contract so any consumer (not just the polygon draw) can frame.
	fitBounds(box: BoundingBox, padding?: number): void;
	// Fires after ANY viewport change (drag, zoom, programmatic). Use for
	// keeping pin/readout in sync with whatever the map is showing.
	onmove?: ((viewport: MapViewport) => void) | undefined;
	// Fires ONLY after a user-initiated drag stops. Distinct from `onmove`
	// because reverse-geocoding should never re-trigger on programmatic
	// setCenter (e.g., after a forward search) — it would just re-confirm
	// the same address while burning API quota.
	ondragend?: ((viewport: MapViewport) => void) | undefined;
}

export interface MapAdapter extends GeocodeService {
	mount(opts: MountOpts): Promise<MapHandle>;
	destroy(handle: MapHandle): void;
}

// LocationPicker props ======================================================

// Polygon picker ============================================================

// GeoJSON Polygon — re-exported here as the workspace canonical shape
// for parcel boundaries. Same structure as @sbx/units.GeoJSONPolygon;
// kept in two places only because @sbx/units is the pure math lib and
// shouldn't pull Svelte typing concerns.
export interface PolygonRing extends Array<[number, number]> {} // [lng, lat][]
export interface MapPolygon {
	type: 'Polygon';
	coordinates: PolygonRing[];
}

export interface PolygonPickerProps {
	polygon?: MapPolygon | null;
	center?: LngLat;
	defaultCenter?: LngLat;
	defaultZoom?: number;
	height?: string;
	// Read-only display. When true the picker mounts a NON-interactive map (no
	// pan/zoom gestures, no draw/edit tools, no control chrome) that just shows the
	// saved boundary — the Land card READ view. Mirrors LocationPicker.readonly so
	// the plot read map matches the address read map exactly. Default false.
	readonly?: boolean;
	apiKey?: string;
	provider?: MapProvider;
	surface?: MapSurface;
	// Persisted viewport — restored on mount if present. When the user has
	// drawn a polygon the picker still calls fitBounds() over it (the
	// polygon's extent wins over the saved view). Empty-state pickers use
	// the persisted view as their initial camera.
	mapView?: MapViewState;
	// Location gate. When true and `center` is null/undefined the Google map
	// is NOT mounted — a placeholder empty-state renders instead (no API
	// call is made). Dropping a pin upstream (which sets `center`) opens the
	// map. When false (default) the picker mounts with its fallback center.
	requireCenter?: boolean;
	// Default placeholder copy shown by the location gate. Overridden by the
	// `noLocation` snippet when one is supplied.
	noLocationMessage?: string;
	// Richer location-gate placeholder. When provided it is rendered in place
	// of `noLocationMessage`, letting the consumer supply a message + an
	// action (e.g. a "Go to pin" button that scrolls to the address map).
	noLocation?: Snippet;
	// Fires after every vertex edit (drag / add midpoint / delete). null
	// means the polygon was deleted. `areaSqm` is the shoelace area of the
	// current ring — consumers derive a land-size proposal from it.
	onpolygonchange?: (polygon: MapPolygon | null, areaSqm: number) => void;
	// Fires when the agent pans / zooms / changes mapType / heading /
	// tilt. Consumers persist this on the property for next-visit restore.
	onmapviewchange?: (view: MapViewState) => void;
}

// Admin-cascade values driving the empty-state map frame (see
// LocationPickerProps.adminFrame). All optional — the picker frames to the
// deepest non-empty level. `country` is the human label (e.g. "Thailand"),
// NOT the ISO code, so it assembles into a geocodable query string.
export interface AdminFrameQuery {
	subDistrict?: string;
	district?: string;
	province?: string;
	country?: string;
}

export interface LocationPickerProps {
	lat?: number;
	lng?: number;
	provider?: MapProvider;
	surface?: MapSurface;
	defaultCenter?: LngLat;
	defaultZoom?: number;
	height?: string;
	// Persisted viewport — restored on mount if present. The pin and the
	// camera are independent: lat/lng pins the property, mapView centers
	// the camera. Empty-state pickers (no lat/lng yet) use mapView as the
	// initial camera so the agent's last work area appears first.
	mapView?: MapViewState;
	onchange?: (loc: LngLat) => void;
	// Fires after the agent pans / zooms / changes mapType. Consumers
	// persist this on the property for next-visit restore.
	onmapviewchange?: (view: MapViewState) => void;
	// Fires AUTOMATICALLY after the committed pin's reverse-geocode resolves
	// (an explicit Set-this-location action — not a drag/search preview). The
	// consumer fills its OWN address fields from the structured result, empty-
	// only (current value wins). The picker never touches the form; it only
	// emits the resolved address. Replaces the former "Use this address" diff
	// modal flow (2606-053).
	onaddressresolve?: (result: GeocodeResult) => void;
	// Country bias for forward AND reverse geocoding (ISO ccTLD: 'th',
	// 'us', etc.). Default unset = global search.
	country?: string;
	// Minimum zoom for reverse-geocoding the map center after a drag.
	// Below this, reverse-geocode is skipped and the search input is
	// cleared (avoids polluting the form with continent/country labels).
	reverseGeocodeMinZoom?: number;
	// Debounce window (ms) between drag-end and the reverse-geocode call.
	reverseGeocodeDebounceMs?: number;
	// Admin-cascade framing (optional). When NO pin is set yet, the picker
	// debounce-geocodes the DEEPEST non-empty admin level and centers the camera
	// on it at a FIXED ladder zoom (province=9 ▸ district=11 ▸ subDistrict=13) —
	// so the camera frames the region the agent has filled in the cascade above,
	// tightening as deeper levels are picked. Once a pin is confirmed via the
	// "Set this location" button, the pin wins and frames at the location zoom
	// (16). Pass the raw cascade values; the picker assembles the query itself.
	adminFrame?: AdminFrameQuery;
	// Provider API key passed in by the consumer. The package stays
	// SvelteKit-agnostic; consumers in SvelteKit projects read the value
	// from $env/dynamic/public and pass it through. When omitted, the
	// picker falls back to import.meta.env.PUBLIC_GOOGLE_MAPS_API_KEY /
	// PUBLIC_MAPBOX_TOKEN — works in vite dev mode, fails in production
	// Node builds where the env is only available at runtime.
	apiKey?: string;
	// Read-only display. When true the picker is a static pin preview: the map
	// is non-interactive (no pan/zoom gestures) and renders ZERO controls — no
	// search, no map-type, no fullscreen, no zoom, no Set/Cancel, no manual
	// coordinates. Used for card READ views where the coordinates are shown as
	// text and the map is purely illustrative. Default false = the editor.
	readonly?: boolean;
}

// Parcel viewer (read-only fullscreen) — provider-agnostic ===================

// Which map a SURFACE gets — by DOMAIN, not by user role. A public page always uses
// the public provider (MapLibre terrain), even for an admin visitor; the Google map
// appears only on admin pages. New providers are added in the factory, never at the
// call site (#0303 / Decision #0304).
export type MapViewerDomain = 'public' | 'admin';

export interface MapViewerMountOpts {
	/** The element the adapter mounts its map into. */
	container: HTMLElement;
	/** Location pin. */
	center: LngLat;
	/** Parcel boundary ([lng,lat] ring); empty for a pin-only viewer (no plot/legend). */
	ring: GeoCoordinate[];
	/** Provider API key (Google admin adapter); ignored by keyless providers. */
	apiKey?: string;
	/** Render the location marker as an APPROXIMATE area indicator — a soft, semi-transparent
	 *  dashed halo (no precise point) — instead of the exact pin. Set when the map is framed on
	 *  an AREA centroid because the exact location is private or absent (task 2607-115). The
	 *  plot + highlight semantics are unchanged; only the marker glyph differs. */
	approximate?: boolean;
	/** Initial camera zoom for a PIN-ONLY frame (no plot). Used to frame an AREA centroid at its
	 *  level (province ≈ 9 ▸ district ≈ 11 ▸ sub-district ≈ 13). When a plot ring is present the
	 *  adapter fitBounds() to it and this is ignored. Defaults to the adapter's location zoom. */
	zoom?: number;
	/** Area NAME shown as the caption on the approximate-area marker (only used when
	 *  `approximate` is set) — e.g. the sub-district the centroid represents. */
	areaLabel?: string;
	/** Fires when the user hovers a map vertex → the viewer highlights that edge's
	 *  legend row (the inverse of {@link MapViewerAdapter.setHighlightEdge}). */
	onNodeHover?: (edgeIndex: number | null) => void;
	/** Fires on a FATAL load failure (style/network/no-WebGL) so the viewer can show
	 *  its static fallback. Non-fatal post-load tile errors must NOT call this. */
	onError?: () => void;
}

// The map side of the fullscreen viewer. The MapViewer shell owns the chrome (header,
// close, "Get directions", the PlotLegend, the highlight STATE, the map-type switcher
// UI); the adapter owns the MAP — tiles, the red pin, the blue plot + casing, the
// highlighted edge, the camera framing, and (admin) the base-map types. This is the
// "doesn't matter what map" boundary: pin + plot + highlight are identical across
// providers; only the drawing differs. Resolve one via {@link createMapViewerAdapter}.
export interface MapViewerAdapter {
	/** Mount the map, draw the pin + plot, frame them. Rejects only on a synchronous
	 *  setup error; async/load failures surface via {@link MapViewerMountOpts.onError}. */
	mount(opts: MapViewerMountOpts): Promise<void>;
	/** Draw (or clear, with null) the highlighted edge — driven by a legend-row hover. */
	setHighlightEdge(edgeIndex: number | null): void;
	/** Base-map types this provider offers (Google: roadmap·satellite·terrain·hybrid;
	 *  the public terrain provider offers none → the shell hides the switcher). */
	readonly mapTypes: readonly string[];
	/** The base map shown initially / after a switch (one of {@link mapTypes}). */
	readonly defaultMapType: string;
	/** Switch the base map (no-op when {@link mapTypes} is empty). */
	setMapType(type: string): void;
	/** Tear down the map, markers, and listeners. Idempotent. */
	destroy(): void;
}

// Board (multi-marker catalog map) ==========================================

// The BOARD is the second map job (task 2607-133), a sibling of the single-property viewer
// above — not an extension of it. A VIEWER frames ONE location and its plot; a BOARD plots
// MANY and owns a selection. Folding them together would force dead `ring`/`setHighlightEdge`
// on the board and a dead `setPoints` on the viewer, which is exactly the "one component
// swapping backends" shape Decision #0303 rejects.

/** One plottable thing on the board: an exact property location, or an AREA group standing in
 *  for the listings whose exact location is private/absent (the 3-entity model, #0354). */
export interface MapMarkerPoint {
	/** Stable identity — the marker set is reconciled by this key across filter changes, so a
	 *  re-filter mutates surviving markers instead of tearing the whole set down. Area keys and
	 *  property ids must not collide (prefix property points, e.g. `p:<id>`). */
	key: string;
	lat: number;
	lng: number;
	/** Pill text — a formatted price, or any short label. */
	label: string;
	/** How many listings this marker stands for. >1 renders the count chip. */
	count?: number;
	/** True when the coordinate is an AREA centroid rather than an exact location. Drives the
	 *  approximate treatment and the accessible name; never leaks a private doorstep. */
	approximate?: boolean;
	/** Accessible name. Required — a map whose markers announce "Map marker" is unusable. */
	ariaLabel: string;
}

/** A camera frame — the exact bounds the board fits instead of "all points". The host computes
 *  it (e.g. the density core of the result set, outliers excluded) so the map opens on the
 *  inventory, not on the empty span between an island and one far-away listing. */
export interface MapBoardFrame {
	sw: LngLat;
	ne: LngLat;
}

export interface MapBoardMountOpts {
	/** The element the adapter mounts its map into. */
	container: HTMLElement;
	/** Initial marker set. Update later via {@link MapBoardAdapter.setPoints}. */
	points: readonly MapMarkerPoint[];
	/** Frame the camera to the points on mount. When false the adapter uses center/zoom. */
	fitToPoints?: boolean;
	/** Fit THIS frame instead of the whole point set (null/undefined = all points). Change it
	 *  later via {@link MapBoardAdapter.setFrame}. */
	frame?: MapBoardFrame | null;
	/** False renders an AMBIENT board — no zoom control, no drag/scroll/keyboard camera (marker
	 *  clicks still work). For decorative surfaces like the homepage band, where a scroll-trap
	 *  and stray zoom chrome cost more than they give. Default true. */
	interactive?: boolean;
	/** When set, activating a marker pans it clear of host chrome overlapping the map's bottom
	 *  edge (the listings rail) — a selected pill must never sit under the cards it opened. */
	activePanPadding?: { bottom: number };
	/** Fallback camera when there are no points (or fitting is off). */
	center?: LngLat;
	zoom?: number;
	/** Inset kept between the fitted points and the map edges — a host whose chrome overlaps
	 *  the map (a filter rail, a floating card) passes asymmetric padding so no marker hides
	 *  underneath it. */
	fitPadding?: { top: number; right: number; bottom: number; left: number };
	/** Fires when a marker is activated by click, Enter or Space. */
	onSelect?: (key: string) => void;
	/** Fires when a merged cluster is activated but zooming CANNOT separate its members (they
	 *  share, or nearly share, one coordinate — the common case for area-based listings). The
	 *  host reveals the members' listings itself; without this handler the adapter can only
	 *  nudge the zoom, which reads as a dead control. `memberKeys` are the merged marker keys. */
	onClusterSelect?: (clusterKey: string, memberKeys: readonly string[]) => void;
	/** Fires when the map background is clicked — the host clears its selection. */
	onDeselect?: () => void;
	/** Fires on a FATAL load failure (style/network/no-WebGL) so the host can degrade. */
	onError?: () => void;
}

// The map side of the catalog board. The HOST owns the chrome (view toggle, filter rail,
// preview card, count line) and the selection STATE; the adapter owns the MAP — tiles, the
// markers, the camera, and turning a click into a key.
export interface MapBoardAdapter {
	/** Mount the map and draw the initial marker set. Rejects only on a synchronous setup
	 *  error; async/load failures surface via {@link MapBoardMountOpts.onError}. */
	mount(opts: MapBoardMountOpts): Promise<void>;
	/** Replace the marker set. Reconciles BY KEY — surviving markers are mutated in place, so a
	 *  filter change never flickers the whole board or drops the selection. */
	setPoints(points: readonly MapMarkerPoint[]): void;
	/** Mark one marker selected (null clears). Raises it above its neighbours. */
	setActive(key: string | null): void;
	/** Replace the camera frame (null = frame all points) and refit. */
	setFrame(frame: MapBoardFrame | null): void;
	/** Frame the camera to the current frame (or all points when none is set). */
	fit(): void;
	/** Tear down the map, every marker, and their listeners. Idempotent. */
	destroy(): void;
}

// Errors ====================================================================

export class MapConfigError extends Error {
	constructor(public readonly reason: string) {
		super(`Map provider config error: ${reason}`);
		this.name = 'MapConfigError';
	}
}

export class MapNotImplementedError extends Error {
	constructor(provider: MapProvider) {
		super(`Map adapter not implemented for provider: ${provider}`);
		this.name = 'MapNotImplementedError';
	}
}
