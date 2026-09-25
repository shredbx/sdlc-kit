// Google viewer adapter (#0304) — the ADMIN read-only fullscreen map: a navigable
// (pan/zoom/tilt, NOT editable) Google map with the roadmap/satellite/terrain/hybrid
// switcher, plus the same red pin + blue plot + lettered vertex handles + highlighted
// edge the public MapLibre viewer draws. The admin-domain sibling of maplibre-viewer.ts
// — identical pin/plot/highlight semantics, only the DRAWING differs (Google overlays
// vs MapLibre GL layers), per the "admin = Google" rule (Decision #0304). Google Maps JS
// is DYNAMICALLY loaded via the shared loader, so admin pages ship none of the SDK until
// the viewer opens. Pure geometry lives in parcel.ts (unit-tested); this is the SDK/DOM
// glue, so it isn't unit-tested (no Google SDK under vitest) — exactly like the public
// sibling. Implements the provider-agnostic MapViewerAdapter.
import type { GeoCoordinate } from '@sbx/units';
import { nodeLabels, ringBounds, sanitizeRing } from '../parcel.js';
import { loadGoogleMaps } from '../google-loader.js';
import type { LngLat, MapViewerAdapter, MapViewerMountOpts } from '../types.js';
import { AREA_TONE_RGB, locationPinDataUri } from '../markers.js';

// Plot blue (#2563EB) — the designer-tool plot color unified across the admin picker,
// the public static preview, and both MapViewer adapters (#0304). Matches
// PolygonPicker's POLYGON_STYLE so the admin preview reads identically to the editor.
const PLOT_BLUE = '#2563EB';
// Approximate-AREA marker tone (neutral slate) — the ONE area tone shared via markers.ts.
const MARKER_APPROX = `rgb(${AREA_TONE_RGB})`;

// Plot fill/stroke, mirroring PolygonPicker's POLYGON_STYLE. strokeWeight is bumped to 3
// for the read-only viewer (no editable handles thicken the outline, so the line carries
// the plot's legibility — same weight as the MapLibre sibling's plot-line).
const POLYGON_STYLE = {
	strokeColor: PLOT_BLUE,
	strokeWeight: 3,
	fillColor: PLOT_BLUE,
	fillOpacity: 0.18
};

// Minimal Google Maps typings — we only touch the surface this viewer needs. A narrower
// view than google.ts/PolygonPicker declare (no geocoder, no overlays, no editing), but
// the same shapes for the parts we share (Marker icon/label, Polygon, Polyline).
interface GMap {
	setMapTypeId(mapTypeId: string): void;
	// Google accepts a LatLngBoundsLiteral ({north,south,east,west}) directly plus an
	// optional pixel padding — we pass the literal so we never build a LatLngBounds.
	fitBounds(
		bounds: { north: number; south: number; east: number; west: number },
		padding?: number
	): void;
	setCenter(loc: { lat: number; lng: number }): void;
	setZoom(zoom: number): void;
}
interface GMarker {
	setMap(map: GMap | null): void;
	addListener(event: string, handler: () => void): { remove(): void };
}
interface GPolygon {
	setMap(map: GMap | null): void;
}
interface GPolyline {
	setMap(map: GMap | null): void;
}
// Symbol icon for the lettered vertex handles — a white-filled blue-stroked circle, the
// Google equivalent of the MapLibre sibling's makeNodeElement() handle.
interface GPoint {
	x: number;
	y: number;
}
interface GMarkerSymbol {
	path: number;
	scale: number;
	fillColor: string;
	fillOpacity: number;
	strokeColor: string;
	strokeWeight: number;
	labelOrigin?: GPoint;
}
// Custom image icon (a data: URI) with a pixel anchor — how the canonical red pin renders on
// Google, its TIP on the coordinate (anchor = width/2, height).
interface GMarkerIcon {
	url: string;
	anchor?: GPoint;
	labelOrigin?: GPoint;
}
interface GMarkerLabel {
	text: string;
	color?: string;
	fontSize?: string;
	fontWeight?: string;
}
interface GMarkerCtor {
	new (opts: {
		position: { lat: number; lng: number };
		map: GMap;
		clickable?: boolean;
		cursor?: string;
		zIndex?: number;
		icon?: GMarkerSymbol | GMarkerIcon;
		label?: GMarkerLabel;
	}): GMarker;
}
interface GPolygonCtor {
	new (opts: {
		paths: Array<{ lat: number; lng: number }>;
		strokeColor?: string;
		strokeWeight?: number;
		fillColor?: string;
		fillOpacity?: number;
		clickable?: boolean;
		zIndex?: number;
	}): GPolygon;
}
interface GPolylineCtor {
	new (opts: {
		path: Array<{ lat: number; lng: number }>;
		strokeColor?: string;
		strokeWeight?: number;
		strokeOpacity?: number;
		clickable?: boolean;
		zIndex?: number;
	}): GPolyline;
}
interface GMapsNamespace {
	Map: new (
		container: HTMLElement,
		opts: {
			center: { lat: number; lng: number };
			zoom: number;
			disableDefaultUI?: boolean;
			clickableIcons?: boolean;
			zoomControl?: boolean;
			zoomControlOptions?: { position: number };
			fullscreenControl?: boolean;
			mapTypeControl?: boolean;
			gestureHandling?: string;
			mapTypeId?: string;
		}
	) => GMap;
	Marker: GMarkerCtor;
	Polygon: GPolygonCtor;
	Polyline: GPolylineCtor;
	// google.maps.SymbolPath.CIRCLE — the predefined vector path for the vertex handles.
	SymbolPath: { CIRCLE: number };
	// google.maps.Point — a pixel offset, used to anchor the custom pin icon at its tip.
	Point: new (x: number, y: number) => GPoint;
	// Positional subset — only the corner we pin the zoom control to.
	ControlPosition: { LEFT_BOTTOM: number };
	event: { clearInstanceListeners(instance: unknown): void };
}

// Convert a [lng,lat] ring coordinate to Google's {lat,lng} literal.
function toLatLng(c: GeoCoordinate): { lat: number; lng: number } {
	return { lat: c[1], lng: c[0] };
}

export function createGoogleViewerAdapter(): MapViewerAdapter {
	let ns: GMapsNamespace | null = null;
	let map: GMap | null = null;
	let polygon: GPolygon | null = null;
	let pinMarker: GMarker | null = null;
	let nodeMarkers: GMarker[] = [];
	let listeners: Array<{ remove(): void }> = [];
	let highlightLine: GPolyline | null = null;
	let ring: GeoCoordinate[] = [];
	let nodes: GeoCoordinate[] = [];
	let highlight: number | null = null;
	let generation = 0;

	// Draw (or clear) the highlighted edge: remove the old line, then — when an edge is
	// hovered and the ring has at least two nodes — stroke a fat blue line over that side
	// (node[i] → node[(i+1)%n], wrapping the closing edge). Mirrors the MapLibre sibling's
	// edge-hl layer (same color/width/opacity).
	function applyHighlight(): void {
		highlightLine?.setMap(null);
		highlightLine = null;
		if (!map || !ns || highlight === null || nodes.length < 2) return;
		const a = nodes[highlight];
		const b = nodes[(highlight + 1) % nodes.length];
		if (!a || !b) return;
		highlightLine = new ns.Polyline({
			path: [toLatLng(a), toLatLng(b)],
			strokeColor: PLOT_BLUE,
			strokeWeight: 6,
			strokeOpacity: 0.85,
			clickable: false,
			zIndex: 7
		});
		highlightLine.setMap(map);
	}

	// Detach every overlay + listener and null all refs. Bumping generation aborts any
	// in-flight mount() that resumes after this. Idempotent — every step guards nulls, so
	// it's safe to call from destroy() AND from an internal load abort. Mirrors the
	// MapLibre sibling's teardown() shape.
	function teardown(): void {
		generation++;
		polygon?.setMap(null);
		polygon = null;
		pinMarker?.setMap(null);
		pinMarker = null;
		for (const mk of nodeMarkers) mk.setMap(null);
		nodeMarkers = [];
		highlightLine?.setMap(null);
		highlightLine = null;
		for (const l of listeners) l.remove();
		listeners = [];
		if (ns && map) ns.event.clearInstanceListeners(map);
		map = null;
		ns = null;
		nodes = [];
		ring = [];
	}

	return {
		// Google's four base maps — the shell renders the switcher UI from this list.
		mapTypes: ['roadmap', 'satellite', 'terrain', 'hybrid'],
		// Hybrid (satellite + labels) is the authoring default — shows the parcel against
		// the real ground while keeping road/place labels for orientation.
		defaultMapType: 'hybrid',
		setMapType(type: string) {
			map?.setMapTypeId(type);
		},
		setHighlightEdge(edgeIndex: number | null) {
			highlight = edgeIndex;
			applyHighlight();
		},
		async mount(opts: MapViewerMountOpts) {
			const myGen = ++generation;
			ring = opts.ring;
			// Google needs an API key — without one we can't even request the SDK, so this
			// is a FATAL load failure → the shell shows its static fallback (never blank).
			if (!opts.apiKey) {
				opts.onError?.();
				return;
			}
			try {
				// The shared loader returns an opaque namespace; narrow it to our typed view.
				ns = (await loadGoogleMaps(opts.apiKey)) as GMapsNamespace;
			} catch {
				opts.onError?.();
				return;
			}
			if (myGen !== generation) return; // destroyed mid-load
			const g = ns;

			const center: LngLat = { lat: opts.center.lat, lng: opts.center.lng };
			const m = new g.Map(opts.container, {
				// Kill ALL of Google's default chrome, then add back only the zoom stack —
				// the MapViewer SHELL owns the base-map switcher, header, and close.
				disableDefaultUI: true,
				clickableIcons: false,
				zoomControl: true,
				// Zoom bottom-left to match the public viewer's NavigationControl corner.
				zoomControlOptions: { position: g.ControlPosition.LEFT_BOTTOM },
				fullscreenControl: false,
				mapTypeControl: false,
				// Full pan/zoom/tilt — this is a navigable (look-only) viewer, not editable.
				gestureHandling: 'greedy',
				mapTypeId: 'hybrid',
				center,
				zoom: 15
			});
			map = m;

			nodes = sanitizeRing(ring);
			const labels = nodeLabels(nodes.length);

			// PLOT — only a real polygon (≥3 vertices) gets a fill; a pin-only viewer skips it.
			if (nodes.length >= 3) {
				polygon = new g.Polygon({
					paths: nodes.map(toLatLng),
					...POLYGON_STYLE,
					clickable: false,
					zIndex: 4
				});
				polygon.setMap(m);
			}

			// LOCATION PIN — Google's DEFAULT marker is the red teardrop (≈ #EA4335),
			// matching the public viewer's red pin exactly. No icon override + no white
			// casing needed: the default already carries its own drop shadow + outline.
			// PROPERTY pin — the ONE canonical red teardrop + white circle (shared markers.ts),
			// its TIP on the coordinate (anchor width/2, height). APPROXIMATE → the dashed slate
			// area halo captioned with the area name (task 2607-115).
			const pinIcon: GMarkerSymbol | GMarkerIcon = opts.approximate
				? {
						path: g.SymbolPath.CIRCLE,
						scale: 16,
						fillColor: MARKER_APPROX,
						fillOpacity: 0.16,
						strokeColor: MARKER_APPROX,
						strokeWeight: 2
					}
				: { url: locationPinDataUri(), anchor: new g.Point(14, 38) };
			const areaLabel: GMarkerLabel | undefined =
				opts.approximate && opts.areaLabel
					? { text: opts.areaLabel, color: MARKER_APPROX, fontSize: '11px', fontWeight: '600' }
					: undefined;
			pinMarker = new g.Marker({
				position: center,
				map: m,
				clickable: false,
				zIndex: 5,
				icon: pinIcon,
				label: areaLabel
			});

			// LETTERED VERTEX HANDLES — one white-filled blue-stroked circle per node,
			// labelled A·B·C… cross-referencing the legend rows. Hover wires the inverse
			// highlight (node → legend row) via onNodeHover, same as the MapLibre sibling.
			const symbol: GMarkerSymbol = {
				path: g.SymbolPath.CIRCLE,
				scale: 9,
				fillColor: '#ffffff',
				fillOpacity: 1,
				strokeColor: PLOT_BLUE,
				strokeWeight: 2
			};
			nodeMarkers = nodes.map((c, i) => {
				const mk = new g.Marker({
					position: toLatLng(c),
					map: m,
					clickable: true,
					cursor: 'pointer',
					zIndex: 6,
					icon: symbol,
					label: {
						text: labels[i] ?? '',
						color: PLOT_BLUE,
						fontSize: '11px',
						fontWeight: '600'
					}
				});
				// Keep the listener handles so teardown() can detach them.
				listeners.push(mk.addListener('mouseover', () => opts.onNodeHover?.(i)));
				listeners.push(mk.addListener('mouseout', () => opts.onNodeHover?.(null)));
				return mk;
			});

			// FRAME — fit the camera to the parcel when there's a plot; otherwise tighten
			// onto the lone pin. 64px padding matches the MapLibre sibling's fitBounds.
			if (nodes.length >= 3) {
				const b = ringBounds(ring);
				if (b) {
					m.fitBounds({ north: b.north, south: b.south, east: b.east, west: b.west }, 64);
				}
			} else {
				m.setCenter(center);
				// Area frames pass their level zoom (province/district/sub-district); default to
				// the location zoom for an exact pin-only viewer.
				m.setZoom(opts.zoom ?? 16);
			}

			applyHighlight();
		},
		destroy() {
			teardown();
		}
	};
}
