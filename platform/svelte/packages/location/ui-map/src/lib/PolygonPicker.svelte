<script lang="ts">
	/**
	 * PolygonPicker — draw / edit a property boundary on a Google map.
	 *
	 * Interaction model — a floating three-mode toolbar (top-left) drives a
	 * small state machine over a single boundary polygon:
	 *   - drag (default): map pans freely; the polygon is locked
	 *     (setEditable(false)) so vertices can't be nudged by accident.
	 *   - select: the polygon is editable — drag a vertex, click an edge
	 *     midpoint to insert one. A corner trash control (shown only in
	 *     select mode when a polygon exists) clears the whole boundary and
	 *     returns to the empty state where draw becomes available again.
	 *   - draw (available only when no polygon exists): a CUSTOM native draw
	 *     handler. Each map click appends a vertex; the in-progress shape is
	 *     drawn live as a polyline + vertex markers. With >=3 points, clicking
	 *     the FIRST vertex (or the Finish affordance) closes the ring into the
	 *     editable polygon and switches to select mode. Escape — or switching
	 *     to another mode — cancels an in-progress draw.
	 *
	 * The legacy google.maps.drawing.DrawingManager was removed: it is no
	 * longer shipped on the Maps `weekly` channel, so the custom handler
	 * above is the boundary draw path now. The map-type segmented control
	 * (Map/Sat/Terr/Hyb) floats top-right.
	 *
	 * Location gate — when `requireCenter` is set and no `center` is provided,
	 * the Google map is NOT mounted; a placeholder empty-state renders
	 * instead (the consumer's `noLocation` snippet, or `noLocationMessage`).
	 * The anchor pin is always shown once a `center` exists and the map
	 * renders. On first open with a center and no saved camera the map
	 * centres + zooms to parcel level; `mapView` camera persistence is kept.
	 *
	 * Emits `onpolygonchange(polygon, areaSqm)` on every vertex edit so the
	 * consumer can derive a land size from the boundary in its own land-size
	 * helper.
	 *
	 * Mapbox adapter not yet implemented — this component uses Google
	 * directly for v1. Adding Mapbox would require an equivalent custom draw
	 * over mapbox-gl. See task 2605-067 followups.
	 */

	import { onMount, untrack, type Snippet } from 'svelte';
	import {
		formatLengthAuto,
		haversineMeters,
		polygonAreaSqm,
		polygonFromLatLng,
		polygonPerimeterMeters,
		type GeoCoordinate,
		type LandSizeUnit
	} from '@sbx/units';
	import type { LngLat, MapPolygon, PolygonPickerProps } from './types.js';
	import { MapConfigError } from './types.js';
	import { readBrowserEnv, resolveApiKey, resolveProvider } from './env.js';
	import { loadGoogleMaps } from './google-loader.js';
	import { locationPinDataUri } from './markers.js';
	import PlotLegend from './PlotLegend.svelte';

	const DEFAULT_CENTER: LngLat = { lat: 9.7489, lng: 100.031 };
	const DEFAULT_ZOOM = 15;

	let {
		polygon = null,
		center,
		defaultCenter = DEFAULT_CENTER,
		defaultZoom = DEFAULT_ZOOM,
		height = '320px',
		apiKey: apiKeyProp,
		provider: providerOverride,
		surface,
		readonly = false,
		mapView,
		requireCenter = false,
		noLocationMessage = 'Set the property location first to draw the boundary.',
		noLocation,
		onpolygonchange,
		onmapviewchange
	}: PolygonPickerProps = $props();

	// Minimal Google Maps typings — we only touch the surface we need.
	interface MapTypeStyle {
		featureType?: string;
		elementType?: string;
		stylers: Array<Record<string, string | number>>;
	}
	interface GMap {
		setCenter(loc: LngLat): void;
		setZoom(z: number): void;
		getZoom(): number;
		getCenter(): GLatLng;
		fitBounds(bounds: GBounds, padding?: number): void;
		panTo(loc: LngLat): void;
		getDiv(): HTMLElement;
		addListener(event: string, handler: (e: GMapMouseEvent) => void): { remove(): void };
		setOptions(opts: { styles?: MapTypeStyle[] | null; mapTypeId?: string; draggable?: boolean }): void;
		setMapTypeId(mapTypeId: string): void;
	}
	interface GMapMouseEvent {
		latLng?: GLatLng;
	}
	// Polygon edge/vertex mouse event — Google enriches the plain mouse event
	// on a Polygon with the segment (`edge`) or corner (`vertex`) index that was
	// hit, plus the underlying DOM event (needed to position the context menu at
	// the cursor). `edge` is the index of the segment i→i+1 for an edge click.
	interface GPolyMouseEvent {
		latLng?: GLatLng;
		edge?: number;
		vertex?: number;
		domEvent?: MouseEvent;
	}
	interface GMarker {
		setPosition(loc: LngLat): void;
		setMap(m: GMap | null): void;
		addListener(event: string, handler: () => void): { remove(): void };
	}
	interface GLatLng {
		lat(): number;
		lng(): number;
	}
	interface GBounds {
		extend(loc: LngLat): GBounds;
		isEmpty(): boolean;
	}
	interface GMVCArray<T> {
		getLength(): number;
		getAt(i: number): T;
		forEach(cb: (value: T, i: number) => void): void;
		// Splice a vertex into the path at `index`. Used to add a node on an
		// edge (dbl-click / context menu) — inserting at edge+1 puts the new
		// vertex between the two endpoints of that segment.
		insertAt(index: number, latLng: GLatLng): void;
	}
	interface GPolygon {
		setMap(map: GMap | null): void;
		getPath(): GMVCArray<GLatLng>;
		getPaths(): GMVCArray<GMVCArray<GLatLng>>;
		setEditable(b: boolean): void;
		setDraggable(b: boolean): void;
		setOptions(opts: { editable?: boolean; draggable?: boolean; clickable?: boolean }): void;
		// `dragend` fires with no payload; `dblclick` / `rightclick` carry a
		// GPolyMouseEvent (edge/vertex/latLng/domEvent). The optional arg covers
		// both so we keep a single binder without a cast.
		addListener(
			event: string,
			handler: (e?: GPolyMouseEvent) => void
		): { remove(): void };
	}
	interface GPolygonCtor {
		new (opts: {
			paths: Array<LngLat | { lat: number; lng: number }> | LngLat[][];
			editable?: boolean;
			draggable?: boolean;
			clickable?: boolean;
			strokeColor?: string;
			strokeWeight?: number;
			fillColor?: string;
			fillOpacity?: number;
			zIndex?: number;
		}): GPolygon;
	}
	interface GPolyline {
		setMap(map: GMap | null): void;
		setPath(path: Array<LngLat | { lat: number; lng: number }>): void;
	}
	interface GPolylineCtor {
		new (opts: {
			path: Array<LngLat | { lat: number; lng: number }>;
			strokeColor?: string;
			strokeWeight?: number;
			strokeOpacity?: number;
			clickable?: boolean;
			zIndex?: number;
		}): GPolyline;
	}
	interface GLatLngCtor {
		new (lat: number, lng: number): GLatLng;
	}
	interface GProjection {
		fromLatLngToDivPixel(latLng: GLatLng): { x: number; y: number } | null;
	}
	// Minimal OverlayView surface — Google's docs treat this class as a
	// base for HTML overlays; we extend it inside the component.
	interface GOverlayViewBase {
		setMap(map: GMap | null): void;
		getProjection(): GProjection | undefined;
		getPanes(): { overlayLayer: HTMLElement; floatPane: HTMLElement } | undefined;
	}
	interface GOverlayViewCtor {
		new (): GOverlayViewBase;
		preventMapHitsAndGesturesFrom?(el: Element): void;
	}
	// Google's ControlPosition enum — we only need RIGHT_BOTTOM so the zoom
	// stack settles into one compact corner. The runtime values come from
	// the SDK; we re-declare a positional subset here so the typing matches
	// what we actually pass.
	interface GControlPosition {
		RIGHT_BOTTOM: number;
		RIGHT_TOP: number;
		LEFT_BOTTOM: number;
		LEFT_TOP: number;
	}
	interface GMapOptions {
		center: LngLat;
		zoom: number;
		disableDefaultUI?: boolean;
		mapTypeControl?: boolean;
		streetViewControl?: boolean;
		zoomControl?: boolean;
		zoomControlOptions?: { position: number };
		fullscreenControl?: boolean;
		rotateControl?: boolean;
		scaleControl?: boolean;
		panControl?: boolean;
		keyboardShortcuts?: boolean;
		gestureHandling?: string;
		disableDoubleClickZoom?: boolean;
		clickableIcons?: boolean;
		draggableCursor?: string;
	}
	interface GMarkerCtor {
		new (opts: {
			position: LngLat;
			map: GMap;
			clickable?: boolean;
			opacity?: number;
			zIndex?: number;
			title?: string;
			cursor?: string;
			icon?: unknown;
		}): GMarker;
	}
	interface GMapsNamespace {
		Map: new (container: HTMLElement, opts: GMapOptions) => GMap;
		Marker: GMarkerCtor;
		Polygon: GPolygonCtor;
		Polyline: GPolylineCtor;
		LatLng: GLatLngCtor;
		LatLngBounds: new () => GBounds;
		OverlayView: GOverlayViewCtor;
		ControlPosition: GControlPosition;
		event: {
			clearInstanceListeners(instance: unknown): void;
			// MAP-05 — fire a Google Maps event programmatically. Used to trigger
			// 'resize' after a custom-fullscreen box change so Google repaints its
			// tiles at the new container size (the native button is hidden).
			trigger(instance: unknown, eventName: string): void;
		};
		SymbolPath?: { CIRCLE: number };
		// google.maps.Point — pixel offset used to anchor the canonical pin icon at its tip.
		Point?: new (x: number, y: number) => unknown;
	}

	type DrawMode = 'drag' | 'select' | 'draw';

	let mapContainer = $state<HTMLDivElement | null>(null);
	// MAP-05 — the canvas element we fullscreen. It wraps BOTH the map div AND
	// the picker overlays (boundary toolbar, map-type control, polygon), so they
	// all stay visible in fullscreen — unlike Google's native button, which
	// fullscreens only the bare map div and drops the overlays.
	let canvasEl = $state<HTMLDivElement | null>(null);
	let isFullscreen = $state(false);
	let errorMessage = $state<string | null>(null);
	let loading = $state(true);
	let perimeterMeters = $state(0);
	let sideCount = $state(0);
	let hasPolygon = $state(false);
	let mode = $state<DrawMode>('drag');
	// Vertex count of the in-progress draft (drives the Finish affordance).
	let draftCount = $state(0);

	// Whether the location gate is closed — requireCenter set AND no center.
	// When closed we never mount the Google map (saves an API call) and show
	// the placeholder instead.
	const gateClosed = $derived(requireCenter && !center);

	let mapInstance: GMap | null = null;
	let polygonInstance: GPolygon | null = null;
	let mapsNs: GMapsNamespace | null = null;
	let areaSqmValue = 0; // last computed area — emitted, not rendered in the footer
	let listeners: Array<{ remove(): void }> = [];
	let initStarted = false; // makes initMap idempotent across $effect re-runs
	let disposed = false; // aborts an in-flight async load on unmount
	// Always-on anchor pin at `center`. Kept as a ref so future state changes
	// can adjust it; shown whenever the map renders with a center.
	let locationPinMarker: GMarker | null = null;
	// Currently selected map type — drives the terrain segmented control and
	// the persisted mapView.map_type. Defaults to 'roadmap' (Google's default).
	let currentMapType = $state<'roadmap' | 'satellite' | 'terrain' | 'hybrid'>(
		untrack(() => (mapView?.map_type as 'roadmap' | 'satellite' | 'terrain' | 'hybrid' | undefined) ?? 'hybrid')
	);
	// First-open anchor: on the first render with a center and no saved
	// camera, snap to parcel zoom. Only fires once.
	let firstActivate = true;
	// HTML overlays anchored at edge midpoints showing side length. Rebuilt
	// on every vertex edit — cheap for the vertex counts (~3–50) we expect.
	let edgeOverlays: GOverlayViewBase[] = [];

	// ── Custom-draw in-progress state ─────────────────────────────────────
	// While `mode === 'draw'` the user is laying down vertices. We keep the
	// raw lat/lng points, a live Polyline tracing them, and one Marker per
	// vertex. The FIRST marker carries a click handler that closes the ring
	// (>=3 points). All of this is torn down on finish OR cancel.
	let draftPoints: GLatLng[] = [];
	let draftPolyline: GPolyline | null = null;
	let draftMarkers: GMarker[] = [];
	let draftFirstMarkerListener: { remove(): void } | null = null;
	let drawClickListener: { remove(): void } | null = null;
	let escKeyHandler: ((e: KeyboardEvent) => void) | null = null;

	// ── Live legend overlay (fullscreen-only) ─────────────────────────────
	// `liveRing` mirrors the parcel as it is drawn/edited — it feeds PlotLegend
	// (area / perimeter / lettered survey diagram). Kept as reactive $state so the
	// fullscreen panel updates per draw-click and per vertex edit. Empty (<3 pts)
	// hides the panel (PlotLegend self-hides below 3 vertices anyway). The panel is
	// rendered ONLY in fullscreen so the inline editor's footer/toolbar are untouched.
	let liveRing = $state<GeoCoordinate[]>([]);
	// The panel owns its own unit toggle (m² · rai · ngan · wah · sqft); bound so a
	// switch in the legend persists across re-renders within the session.
	let legendUnit = $state<LandSizeUnit>('sqm');
	// Index of the edge the legend is hovering, or null. Drives the on-map highlight
	// line below so hovering a legend row lights the matching edge on Google's map.
	let legendHighlight = $state<number | null>(null);
	// The transient highlight polyline drawn over the parcel edge under the cursor.
	// Reuses the file's existing GPolyline typing — no new SDK surface introduced.
	let highlightLine: GPolyline | null = null;

	// ── Edge context menu (select mode) ───────────────────────────────────
	// Right-clicking an edge opens a one-item menu ("Add node here") at the
	// cursor. Positions are px relative to `canvasEl` so the menu lives inside
	// the fullscreen box too. `menuLatLng` is the point the node will be added
	// at; `menuEdge` is the resolved segment index (edge+1 is the insert slot).
	let menuOpen = $state(false);
	let menuX = $state(0);
	let menuY = $state(0);
	let menuLatLng: GLatLng | null = null;
	let menuEdge = 0;

	// Max zoom on first-open. 19 is roughly building-level for roadmap /
	// satellite at most lats; high enough to draw a parcel without scrolling.
	const FIRST_ACTIVATE_ZOOM = 19;

	// Plot blue (#2563EB) — the designer-tool plot color, unified across the admin
	// picker, the public static preview, and the MapLibre MapViewer (#0303). Was
	// brand red. Cascades to the live draft polyline + vertex-handle stroke below.
	const POLYGON_STYLE = {
		strokeColor: '#2563EB',
		strokeWeight: 2,
		fillColor: '#2563EB',
		fillOpacity: 0.18,
		zIndex: 4
	};

	function ringFromPath(path: GMVCArray<GLatLng>): GeoCoordinate[] {
		const out: GeoCoordinate[] = [];
		path.forEach((p) => {
			out.push([p.lng(), p.lat()]);
		});
		return out;
	}

	// HTML-overlay class for per-edge length labels. Lazily declared inside a
	// factory because google.maps.OverlayView only exists after the SDK has
	// loaded — a top-level class extending it would fail at import time.
	function createEdgeLabelOverlayClass(ns: GMapsNamespace) {
		return class EdgeLabel extends ns.OverlayView {
			private latLng: GLatLng;
			private text: string;
			private div: HTMLDivElement | null = null;

			constructor(latLng: GLatLng, text: string) {
				super();
				this.latLng = latLng;
				this.text = text;
			}

			onAdd() {
				const div = document.createElement('div');
				div.className = 'ui-map-edge-label';
				div.textContent = this.text;
				this.div = div;
				const panes = this.getPanes();
				if (panes) panes.overlayLayer.appendChild(div);
			}

			draw() {
				if (!this.div) return;
				const proj = this.getProjection();
				if (!proj) return;
				const pt = proj.fromLatLngToDivPixel(this.latLng);
				if (!pt) return;
				this.div.style.left = `${pt.x}px`;
				this.div.style.top = `${pt.y}px`;
			}

			onRemove() {
				if (this.div?.parentNode) this.div.parentNode.removeChild(this.div);
				this.div = null;
			}
		};
	}

	let EdgeLabelClass: ReturnType<typeof createEdgeLabelOverlayClass> | null = null;

	function clearEdgeOverlays() {
		for (const ov of edgeOverlays) ov.setMap(null);
		edgeOverlays = [];
	}

	// Rebuild edge labels from the current polygon vertices. Each label sits
	// at the midpoint of an edge and shows its haversine length.
	function rebuildEdgeOverlays() {
		clearEdgeOverlays();
		if (!polygonInstance || !mapsNs || !mapInstance || !EdgeLabelClass) return;
		const ring = ringFromPath(polygonInstance.getPath());
		if (ring.length < 2) return;
		const n = ring.length;
		const next: GOverlayViewBase[] = [];
		for (let i = 0; i < n; i++) {
			const [lng1, lat1] = ring[i];
			const [lng2, lat2] = ring[(i + 1) % n];
			const meters = haversineMeters([lng1, lat1], [lng2, lat2]);
			if (meters < 1) continue; // skip degenerate edges
			const midLng = (lng1 + lng2) / 2;
			const midLat = (lat1 + lat2) / 2;
			const latLng = new mapsNs.LatLng(midLat, midLng);
			const overlay = new EdgeLabelClass(latLng, formatLengthAuto(meters));
			overlay.setMap(mapInstance);
			next.push(overlay);
		}
		edgeOverlays = next;
	}

	function recenterOnPolygon() {
		if (!polygonInstance || !mapsNs || !mapInstance) return;
		const ring = ringFromPath(polygonInstance.getPath());
		if (ring.length < 3) return;
		const bounds = new mapsNs.LatLngBounds();
		for (const [lng, lat] of ring) bounds.extend({ lat, lng });
		if (!bounds.isEmpty()) mapInstance.fitBounds(bounds, 60);
	}

	function recomputeArea() {
		if (!polygonInstance) {
			areaSqmValue = 0;
			perimeterMeters = 0;
			sideCount = 0;
			hasPolygon = false;
			clearEdgeOverlays();
			// No committed parcel → the live legend has nothing to show.
			liveRing = [];
			onpolygonchange?.(null, 0);
			return;
		}
		const ring = ringFromPath(polygonInstance.getPath());
		const area = polygonAreaSqm(ring);
		const perim = polygonPerimeterMeters(ring);
		areaSqmValue = area;
		perimeterMeters = perim;
		sideCount = ring.length;
		hasPolygon = ring.length >= 3;
		// Feed the live legend from the committed polygon (post-edit). Below 3
		// vertices we clear it so the panel hides rather than render a degenerate plot.
		liveRing = ring.length >= 3 ? ring : [];
		rebuildEdgeOverlays();
		const next: MapPolygon | null = hasPolygon
			? polygonFromLatLng(ring.map(([lng, lat]) => ({ lat, lng })))
			: null;
		onpolygonchange?.(next, area);
	}

	function bindPolygonListeners() {
		if (!polygonInstance) return;
		// Path mutations (drag vertex, add midpoint, delete vertex). Path
		// itself is an MVCArray; we listen on its set_at / insert_at /
		// remove_at events plus the polygon's dragend.
		const path = polygonInstance.getPath() as unknown as {
			addListener(event: string, handler: () => void): { remove(): void };
		};
		listeners.push(
			path.addListener('set_at', recomputeArea),
			path.addListener('insert_at', recomputeArea),
			path.addListener('remove_at', recomputeArea),
			polygonInstance.addListener('dragend', recomputeArea)
		);
	}

	// ── Add-node-on-edge (LandCanvas-style) ───────────────────────────────
	// Two ways to add a vertex on an edge in select mode: double-click the edge,
	// or right-click → "Add node here". Both resolve a segment index then splice
	// a vertex into the path with insertAt(edge+1, latLng).

	// Squared distance from point p to the segment a→b in planar lng/lat space.
	// A planar approximation is fine at parcel scale (sub-km) where lng/lat
	// distortion is negligible; we only need the *relative* nearest segment, not
	// a true geodesic distance. Returns the squared distance to avoid a sqrt.
	function segDistSq(
		px: number,
		py: number,
		ax: number,
		ay: number,
		bx: number,
		by: number
	): number {
		const dx = bx - ax;
		const dy = by - ay;
		const lenSq = dx * dx + dy * dy;
		// Degenerate (zero-length) edge → distance to the endpoint.
		const t = lenSq === 0 ? 0 : Math.max(0, Math.min(1, ((px - ax) * dx + (py - ay) * dy) / lenSq));
		const cx = ax + t * dx;
		const cy = ay + t * dy;
		const ex = px - cx;
		const ey = py - cy;
		return ex * ex + ey * ey;
	}

	// Resolve the segment index for a click. Prefer Google's `edge` (the exact
	// segment hit); fall back to the nearest segment of the current ring when it
	// is null/undefined (e.g. a dbl-click that lands slightly off the line).
	function resolveEdgeIndex(e: GPolyMouseEvent, clicked: GLatLng): number {
		if (typeof e.edge === 'number') return e.edge;
		if (!polygonInstance) return 0;
		const ring = ringFromPath(polygonInstance.getPath()); // [lng, lat] pairs
		const n = ring.length;
		if (n < 2) return 0;
		const px = clicked.lng();
		const py = clicked.lat();
		let best = 0;
		let bestSq = Infinity;
		for (let i = 0; i < n; i++) {
			const [ax, ay] = ring[i];
			const [bx, by] = ring[(i + 1) % n];
			const d = segDistSq(px, py, ax, ay, bx, by);
			if (d < bestSq) {
				bestSq = d;
				best = i;
			}
		}
		return best;
	}

	// Splice a node into the path between the two endpoints of `edgeIndex`. The
	// path's insert_at listener fires recomputeArea; we also call it directly so
	// area/legend/edge-labels update even if that listener path ever changes.
	function insertNodeAt(edgeIndex: number, latLng: GLatLng) {
		if (!polygonInstance) return;
		const path = polygonInstance.getPath();
		path.insertAt(edgeIndex + 1, latLng);
		recomputeArea();
	}

	// dbl-click an edge → insert immediately at the clicked point. Guarded on
	// select mode so dbl-clicks in drag/draw never mutate the boundary.
	function handlePolygonDblClick(e?: GPolyMouseEvent) {
		if (mode !== 'select' || !e?.latLng || !polygonInstance) return;
		closeEdgeMenu();
		insertNodeAt(resolveEdgeIndex(e, e.latLng), e.latLng);
	}

	// right-click an edge → open the "Add node here" menu at the cursor. We stash
	// the clicked point + resolved edge so the menu action can splice the node.
	function handlePolygonRightClick(e?: GPolyMouseEvent) {
		if (mode !== 'select' || !e?.latLng || !polygonInstance) return;
		menuLatLng = e.latLng;
		menuEdge = resolveEdgeIndex(e, e.latLng);
		// Position at the cursor relative to the canvas box (works in fullscreen,
		// where clientX/Y are still viewport-relative and the canvas fills it).
		const rect = canvasEl?.getBoundingClientRect();
		const dom = e.domEvent;
		if (rect && dom) {
			menuX = dom.clientX - rect.left;
			menuY = dom.clientY - rect.top;
		} else if (rect) {
			// Fallback — centre the menu in the canvas if the DOM event is absent.
			menuX = rect.width / 2;
			menuY = rect.height / 2;
		}
		menuOpen = true;
	}

	function closeEdgeMenu() {
		if (!menuOpen) return;
		menuOpen = false;
		menuLatLng = null;
	}

	// Menu action — splice the stored node, then close.
	function addNodeFromMenu() {
		if (menuLatLng) insertNodeAt(menuEdge, menuLatLng);
		closeEdgeMenu();
	}

	// Wire the dbl-click / right-click edge listeners on the live polygon, plus
	// the map-move listeners that dismiss an open menu. Called wherever a polygon
	// is (re)created (loadExistingPolygon, finishDraw). Handles land in the shared
	// `listeners` array so onMount cleanup tears them down — no leaks.
	function bindEdgeAddListeners() {
		if (!polygonInstance || !mapInstance) return;
		listeners.push(
			polygonInstance.addListener('dblclick', handlePolygonDblClick),
			polygonInstance.addListener('rightclick', handlePolygonRightClick),
			// Any map interaction dismisses the menu so it never lingers detached
			// from its cursor anchor after the camera moves.
			mapInstance.addListener('click', closeEdgeMenu),
			mapInstance.addListener('dragstart', closeEdgeMenu),
			mapInstance.addListener('zoom_changed', closeEdgeMenu)
		);
	}

	// Apply the interactivity a mode implies. select → editable + clickable
	// (drag a vertex, click an edge midpoint). drag / draw → fully inert
	// (editable off, clickable off) so pan gestures and draw clicks pass
	// straight through to the map instead of being swallowed by the polygon.
	// Whole-shape drag is never offered — vertices move, the parcel doesn't.
	function applyModeToPolygon() {
		if (!polygonInstance) return;
		const selecting = mode === 'select';
		polygonInstance.setOptions({
			editable: selecting,
			draggable: false,
			clickable: selecting
		});
	}

	function loadExistingPolygon(ns: GMapsNamespace, map: GMap, p: MapPolygon) {
		const ring = p.coordinates[0];
		// Filter the closing duplicate vertex if present — google.maps
		// renders + edits without it.
		const last = ring[ring.length - 1];
		const first = ring[0];
		const isClosed = ring.length > 1 && first[0] === last[0] && first[1] === last[1];
		const open = isClosed ? ring.slice(0, -1) : ring;
		const path = open.map(([lng, lat]) => ({ lat, lng }));
		polygonInstance = new ns.Polygon({
			paths: path,
			editable: false,
			draggable: false,
			clickable: false,
			...POLYGON_STYLE
		});
		polygonInstance.setMap(map);
		bindPolygonListeners();
		bindEdgeAddListeners();
		recomputeArea();

		// Fit map viewport to the polygon bounds.
		const bounds = new ns.LatLngBounds();
		for (const v of path) bounds.extend(v);
		if (!bounds.isEmpty()) map.fitBounds(bounds, 40);
	}

	// ── Mode transitions ──────────────────────────────────────────────────
	function setMode(next: DrawMode) {
		if (next === mode) return;
		// Leaving draw mid-shape cancels the draft.
		if (mode === 'draw' && next !== 'draw') cancelDraw();
		// draw is only meaningful when there's no polygon yet.
		if (next === 'draw' && polygonInstance) return;
		mode = next;
		// Pan is only enabled in drag mode; select/draw lock the map drag so
		// vertex edits + draw clicks aren't swallowed by a pan gesture.
		mapInstance?.setOptions({ draggable: mode === 'drag' });
		applyModeToPolygon();
		if (next === 'draw') enterDraw();
	}

	// Begin a fresh draft: clear any prior in-progress overlays, attach the
	// map click listener, and arm Escape-to-cancel.
	function enterDraw() {
		if (!mapInstance || !mapsNs) return;
		clearDraft();
		drawClickListener = mapInstance.addListener('click', (e: GMapMouseEvent) => {
			if (!e.latLng) return;
			addDraftPoint(e.latLng);
		});
		escKeyHandler = (e: KeyboardEvent) => {
			if (e.key === 'Escape') {
				cancelDraw();
				setMode('drag');
			}
		};
		window.addEventListener('keydown', escKeyHandler);
	}

	function addDraftPoint(latLng: GLatLng) {
		if (!mapInstance || !mapsNs) return;
		draftPoints.push(latLng);
		draftCount = draftPoints.length;
		// Mirror the in-progress draft into the live legend so the fullscreen panel
		// updates per click while drawing (before the ring is committed to a polygon).
		liveRing = draftPoints.map((p) => [p.lng(), p.lat()] as GeoCoordinate);
		// Live polyline tracing the draft.
		const pathLiterals = draftPoints.map((p) => ({ lat: p.lat(), lng: p.lng() }));
		if (!draftPolyline) {
			draftPolyline = new mapsNs.Polyline({
				path: pathLiterals,
				strokeColor: POLYGON_STYLE.strokeColor,
				strokeWeight: POLYGON_STYLE.strokeWeight,
				strokeOpacity: 0.9,
				clickable: false,
				zIndex: 5
			});
			draftPolyline.setMap(mapInstance);
		} else {
			draftPolyline.setPath(pathLiterals);
		}
		// Vertex marker. The FIRST one closes the ring when >=3 points exist.
		const idx = draftPoints.length - 1;
		const marker = new mapsNs.Marker({
			position: { lat: latLng.lat(), lng: latLng.lng() },
			map: mapInstance,
			clickable: true,
			zIndex: 6,
			cursor: idx === 0 ? 'pointer' : 'default',
			title: idx === 0 ? 'Click to close the boundary' : undefined,
			icon: mapsNs.SymbolPath
				? {
						path: mapsNs.SymbolPath.CIRCLE,
						scale: idx === 0 ? 6 : 4,
						fillColor: '#ffffff',
						fillOpacity: 1,
						strokeColor: POLYGON_STYLE.strokeColor,
						strokeWeight: 2
					}
				: undefined
		});
		if (idx === 0) {
			draftFirstMarkerListener = marker.addListener('click', () => {
				if (draftPoints.length >= 3) finishDraw();
			});
		}
		draftMarkers.push(marker);
	}

	// Close the draft into the editable boundary polygon, then switch to
	// select mode so the user can immediately fine-tune vertices.
	function finishDraw() {
		if (!mapInstance || !mapsNs || draftPoints.length < 3) return;
		const path = draftPoints.map((p) => ({ lat: p.lat(), lng: p.lng() }));
		clearDraft();
		// Tear down the draw listeners — finishing exits draw mode.
		removeDrawListeners();
		polygonInstance = new mapsNs.Polygon({
			paths: path,
			editable: false,
			draggable: false,
			clickable: false,
			...POLYGON_STYLE
		});
		polygonInstance.setMap(mapInstance);
		bindPolygonListeners();
		bindEdgeAddListeners();
		recomputeArea();
		// Land on select mode so vertices are immediately fine-tunable —
		// applyModeToPolygon flips the freshly-created polygon to editable.
		mode = 'select';
		applyModeToPolygon();
		mapInstance.setOptions({ draggable: false });
	}

	// Abort an in-progress draft (Escape / mode switch). Idempotent.
	function cancelDraw() {
		clearDraft();
		removeDrawListeners();
	}

	function removeDrawListeners() {
		if (drawClickListener) {
			drawClickListener.remove();
			drawClickListener = null;
		}
		if (escKeyHandler) {
			window.removeEventListener('keydown', escKeyHandler);
			escKeyHandler = null;
		}
	}

	// Remove the draft polyline + vertex markers and reset the draft buffer.
	function clearDraft() {
		if (draftFirstMarkerListener) {
			draftFirstMarkerListener.remove();
			draftFirstMarkerListener = null;
		}
		if (draftPolyline) {
			draftPolyline.setMap(null);
			draftPolyline = null;
		}
		for (const m of draftMarkers) m.setMap(null);
		draftMarkers = [];
		draftPoints = [];
		draftCount = 0;
		// Drop the draft from the live legend. Safe on the finish path: finishDraw()
		// calls clearDraft() BEFORE building the polygon, then recomputeArea() re-seeds
		// liveRing from the committed shape — so the panel never flickers empty.
		liveRing = [];
	}

	function setMapType(t: 'roadmap' | 'satellite' | 'terrain' | 'hybrid') {
		if (!mapInstance) return;
		currentMapType = t;
		mapInstance.setMapTypeId(t);
		// Emit so the consumer persists map_type alongside zoom/center. The
		// idle listener also fires after setMapTypeId, but that path only
		// carries zoom/center — explicit emit ensures map_type round-trips.
		const z = mapInstance.getZoom();
		const c = mapInstance.getCenter();
		onmapviewchange?.({
			zoom: z,
			center: [c.lng(), c.lat()],
			map_type: t
		});
	}

	// MAP-05 — custom fullscreen. Fullscreens the WHOLE canvas (map + overlays)
	// so the boundary toolbar / map-type control / polygon stay visible — the
	// native button only covered the bare map div. Esc-to-exit is native to the
	// Fullscreen API; the toggle button is always rendered, so focus is never
	// dropped to <body>.
	function toggleFullscreen() {
		if (!canvasEl) return;
		if (document.fullscreenElement === canvasEl) {
			void document.exitFullscreen();
		} else {
			void canvasEl.requestFullscreen?.();
		}
	}

	// Clear the whole boundary (the corner trash control, select mode only).
	// Returns to the empty state where draw becomes available again.
	function clearPolygon() {
		cancelDraw();
		// No boundary → no edge to add a node to; dismiss any open menu.
		closeEdgeMenu();
		if (polygonInstance) {
			polygonInstance.setMap(null);
			polygonInstance = null;
		}
		clearEdgeOverlays();
		areaSqmValue = 0;
		perimeterMeters = 0;
		sideCount = 0;
		hasPolygon = false;
		// Boundary gone → hide the live legend and drop any pending edge highlight
		// (the $effect below tears down highlightLine once legendHighlight clears).
		liveRing = [];
		legendHighlight = null;
		mode = 'drag';
		mapInstance?.setOptions({ draggable: true });
		onpolygonchange?.(null, 0);
	}

	// Paint (or clear) the on-map highlight for the legend-hovered edge. The
	// fullscreen PlotLegend emits the edge index it is hovering via onhoveredge →
	// legendHighlight; we mirror it as a bold blue polyline laid over that edge of
	// the live parcel so the diagram row and the map read as one. Reuses POLYGON_STYLE's
	// plot blue (#2563EB) for visual continuity with the boundary itself. Always
	// tears down the prior line first so only one highlight exists at a time.
	function applyLegendHighlight() {
		if (highlightLine) {
			highlightLine.setMap(null);
			highlightLine = null;
		}
		if (legendHighlight == null || !mapsNs || !mapInstance || liveRing.length < 2) return;
		const a = liveRing[legendHighlight];
		const b = liveRing[(legendHighlight + 1) % liveRing.length];
		if (!a || !b) return;
		// liveRing is [lng, lat]; Google's Polyline wants {lat, lng}.
		highlightLine = new mapsNs.Polyline({
			path: [
				{ lat: a[1], lng: a[0] },
				{ lat: b[1], lng: b[0] }
			],
			strokeColor: '#2563EB',
			strokeWeight: 6,
			strokeOpacity: 0.85,
			clickable: false,
			zIndex: 7
		});
		highlightLine.setMap(mapInstance);
	}

	// Re-paint the highlight whenever the hovered edge OR the live ring changes
	// (a vertex edit can shift the edge under a held hover). Both reads are
	// tracked so the effect re-runs on either.
	$effect(() => {
		legendHighlight;
		liveRing;
		applyLegendHighlight();
	});

	// Build the read-only anchor pin at `c`. A compact filled circle with a
	// white ring (SymbolPath.CIRCLE) — deliberately distinct from the big
	// draggable teardrop of the address picker, signalling "reference point;
	// move it on the map above". Falls back to a default marker if SymbolPath
	// isn't available. Used on first mount and by the center $effect.
	function buildLocationPin(c: LngLat) {
		if (!mapsNs || !mapInstance) return;
		locationPinMarker = new mapsNs.Marker({
			position: c,
			map: mapInstance,
			clickable: false,
			zIndex: 1,
			title: 'Property location (read-only — drag on the address map above)',
			// ONE canonical property pin (red teardrop + white circle), shared markers.ts — its
			// TIP on the coordinate (anchor width/2, height). Replaces the old red dot so the plot
			// editor's pin matches the address map + viewers exactly.
			icon: mapsNs.Point
				? { url: locationPinDataUri(), anchor: new mapsNs.Point(14, 38) }
				: { url: locationPinDataUri() }
		});
	}

	// Build the Google map. Idempotent across $effect re-runs via initStarted,
	// abortable mid-load via disposed (set on unmount). Reads `center` at call
	// time so the post-ungate path picks up the freshly-dropped pin.
	async function initMap() {
		if (initStarted || !mapContainer) return;
		initStarted = true;
		loading = true;
		try {
			const env = readBrowserEnv();
			const p = resolveProvider({
				provider: providerOverride,
				surface,
				envProvider: env.provider
			});
			if (p !== 'google') {
				// v1 only ships Google polygon support — see file header.
				// Surface an inline error rather than throwing so the form
				// remains usable.
				throw new MapConfigError('PolygonPicker currently supports Google provider only');
			}
			const key =
				apiKeyProp ||
				resolveApiKey({
					provider: p,
					googleKey: env.googleKey,
					mapboxToken: env.mapboxToken
				});
			if (!key) {
				throw new MapConfigError('PUBLIC_GOOGLE_MAPS_API_KEY is empty');
			}
			if (!mapContainer) {
				throw new MapConfigError('Map container element not ready');
			}

			const ns = (await loadGoogleMaps(key)) as GMapsNamespace;
			if (disposed) return;
			mapsNs = ns;
			EdgeLabelClass = createEdgeLabelOverlayClass(ns);

			// Camera priority on mount:
			//   1. Saved mapView.center / mapView.zoom (agent's last view)
			//   2. center prop (LocationPicker pin propagated to this picker)
			//   3. defaultCenter / defaultZoom (regional fallback)
			const initialCenter = mapView?.center
				? { lat: mapView.center[1], lng: mapView.center[0] }
				: (center ?? defaultCenter);
			const initialZoom = typeof mapView?.zoom === 'number' ? mapView.zoom : defaultZoom;

			// Lock the map chrome to a single compact `+/−` control bottom-right.
			// Setting flags individually leaves stray widgets behind in some
			// Google Maps versions (pan cross, rotate compass) — use
			// `disableDefaultUI` as the floor and opt back into zoom only.
			// Readonly (Land card READ view) mounts a fully inert map — no gestures,
			// no zoom control, no keyboard — mirroring the adapter's readonly chrome so
			// the plot read map matches the address LocationPicker read map exactly.
			const map = new ns.Map(mapContainer, {
				center: initialCenter,
				zoom: initialZoom,
				disableDefaultUI: true,
				clickableIcons: false,
				...(readonly
					? { gestureHandling: 'none', keyboardShortcuts: false, disableDoubleClickZoom: true }
					: {
							zoomControl: true,
							zoomControlOptions: { position: ns.ControlPosition.RIGHT_BOTTOM },
							// Suppress native dbl-click zoom so dbl-click-an-edge (add a
							// node, select mode) doesn't also zoom the map underneath it.
							disableDoubleClickZoom: true
						})
			});
			mapInstance = map;

			// Always-on anchor pin — a compact reference dot at the property
			// location (not an editable teardrop). Shown whenever a center
			// exists; repositioned by the center $effect when the address moves.
			if (center) buildLocationPin(center);

			// Initial map type — persisted mapView wins, else the 'hybrid'
			// default (currentMapType already resolved that precedence): satellite
			// imagery + labels is the most useful base for tracing a parcel.
			map.setMapTypeId(currentMapType);

			// First open with a center and no saved camera → snap to the
			// pin at parcel zoom so drawing starts framed on the property.
			if (firstActivate && center && !mapView?.center) {
				map.setZoom(FIRST_ACTIVATE_ZOOM);
				map.panTo(center);
			}
			firstActivate = false;

			// Emit camera state on every idle (after pan / zoom / programmatic
			// fitBounds settles). Saved view reflects what the agent SEES, so
			// polygon edits that re-fit bounds also persist their camera.
			listeners.push(
				map.addListener('idle', () => {
					if (!mapInstance) return;
					const z = mapInstance.getZoom();
					const c = mapInstance.getCenter();
					onmapviewchange?.({
						zoom: z,
						center: [c.lng(), c.lat()]
					});
				})
			);

			if (polygon && polygon.coordinates?.[0]?.length >= 3) {
				loadExistingPolygon(ns, map, polygon);
			}
			// Start in drag mode regardless — the polygon, if any, loads
			// non-editable; the user opts into select to edit it. Readonly keeps the
			// map fully inert (no drag).
			if (!readonly) map.setOptions({ draggable: true });

			loading = false;
		} catch (e) {
			if (disposed) return;
			errorMessage =
				e instanceof MapConfigError
					? e.message
					: e instanceof Error
						? e.message
						: 'Map could not be loaded';
			loading = false;
		}
	}

	// Build the map when the gate is open AND the canvas container has rendered.
	// Covers BOTH at-mount (center already known) and ungate-later (pin dropped →
	// gateClosed flips false → {:else} mounts the container → this re-fires).
	// initStarted makes it run exactly once.
	$effect(() => {
		if (gateClosed || !mapContainer) return;
		void initMap();
	});

	// React to external `center` changes after the map is built (the Address
	// LocationPicker pin moved → the consumer updates `center`). Reposition the
	// anchor pin always; re-center the camera ONLY while no boundary exists —
	// once a parcel is drawn it owns the view, so we never yank the user off it.
	// Tracks `center` only (mapInstance / *Instance are plain refs, not state).
	$effect(() => {
		const c = center;
		if (!c) return;
		const map = mapInstance;
		if (!map) return;
		untrack(() => {
			if (locationPinMarker) locationPinMarker.setPosition(c);
			else buildLocationPin(c);
			if (!polygonInstance) map.panTo(c);
		});
	});

	onMount(() => {
		// MAP-05 — track fullscreen enter/exit (Esc also exits, firing this).
		// Sync the flag (drives the toggle icon/label) and repaint Google's tiles
		// at the new container size. PolygonPicker mounts Google directly (no
		// adapter handle), so the resize is inlined here.
		const onFullscreenChange = () => {
			isFullscreen = document.fullscreenElement === canvasEl;
			// The canvas box just changed — repaint Google's tiles at the new size,
			// preserving the center (a raw resize pins the top-left corner).
			if (mapsNs && mapInstance) {
				const c = mapInstance.getCenter();
				mapsNs.event.trigger(mapInstance, 'resize');
				if (c) mapInstance.setCenter({ lat: c.lat(), lng: c.lng() });
			}
		};
		document.addEventListener('fullscreenchange', onFullscreenChange);

		// Escape dismisses the edge context menu (separate from the draw-mode
		// Escape, which cancels a draft — both can be live, never at once).
		const onMenuEscape = (e: KeyboardEvent) => {
			if (e.key === 'Escape') closeEdgeMenu();
		};
		window.addEventListener('keydown', onMenuEscape);

		return () => {
			disposed = true;
			document.removeEventListener('fullscreenchange', onFullscreenChange);
			window.removeEventListener('keydown', onMenuEscape);
			removeDrawListeners();
			clearDraft();
			for (const l of listeners) l.remove();
			listeners = [];
			clearEdgeOverlays();
			// Tear down the transient legend-edge highlight line if one is live.
			highlightLine?.setMap(null);
			highlightLine = null;
			if (polygonInstance) {
				polygonInstance.setMap(null);
				polygonInstance = null;
			}
			if (locationPinMarker) {
				locationPinMarker.setMap(null);
				locationPinMarker = null;
			}
			if (mapsNs && mapInstance) {
				mapsNs.event.clearInstanceListeners(mapInstance);
			}
			mapInstance = null;
			mapsNs = null;
			EdgeLabelClass = null;
		};
	});

	// External `center` changes ARE handled (the effect above repositions the
	// pin + re-centers while no boundary exists). External `polygon` prop swaps
	// mid-mount are NOT — the polygon loads once on mount; re-mount via {#key}
	// from the consumer to swap shapes.
</script>

<div class="ui-map-polygon-picker" class:is-readonly={readonly} style:height>
	{#if gateClosed}
		<!-- Location gate — no center yet. Render the consumer's richer
		     placeholder snippet when supplied, else the default message. The
		     Google map is not mounted at all in this state. -->
		<div class="ui-map-polygon-picker__gate" role="note">
			{#if noLocation}
				{@render noLocation()}
			{:else}
				<span class="ui-map-polygon-picker__gate-icon" aria-hidden="true">
					<svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="1.6">
						<circle cx="12" cy="10" r="3" />
						<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z" />
					</svg>
				</span>
				<span class="ui-map-polygon-picker__gate-text">{noLocationMessage}</span>
			{/if}
		</div>
	{:else}
		<div
			bind:this={canvasEl}
			class="ui-map-polygon-picker__canvas"
			class:is-fullscreen={isFullscreen}
			class:is-loading={loading}
			class:is-error={!!errorMessage}
			role={readonly ? 'img' : undefined}
			aria-label={readonly ? 'Property plot boundary map, read-only' : undefined}
		>
			<div bind:this={mapContainer} class="ui-map-polygon-picker__map"></div>

			{#if loading && !errorMessage}
				<div class="ui-map-polygon-picker__overlay ui-map-polygon-picker__overlay--loading">
					Loading map…
				</div>
			{/if}

			{#if errorMessage}
				<div
					class="ui-map-polygon-picker__overlay ui-map-polygon-picker__overlay--error"
					role="alert"
				>
					<strong>Map unavailable</strong>
					<span>{errorMessage}</span>
				</div>
			{/if}

			{#if !loading && !errorMessage && !readonly}
				<!-- Floating mode toolbar (top-left): drag · select · draw. Real
				     buttons with aria-pressed; the active mode is highlighted.
				     draw is disabled while a polygon exists (clear it first);
				     select is disabled until a polygon exists. -->
				<div
					class="ui-map-polygon-picker__toolbar"
					role="group"
					aria-label="Boundary tools"
				>
					<button
						type="button"
						class="ui-map-polygon-picker__tool"
						class:is-active={mode === 'drag'}
						aria-pressed={mode === 'drag'}
						onclick={() => setMode('drag')}
						title="Pan the map"
						aria-label="Pan the map"
					>
						<!-- Lucide `hand` — an open grab hand reads as "pan/drag the map"
						     far more clearly than the cramped single-path hand it replaced. -->
						<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
							<path d="M18 11V6a2 2 0 0 0-2-2 2 2 0 0 0-2 2"/><path d="M14 10V4a2 2 0 0 0-2-2 2 2 0 0 0-2 2v2"/><path d="M10 10.5V6a2 2 0 0 0-2-2 2 2 0 0 0-2 2v8"/><path d="M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15"/>
						</svg>
					</button>
					<button
						type="button"
						class="ui-map-polygon-picker__tool"
						class:is-active={mode === 'select'}
						aria-pressed={mode === 'select'}
						disabled={!hasPolygon}
						onclick={() => setMode('select')}
						title="Select & edit vertices"
						aria-label="Select and edit vertices"
					>
						<!-- Lucide `mouse-pointer-2` — a cursor arrow is unambiguously
						     "select", unlike the prior glyph that read as a location pin. -->
						<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
							<path d="M4.037 4.688a.495.495 0 0 1 .651-.651l16 6.5a.5.5 0 0 1-.063.947l-6.124 1.58a2 2 0 0 0-1.438 1.435l-1.579 6.126a.5.5 0 0 1-.947.063z"/>
						</svg>
					</button>
					<button
						type="button"
						class="ui-map-polygon-picker__tool"
						class:is-active={mode === 'draw'}
						aria-pressed={mode === 'draw'}
						disabled={hasPolygon}
						onclick={() => setMode('draw')}
						title="Draw the boundary"
						aria-label="Draw the boundary"
					>
						<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
							<path d="M14.5 4.5l5 5L9 20l-5 1 1-5z" />
						</svg>
					</button>
				</div>

				<!-- Finish affordance — appears only while drawing, once a
				     closeable ring (>=3 points) exists. The first vertex also
				     closes the ring; this is the explicit alternative. -->
				{#if mode === 'draw' && draftCount >= 3}
					<button
						type="button"
						class="ui-map-polygon-picker__finish"
						onclick={finishDraw}
					>
						Finish boundary
					</button>
				{/if}

				<!-- Corner trash — clear the whole boundary. Select mode only,
				     and only when a polygon exists. Returns to the empty state. -->
				{#if mode === 'select' && hasPolygon}
					<button
						type="button"
						class="ui-map-polygon-picker__trash"
						onclick={clearPolygon}
						title="Clear the boundary"
						aria-label="Clear the boundary"
					>
						<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
							<path d="M4 7h16M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2m2 0v12a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1V7" />
						</svg>
					</button>
				{/if}

				<!-- Recenter — frame the map on the current boundary. Sits below
				     the trash so both corner controls stay clear of the toolbar. -->
				{#if hasPolygon}
					<button
						type="button"
						class="ui-map-polygon-picker__recenter"
						onclick={recenterOnPolygon}
						title="Recenter on the boundary"
						aria-label="Recenter on the boundary"
					>
						<svg viewBox="0 0 16 16" width="14" height="14" fill="currentColor" aria-hidden="true">
							<path d="M8 1.5a.5.5 0 0 1 .5.5v1.526A4.5 4.5 0 0 1 12.474 7.5H14a.5.5 0 0 1 0 1h-1.526A4.5 4.5 0 0 1 8.5 12.474V14a.5.5 0 0 1-1 0v-1.526A4.5 4.5 0 0 1 3.526 8.5H2a.5.5 0 0 1 0-1h1.526A4.5 4.5 0 0 1 7.5 3.526V2a.5.5 0 0 1 .5-.5zm0 3A3.5 3.5 0 1 0 8 11.5 3.5 3.5 0 0 0 8 4.5zm0 2a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z" />
						</svg>
					</button>
				{/if}

				<!-- MAP-05 — top-right control cluster: base map-type segmented
				     control + custom fullscreen toggle. Mirrors LocationPicker's
				     chrome so the two pickers read as one component. The cluster
				     owns the top-right positioning; the children stay inert layout
				     so they never collide with the boundary toolbar or Google's
				     zoom chrome. -->
				<div class="ui-map-polygon-picker__controls">
					<div
						class="ui-map-polygon-picker__maptype"
						role="group"
						aria-label="Map type"
					>
						{#each [['roadmap', 'Map'], ['satellite', 'Sat'], ['terrain', 'Terr'], ['hybrid', 'Hyb']] as [code, label] (code)}
							<button
								type="button"
								class="ui-map-polygon-picker__maptype-btn"
								class:is-active={currentMapType === code}
								aria-pressed={currentMapType === code}
								onclick={() => setMapType(code as 'roadmap' | 'satellite' | 'terrain' | 'hybrid')}
								title={label}
							>
								{label}
							</button>
						{/each}
					</div>
					<button
						type="button"
						class="ui-map-polygon-picker__fullscreen"
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

			<!-- Live plot legend — fullscreen-ONLY overlay. Mounted INSIDE the canvas
			     so it participates in custom fullscreen; guarded on isFullscreen so the
			     inline editor (with its footer sides·perimeter readout) is untouched.
			     PlotLegend self-hides below 3 vertices, but we also gate on
			     liveRing.length >= 3 so the wrapper never reserves space for an empty
			     panel. Never shown in readonly. The panel owns its own styling/tokens —
			     we only position the wrapper top-right, below the controls cluster. -->
			{#if !loading && !errorMessage && !readonly && isFullscreen && liveRing.length >= 3}
				<div class="ui-map-polygon-picker__legend">
					<PlotLegend
						ring={liveRing}
						bind:unit={legendUnit}
						highlightEdge={legendHighlight}
						onhoveredge={(i) => (legendHighlight = i)}
					/>
				</div>
			{/if}

			<!-- Edge context menu — appears at the cursor on right-click of an edge
			     (select mode). One action: splice a node onto that edge. Lives
			     INSIDE the canvas so it shows in fullscreen too; positioned in px
			     relative to the canvas box. Closes via the action, Escape, a click
			     elsewhere, or a camera move (wired on the map). -->
			{#if menuOpen && !readonly}
				<div
					class="ui-map-polygon-picker__ctxmenu"
					style:left="{menuX}px"
					style:top="{menuY}px"
					role="menu"
				>
					<button
						type="button"
						class="ui-map-polygon-picker__ctxmenu-item"
						role="menuitem"
						onclick={addNodeFromMenu}
					>
						Add node here
					</button>
				</div>
			{/if}
		</div>

		<!-- Footer row — boundary status. Lives outside the map canvas so it
		     never collides with Google's native controls or our floating
		     overlays. Trimmed to sides · perimeter: the area readout moved to
		     the consumer's land-size helper. -->
		{#if !loading && !errorMessage && !readonly}
			<div class="ui-map-polygon-picker__footer">
				<div class="ui-map-polygon-picker__status">
					{#if mode === 'draw'}
						<span class="ui-map-polygon-picker__hint">
							Click on the map to add points. Click the first point — or Finish — to close.
						</span>
					{:else if hasPolygon}
						<span class="ui-map-polygon-picker__status-row">
							<strong>{sideCount}</strong> sides
							<span class="ui-map-polygon-picker__status-sep">·</span>
							<strong>{formatLengthAuto(perimeterMeters)}</strong> perimeter
						</span>
					{:else}
						<span class="ui-map-polygon-picker__hint">No boundary set — pick the draw tool to start</span>
					{/if}
				</div>
			</div>
		{/if}
	{/if}
</div>

<style>
	.ui-map-polygon-picker {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		width: 100%;
	}
	.ui-map-polygon-picker__canvas {
		position: relative;
		width: 100%;
		flex: 1 1 auto;
		min-height: 240px;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.5rem;
		overflow: hidden;
		background: var(--color-surface-muted, #f4f4f5);
	}
	/* MAP-05 — fullscreen: fill the screen, overriding the flex height. The
	   `is-fullscreen` class (driven by the fullscreenchange listener) drives
	   sizing — more reliable cross-browser than the :fullscreen pseudo. Mirrors
	   LocationPicker's fullscreen canvas so the two pickers behave identically. */
	.ui-map-polygon-picker__canvas.is-fullscreen {
		width: 100%;
		height: 100% !important;
		border: 0;
		border-radius: 0;
	}
	/* Readonly (Land READ view) honors the consumer height (e.g. 200px) — drop the
	   interactive min-height so it matches the address read map size exactly. */
	.ui-map-polygon-picker.is-readonly .ui-map-polygon-picker__canvas {
		min-height: 0;
	}
	.ui-map-polygon-picker__map {
		position: absolute;
		inset: 0;
	}
	/* MAP-05 — this Google Maps build ignores both `fullscreenControl: false` AND
	   `disableDefaultUI: true` for the fullscreen control: the native "Toggle
	   fullscreen view" button persists. Its fullscreen covers only this bare map
	   div, dropping the boundary toolbar / map-type control / polygon overlays —
	   the exact bug MAP-05 fixes. Hide it so only our custom fullscreen (which
	   fullscreens the whole canvas) remains. `.gm-fullscreen-control` is Google's
	   stable control class; scoped to this map so other Google maps on the page
	   are untouched. */
	.ui-map-polygon-picker__map :global(.gm-fullscreen-control) {
		display: none !important;
	}
	.ui-map-polygon-picker__overlay {
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
	.ui-map-polygon-picker__overlay--loading {
		color: var(--color-muted, #71717a);
	}
	.ui-map-polygon-picker__overlay--error {
		color: var(--color-error, #b91c1c);
	}

	/* Location gate — placeholder shown when requireCenter is set and no
	   center exists yet. Styled like the map canvas so the card keeps its
	   shape, but with centred guidance instead of a map. */
	.ui-map-polygon-picker__gate {
		position: relative;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		width: 100%;
		flex: 1 1 auto;
		min-height: 240px;
		padding: 1.5rem;
		text-align: center;
		border: 1px dashed var(--color-border, #d4d4d8);
		border-radius: 0.5rem;
		background: var(--color-surface-muted, #f4f4f5);
		color: var(--color-muted, #71717a);
	}
	.ui-map-polygon-picker__gate-icon {
		color: var(--color-border-strong, #a1a1aa);
	}
	.ui-map-polygon-picker__gate-text {
		font-size: 0.875rem;
		line-height: 1.4;
		max-width: 28rem;
	}

	/* Floating boundary toolbar — top-left. A compact pill group that overlays
	   the map. Real buttons; the active mode is filled. */
	.ui-map-polygon-picker__toolbar {
		position: absolute;
		top: 0.625rem;
		left: 0.625rem;
		z-index: 5;
		display: inline-flex;
		gap: 0.125rem;
		padding: 0.1875rem;
		background: var(--color-surface, #fff);
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.5rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
	}
	.ui-map-polygon-picker__tool {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		padding: 0;
		border: 0;
		border-radius: 0.375rem;
		background: transparent;
		color: var(--color-text-muted, #4b5563);
		cursor: pointer;
		transition: background 120ms ease, color 120ms ease;
	}
	/* Bump the glyphs — at 16px the hand/arrow/pencil read too small in the
	   2rem button (live-test feedback). A CSS width beats the SVG's inline
	   presentation attribute, so this resizes without touching the markup. */
	.ui-map-polygon-picker__tool svg {
		width: 1.25rem;
		height: 1.25rem;
	}
	.ui-map-polygon-picker__tool:hover:not(:disabled) {
		background: var(--color-surface-muted, #f4f4f5);
		color: var(--color-text, #18181b);
	}
	.ui-map-polygon-picker__tool.is-active {
		background: var(--color-brand, #e5392b);
		color: #fff;
	}
	.ui-map-polygon-picker__tool:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	/* Finish affordance — centred near the top while drawing a closeable ring. */
	.ui-map-polygon-picker__finish {
		position: absolute;
		top: 0.625rem;
		left: 50%;
		transform: translateX(-50%);
		z-index: 5;
		padding: 0.4rem 0.85rem;
		border: 1px solid transparent;
		border-radius: 999px;
		background: var(--color-brand, #e5392b);
		color: #fff;
		font: inherit;
		font-weight: 600;
		font-size: 0.8125rem;
		cursor: pointer;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
	}
	.ui-map-polygon-picker__finish:hover {
		background: #c92e21;
	}

	/* Corner controls (trash + recenter) — stacked down the left edge below
	   the toolbar so they never crowd the top-right map-type control. */
	.ui-map-polygon-picker__trash,
	.ui-map-polygon-picker__recenter {
		position: absolute;
		left: 0.625rem;
		z-index: 5;
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
		transition: background 120ms ease, color 120ms ease;
	}
	.ui-map-polygon-picker__trash {
		top: 3.75rem;
		color: var(--color-error, #b91c1c);
	}
	.ui-map-polygon-picker__recenter {
		top: 6.25rem;
	}
	.ui-map-polygon-picker__trash:hover,
	.ui-map-polygon-picker__recenter:hover {
		background: var(--color-surface-muted, #f4f4f5);
	}

	/* MAP-05 — top-right control cluster: map-type switcher + fullscreen toggle.
	   The cluster owns the top-right positioning (mirrors LocationPicker's
	   __controls); its children are inert layout boxes. */
	.ui-map-polygon-picker__controls {
		position: absolute;
		top: 0.625rem;
		right: 0.625rem;
		z-index: 5;
		display: flex;
		align-items: flex-start;
		gap: 0.375rem;
	}
	/* Live plot legend wrapper — fullscreen-only. Anchored top-right BELOW the
	   controls cluster (top: 3.5rem clears the ~2rem map-type/fullscreen row + its
	   inset) so the panel never overlaps those buttons. PlotLegend caps its own
	   height to the viewport and carries its own surface/tokens — we only position. */
	.ui-map-polygon-picker__legend {
		position: absolute;
		top: 3.5rem;
		right: 0.625rem;
		z-index: 5;
	}
	/* Edge context menu — a single-item pill anchored at the cursor (px relative
	   to the canvas). z-index 6 keeps it above the map and the other floating
	   controls (z 5). Surface tokens with light fallbacks match the toolbar. */
	.ui-map-polygon-picker__ctxmenu {
		position: absolute;
		z-index: 6;
		min-width: 9rem;
		padding: 0.1875rem;
		background: var(--color-surface, #fff);
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.5rem;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.22);
	}
	.ui-map-polygon-picker__ctxmenu-item {
		display: block;
		width: 100%;
		padding: 0.4rem 0.6rem;
		border: 0;
		border-radius: 0.375rem;
		background: transparent;
		color: var(--color-text, #18181b);
		font: inherit;
		font-size: 0.8125rem;
		text-align: left;
		cursor: pointer;
		transition: background 120ms ease;
	}
	.ui-map-polygon-picker__ctxmenu-item:hover {
		background: var(--color-surface-muted, #f4f4f5);
	}
	/* Map-type segmented control — compact pill group; the active button has a
	   brand-tinted fill. */
	.ui-map-polygon-picker__maptype {
		display: inline-flex;
		border: 1px solid var(--color-border, #d4d4d8);
		border-radius: 0.375rem;
		overflow: hidden;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
	}
	.ui-map-polygon-picker__maptype-btn {
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
	.ui-map-polygon-picker__maptype-btn:last-child {
		border-right: 0;
	}
	.ui-map-polygon-picker__maptype-btn:hover {
		background: var(--color-surface-muted, #f4f4f5);
		color: var(--color-text, #18181b);
	}
	.ui-map-polygon-picker__maptype-btn.is-active {
		background: var(--color-accent, #18181b);
		color: var(--color-on-accent, #fff);
	}
	/* MAP-05 — custom fullscreen toggle: compact square icon button (replaces
	   Google's native button, which fullscreened only the bare map div). Mirrors
	   LocationPicker's __fullscreen for visual parity. */
	.ui-map-polygon-picker__fullscreen {
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
	.ui-map-polygon-picker__fullscreen:hover {
		background: var(--color-surface-muted, #f4f4f5);
	}
	.ui-map-polygon-picker__fullscreen svg {
		display: block;
		fill: none;
		stroke: currentColor;
		stroke-width: 2;
		stroke-linecap: round;
		stroke-linejoin: round;
	}

	/* Footer row directly under the map. Wraps on narrow viewports so the
	   status (left) and any action group (right) stack instead of clipping. */
	.ui-map-polygon-picker__footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
		padding: 0.25rem 0.125rem 0;
	}
	.ui-map-polygon-picker__status {
		flex: 1 1 auto;
		min-width: 0;
		color: var(--color-text, #111827);
		font-family: var(--font-mono, ui-monospace, monospace);
		font-size: 0.8125rem;
		line-height: 1.4;
	}
	.ui-map-polygon-picker__status-row {
		display: inline-flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.25rem;
	}
	.ui-map-polygon-picker__status-sep {
		color: rgba(0, 0, 0, 0.35);
		padding: 0 0.125rem;
	}
	.ui-map-polygon-picker__hint {
		color: var(--color-muted, #71717a);
		font-style: italic;
		font-family: inherit;
	}
	/* HTML overlays painted by EdgeLabel — Google's OverlayView gives us a
	   div in the overlayLayer pane positioned at the projection point. We
	   anchor the label at its top-left + translate so the centre of the pill
	   sits on the edge midpoint exactly. */
	:global(.ui-map-edge-label) {
		position: absolute;
		transform: translate(-50%, -50%);
		padding: 0.125rem 0.4rem;
		background: rgba(255, 255, 255, 0.94);
		color: #111827;
		font-size: 0.6875rem;
		font-weight: 500;
		font-family: var(--font-mono, ui-monospace, monospace);
		border-radius: 0.375rem;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
		pointer-events: none;
		white-space: nowrap;
		line-height: 1.2;
	}
</style>
