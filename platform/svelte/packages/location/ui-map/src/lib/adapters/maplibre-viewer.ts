// MapLibre viewer adapter (#0303 / #0304) — the PUBLIC map for the fullscreen
// MapViewer: free OpenFreeMap 'liberty' terrain/outdoor basemap + a free DEM hillshade,
// the red pin, the blue plot (+ white casing), lettered vertex handles, and the
// highlighted-edge layer. MapLibre GL is DYNAMICALLY imported here so a page ships none
// of the WebGL engine until the viewer opens. Pure geometry/config lives in
// maplibre.ts + parcel.ts (unit-tested); this is the DOM/GL glue, so it isn't unit-
// tested (no WebGL under vitest). Implements the provider-agnostic MapViewerAdapter.
import type { GeoCoordinate } from '@sbx/units';
import { nodeLabels, ringBounds, sanitizeRing } from '../parcel.js';
import {
	EMPTY_FEATURE_COLLECTION,
	TERRAIN_DEM_ENCODING,
	TERRAIN_DEM_TILES,
	edgeToLineFeature,
	edgesToHitFeatures,
	resolveViewerStyle,
	ringToPolygonFeature
} from '../maplibre.js';
import type { MapViewerAdapter, MapViewerMountOpts } from '../types.js';
import { areaMarkerElement, locationPinElement } from '../markers.js';

type MaplibreModule = typeof import('maplibre-gl');
type MaplibreMap = import('maplibre-gl').Map;
type MaplibreMarker = import('maplibre-gl').Marker;
type MaplibreGeoJSONSource = import('maplibre-gl').GeoJSONSource;

const PLOT_BLUE = '#2563EB';

// Small lettered vertex handle (A·B·C…), cross-referencing the legend rows. Inline-styled.
function makeNodeElement(label: string): HTMLDivElement {
	const el = document.createElement('div');
	el.textContent = label;
	el.style.cssText =
		'display:flex;align-items:center;justify-content:center;width:1.15rem;height:1.15rem;' +
		`font-family:ui-monospace,'SF Mono',Menlo,monospace;font-size:0.62rem;font-weight:600;` +
		`color:${PLOT_BLUE};background:#fff;border:1.5px solid ${PLOT_BLUE};border-radius:50%;` +
		'box-shadow:0 1px 2px rgba(0,0,0,0.3);cursor:pointer;';
	return el;
}

export function createMaplibreViewerAdapter(): MapViewerAdapter {
	let map: MaplibreMap | null = null;
	let nodeMarkers: MaplibreMarker[] = [];
	let pinMarker: MaplibreMarker | null = null;
	let ring: GeoCoordinate[] = [];
	let highlight: number | null = null;
	let loaded = false;
	let generation = 0;

	function applyHighlight(): void {
		const src = map?.getSource('edge-hl') as MaplibreGeoJSONSource | undefined;
		if (!src) return;
		src.setData(edgeToLineFeature(ring, highlight) ?? EMPTY_FEATURE_COLLECTION);
	}

	function teardown(): void {
		generation++;
		for (const mk of nodeMarkers) mk.remove();
		nodeMarkers = [];
		pinMarker?.remove();
		pinMarker = null;
		map?.remove();
		map = null;
		loaded = false;
	}

	// Free DEM hillshade for mountain relief (no setTerrain — its 3D mesh blanks the GL
	// canvas until ready; hillshade degrades safely). 'liberty' supplies the green.
	function setupTerrain(m: MaplibreMap): void {
		try {
			if (!m.getSource('terrain-dem')) {
				m.addSource('terrain-dem', {
					type: 'raster-dem',
					tiles: [TERRAIN_DEM_TILES],
					encoding: TERRAIN_DEM_ENCODING,
					tileSize: 256,
					maxzoom: 14
				});
			}
			const firstSymbol = m.getStyle().layers?.find((l) => l.type === 'symbol')?.id;
			if (!m.getLayer('hillshade')) {
				m.addLayer(
					{
						id: 'hillshade',
						type: 'hillshade',
						source: 'terrain-dem',
						paint: {
							'hillshade-exaggeration': 0.28,
							'hillshade-shadow-color': 'rgba(74, 78, 71, 0.55)',
							'hillshade-highlight-color': 'rgba(255, 255, 255, 0.45)',
							'hillshade-accent-color': 'rgba(120, 130, 110, 0.3)'
						}
					},
					firstSymbol
				);
			}
		} catch {
			// Relief is optional — keep the flat basemap + plot on any DEM/style hiccup.
		}
	}

	function drawPlot(ml: MaplibreModule, m: MaplibreMap, center: GeoCoordinate, onNodeHover?: (i: number | null) => void, approximate?: boolean, areaLabel?: string): void {
		const poly = ringToPolygonFeature(ring);
		if (poly) {
			m.addSource('plot', { type: 'geojson', data: poly });
			m.addLayer({
				id: 'plot-fill',
				type: 'fill',
				source: 'plot',
				paint: { 'fill-color': PLOT_BLUE, 'fill-opacity': 0.2 }
			});
			// White casing under the blue outline so the edge stays legible over terrain.
			m.addLayer({
				id: 'plot-casing',
				type: 'line',
				source: 'plot',
				layout: { 'line-join': 'round', 'line-cap': 'round' },
				paint: { 'line-color': '#ffffff', 'line-width': 5.5, 'line-opacity': 0.9 }
			});
			m.addLayer({
				id: 'plot-line',
				type: 'line',
				source: 'plot',
				layout: { 'line-join': 'round' },
				paint: { 'line-color': PLOT_BLUE, 'line-width': 3 }
			});
			// Invisible WIDE hit-line per edge — hovering the OUTLINE (not only a vertex
			// marker) highlights that edge + its legend row. opacity 0 still receives
			// layer events; the generous width is a forgiving target over the thin line.
			// Edge index matches the legend (edge i = node i → node i+1).
			m.addSource('edges-hit', { type: 'geojson', data: edgesToHitFeatures(ring) });
			m.addLayer({
				id: 'edges-hit',
				type: 'line',
				source: 'edges-hit',
				layout: { 'line-cap': 'round', 'line-join': 'round' },
				paint: { 'line-color': '#000000', 'line-width': 18, 'line-opacity': 0 }
			});
			m.on('mousemove', 'edges-hit', (e) => {
				const props = e.features?.[0]?.properties as { edge?: number } | null | undefined;
				const edge = props?.edge;
				if (typeof edge === 'number') {
					onNodeHover?.(edge);
					m.getCanvas().style.cursor = 'pointer';
				}
			});
			m.on('mouseleave', 'edges-hit', () => {
				onNodeHover?.(null);
				m.getCanvas().style.cursor = '';
			});
		}
		m.addSource('edge-hl', { type: 'geojson', data: EMPTY_FEATURE_COLLECTION });
		m.addLayer({
			id: 'edge-hl',
			type: 'line',
			source: 'edge-hl',
			layout: { 'line-cap': 'round', 'line-join': 'round' },
			paint: { 'line-color': PLOT_BLUE, 'line-width': 6, 'line-opacity': 0.85 }
		});

		pinMarker = new ml.Marker({
			element: approximate ? areaMarkerElement(areaLabel) : locationPinElement(),
			anchor: approximate ? 'center' : 'bottom'
		})
			.setLngLat([center[0], center[1]])
			.addTo(m);

		const nodes = sanitizeRing(ring);
		const labels = nodeLabels(nodes.length);
		nodeMarkers = nodes.map((c, i) => {
			const el = makeNodeElement(labels[i] ?? '');
			el.addEventListener('pointerenter', () => onNodeHover?.(i));
			el.addEventListener('pointerleave', () => onNodeHover?.(null));
			return new ml.Marker({ element: el, anchor: 'center' }).setLngLat([c[0], c[1]]).addTo(m);
		});
	}

	return {
		mapTypes: [],
		defaultMapType: 'terrain',
		setMapType() {
			// Public terrain provider has a single base map — nothing to switch.
		},
		setHighlightEdge(edgeIndex: number | null) {
			highlight = edgeIndex;
			applyHighlight();
		},
		async mount(opts: MapViewerMountOpts) {
			const myGen = ++generation;
			ring = opts.ring;
			let ns: MaplibreModule;
			try {
				ns = await import('maplibre-gl');
				// CSS loads with the engine (controls + attribution styling) — both lazy,
				// so the shell stays provider-agnostic and admin pages bundle neither.
				await import('maplibre-gl/dist/maplibre-gl.css');
			} catch {
				opts.onError?.();
				return;
			}
			if (myGen !== generation) return; // destroyed mid-load
			const ml = ('default' in ns ? ns.default : ns) as MaplibreModule;

			const m = new ml.Map({
				container: opts.container,
				style: resolveViewerStyle(),
				center: [opts.center.lng, opts.center.lat],
				// Area frames pass their level zoom (province/district/sub-district); a plot,
				// when present, wins via fitBounds() in the load handler below.
				zoom: opts.zoom ?? 15,
				attributionControl: { compact: true }
			});
			map = m;
			// Zoom +/− only. NO compass/pitch button (showCompass:false) — this is a
			// READ-ONLY viewer; the bottom-left compass-with-pitch square reads as a
			// terrain/3D toggle and isn't needed for a look-only map (the user can still
			// pan/zoom/tilt by gesture). Keep zoom as the one explicit affordance.
			m.addControl(new ml.NavigationControl({ showCompass: false }), 'bottom-left');

			// Errors BEFORE first load = fatal (style/network) → onError. Tile/DEM errors
			// after load are non-fatal and ignored.
			m.on('error', () => {
				if (!loaded) {
					opts.onError?.();
					teardown();
				}
			});

			m.on('load', () => {
				if (myGen !== generation) return;
				loaded = true;
				setupTerrain(m);
				drawPlot(ml, m, [opts.center.lng, opts.center.lat], opts.onNodeHover, opts.approximate, opts.areaLabel);
				const b = ringBounds(ring);
				if (b) {
					m.fitBounds(
						[
							[b.west, b.south],
							[b.east, b.north]
						],
						{ padding: 64, duration: 0, maxZoom: 19 }
					);
				}
				applyHighlight();
			});
		},
		destroy() {
			teardown();
		}
	};
}
