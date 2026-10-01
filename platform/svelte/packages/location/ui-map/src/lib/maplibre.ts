// MapLibre viewer config + GeoJSON helpers (PURE, unit-testable).
//
// The MapLibre map itself is constructed inside the maplibre-viewer adapter — it touches
// WebGL/DOM and can't run under vitest — so everything that CAN be pure lives
// here: the basemap-style resolver, ring→GeoJSON conversion, the highlighted-edge
// LineString, and the directions deep-link. The canvas export slice (#0303) reuses
// these same builders.
//
// This is the PUBLIC-viewer counterpart to adapters/google.ts (the admin authoring
// adapter). A read-only viewer needs neither geocoding nor the picker `MapAdapter`
// contract (mount/search/reverseGeocode/ondragend), so it is intentionally NOT a
// MapAdapter — forcing those methods would be dead ceremony. Admin (Google) and
// public (MapLibre) are different components for different jobs, not one component
// swapping backends. Decision #0303.
import type { GeoCoordinate } from '@sbx/units';
import { sanitizeRing } from './parcel.js';
import type { LngLat } from './types.js';

// Dev basemap: OpenFreeMap public endpoint — no key, no quota (#0303 §2). 'liberty'
// is the rich OUTDOOR style (green landcover for forest/parks, roads, place labels) —
// chosen over the flat grey 'positron' so the viewer reads as terrain (green +, with
// the DEM hillshade below, mountains), not a bare street map. License-clean (OSM/ODbL).
// PROD must swap to a vendored, self-hosted Protomaps style (tracked in #0303 §1–2) so
// we never depend on a remote style we don't control; until that exists, dev and prod
// both use this endpoint.
export const OPENFREEMAP_STYLE = 'https://tiles.openfreemap.org/styles/liberty';

/** Basemap style URL for the public parcel viewer (see {@link OPENFREEMAP_STYLE}). */
export function resolveViewerStyle(): string {
	return OPENFREEMAP_STYLE;
}

// Brand wash for the LISTINGS BOARD basemap (task 2607-133). The liberty style's saturated
// sky-blue water overpowers a page whose palette is warm cream + deep teal — on an island
// catalog the map is MOSTLY water, so the default blue becomes the loudest color on the page.
// These two desaturated teal-greys sit the basemap back into the brand so the white/gold price
// pills are the loudest thing on the map instead. Applied post-load by paint override (never a
// forked style JSON — the style stays the vendor's; we adjust two paints).
export const BRAND_WATER_FILL = '#C8D9D5';
export const BRAND_WATERWAY_LINE = '#B7CCC7';

/** Minimal structural view of a MapLibre map — lets this stay pure-typed (and unit-testable
 *  with a stub) without importing the GL types here. */
export interface PaintableMap {
	getStyle(): { layers?: Array<{ id: string; type: string }> } | undefined;
	setPaintProperty(layerId: string, name: string, value: unknown): void;
}

/** Recolor the basemap's water into the brand wash. Defensive by design: the vendor style's
 *  layer ids are not our contract, so every override is best-effort and a miss is silent — the
 *  board still works on the stock palette. Water FILLS get the wash; waterway LINES (rivers)
 *  get the darker line tone; water LABELS are left alone (they must stay legible). */
export function applyBrandWaterWash(map: PaintableMap): void {
	let layers: Array<{ id: string; type: string }> = [];
	try {
		layers = map.getStyle()?.layers ?? [];
	} catch {
		return;
	}
	for (const layer of layers) {
		const id = layer.id.toLowerCase();
		if (id.includes('water_name') || id.includes('label')) continue;
		try {
			if (layer.type === 'fill' && id.includes('water')) {
				map.setPaintProperty(layer.id, 'fill-color', BRAND_WATER_FILL);
			} else if (layer.type === 'line' && id.includes('waterway')) {
				map.setPaintProperty(layer.id, 'line-color', BRAND_WATERWAY_LINE);
			}
		} catch {
			// Best-effort per layer — a renamed vendor layer must never break the board.
		}
	}
}

// Free, keyless elevation tiles (AWS Open Data terrarium-encoded DEM) — feeds the
// MapLibre 3D terrain mesh + the hillshade relief layer so the viewer shows mountain
// terrain (#0303, "terrain/outdoor" basemap). No key, CORS-enabled, ODbL/attribution.
// A DEM failure is non-fatal: it's added AFTER load, so the basemap + plot still render.
export const TERRAIN_DEM_TILES = 'https://s3.amazonaws.com/elevation-tiles-prod/terrarium/{z}/{x}/{y}.png';
export const TERRAIN_DEM_ENCODING = 'terrarium';

// Minimal GeoJSON shapes — @types/geojson is only a transitive dep of maplibre-gl
// (not resolvable from this package's source under pnpm's strict layout), so we
// declare the exact shapes we emit. Structurally compatible with the GeoJSON the
// MapLibre source API consumes.
export interface PolygonGeometry {
	type: 'Polygon';
	coordinates: [number, number][][];
}
export interface LineStringGeometry {
	type: 'LineString';
	coordinates: [number, number][];
}
export interface GeoFeature<G> {
	type: 'Feature';
	properties: Record<string, never>;
	geometry: G;
}
export interface EmptyFeatureCollection {
	type: 'FeatureCollection';
	features: never[];
}

/** Empty source payload — clears the highlighted-edge layer. */
export const EMPTY_FEATURE_COLLECTION: EmptyFeatureCollection = {
	type: 'FeatureCollection',
	features: []
};

// Convert a raw boundary ring into a closed GeoJSON Polygon feature for the plot
// fill + outline layers. Sanitizes first (drops invalid/duplicate-closing vertices,
// caps the count). A GeoJSON linear ring MUST repeat its first position last, so we
// re-append it here. Returns null for a ring that can't form a polygon (< 3 nodes).
export function ringToPolygonFeature(
	ring: readonly GeoCoordinate[]
): GeoFeature<PolygonGeometry> | null {
	const open = sanitizeRing(ring);
	if (open.length < 3) return null;
	const closed: [number, number][] = [...open, open[0]].map(([lng, lat]) => [lng, lat]);
	return { type: 'Feature', properties: {}, geometry: { type: 'Polygon', coordinates: [closed] } };
}

// Build a 2-point LineString for ONE polygon side, by the same edge index the legend
// uses (edge i = node i → node i+1, wrapping on the last). Drives the hover-highlight
// layer. Returns null when the index is out of range or the ring is too small — the
// caller clears the layer with EMPTY_FEATURE_COLLECTION.
export function edgeToLineFeature(
	ring: readonly GeoCoordinate[],
	edgeIndex: number | null
): GeoFeature<LineStringGeometry> | null {
	const open = sanitizeRing(ring);
	const n = open.length;
	if (edgeIndex == null || edgeIndex < 0 || edgeIndex >= n || n < 2) return null;
	const a = open[edgeIndex];
	const b = open[(edgeIndex + 1) % n];
	return {
		type: 'Feature',
		properties: {},
		geometry: {
			type: 'LineString',
			coordinates: [
				[a[0], a[1]],
				[b[0], b[1]]
			]
		}
	};
}

// A LineString PER polygon side, each carrying its edge index, for the invisible wide
// hover hit-target layer: hovering the OUTLINE (not just a vertex marker) highlights
// that edge + its legend row. Edge i = node i → node i+1, wrapping on the last — the
// SAME indexing as edgeToLineFeature + the legend. Empty collection when < 2 nodes.
export interface EdgeHitFeature {
	type: 'Feature';
	properties: { edge: number };
	geometry: LineStringGeometry;
}
export interface EdgeHitCollection {
	type: 'FeatureCollection';
	features: EdgeHitFeature[];
}
export function edgesToHitFeatures(ring: readonly GeoCoordinate[]): EdgeHitCollection {
	const open = sanitizeRing(ring);
	const n = open.length;
	if (n < 2) return { type: 'FeatureCollection', features: [] };
	const features: EdgeHitFeature[] = [];
	for (let i = 0; i < n; i++) {
		const a = open[i];
		const b = open[(i + 1) % n];
		features.push({
			type: 'Feature',
			properties: { edge: i },
			geometry: { type: 'LineString', coordinates: [[a[0], a[1]], [b[0], b[1]]] }
		});
	}
	return { type: 'FeatureCollection', features };
}

// Google Maps directions deep-link to the property pin — the one sanctioned reason a
// public visitor leaves the platform (turn-by-turn navigation), kept as a secondary
// affordance (#0303 D6). Uses the documented, key-free Maps URL scheme.
export function directionsUrl(center: LngLat): string {
	return `https://www.google.com/maps/dir/?api=1&destination=${center.lat},${center.lng}`;
}
