// STATIC MAP PATH BUILDER — proxy-safe, keyless, generalized from BR canvas/maps.ts
//
// Returns a RELATIVE proxy path: /api/map/static?…
// The server proxy (api-chi) appends the real Google API key and forwards to
// maps.googleapis.com.  This module MUST NEVER emit a key= param or reference
// googleapis in any form — a key in any client-built URL is a HIGH security
// defect.
//
// Ported and generalized from:
//   clients/bestie/projects/bestierealestate/apps/web-svelte/src/lib/canvas/maps.ts
// Reuses coords.ts: isNullIsland, isValidCoords (NEVER re-implement).

import { isNullIsland, isValidCoords } from './coords.js';

// ── Constants ─────────────────────────────────────────────────────────────────

/** Google Static Maps hard limit is 8192; guard with headroom for a degenerate
 *  many-vertex polygon — omit the path rather than emit a 400-bound request. */
const MAX_STATIC_MAP_URL = 7500;

/** ~0.11 m — Google's documented useful precision; keeps path URLs short. */
const COORD_DECIMALS = 6;

/** A drawable closed GeoJSON ring: ≥ 4 positions (triangle + repeated first). */
const MIN_RING_POSITIONS = 4;

/** Polygon stroke/fill alphas: solid stroke / ~20% translucent fill. */
const PATH_STROKE_ALPHA = 'ff';
const PATH_FILL_ALPHA = '33';

/** Default polygon stroke weight in px (spike-verified rendering weight). */
const DEFAULT_PATH_WEIGHT = 3;

/** Neutral dark gray — fallback when brand hex can't ride the URL (rgb()/names/empty). */
const NEUTRAL_MAP_COLOR = '0x333333';

/** Static Maps maptype allowlist. */
const STATIC_MAP_TYPES = ['roadmap', 'satellite', 'hybrid', 'terrain'] as const;
export type StaticMapType = (typeof STATIC_MAP_TYPES)[number];
const DEFAULT_MAP_TYPE: StaticMapType = 'roadmap';

const ZOOM_MIN = 0;
const ZOOM_MAX = 21;

// ── Re-exported legacy types for callers that used the old interface ──────────

/** @deprecated Use StaticMapDescriptor instead. */
export interface StaticMapMarker {
	lat: number;
	lng: number;
	label?: string;
	color?: string;
}

/** @deprecated Use StaticMapDescriptor instead. */
export interface StaticMapPathPoint {
	lat: number;
	lng: number;
}

/** @deprecated Use StaticMapDescriptor instead. */
export interface StaticMapOpts {
	center: { lat: number; lng: number };
	zoom: number;
	size: { width: number; height: number };
	mapType?: StaticMapType | string;
	markers?: StaticMapMarker[];
	path?: StaticMapPathPoint[];
}

// ── New descriptor interface ──────────────────────────────────────────────────

/** Marker descriptor: position + optional CSS hex color. Color is normalized
 *  via toStaticMapsColor (#hex → 0xRRGGBB; anything else → NEUTRAL). */
export interface StaticMapMarkerDescriptor {
	lat: number;
	lng: number;
	color?: string; // CSS #hex accepted; normalized internally to 0xRRGGBB
}

/** Polygon descriptor: ring in GeoJSON [lng, lat] order (matches the wire format
 *  of pkg/address and the PolygonPicker). The builder flips to lat,lng for the
 *  Static Maps API. */
export interface StaticMapPolygonDescriptor {
	/** GeoJSON outer ring — positions as [longitude, latitude]. */
	ring: Array<[number, number]>;
	strokeColor?: string; // CSS #hex; default NEUTRAL
	fillColor?: string; // CSS #hex; default strokeColor
	weight?: number; // px; default DEFAULT_PATH_WEIGHT
}

/** Generic descriptor accepted by buildStaticMapPath(). center+zoom are optional
 *  — when omitted AND a polygon is present, center/zoom are excluded from the
 *  URL so Static Maps auto-fits the camera to the path (spike-proven behavior).
 *  Returns '' when there is nothing drawable.
 *
 *  `markers[]` and `polygons[]` are the multi-entry versions; they APPEND repeated
 *  params via URLSearchParams.append (the api-chi proxy forwards repeated params).
 *  The singular `marker`/`polygon` fields remain for back-compat — they are handled
 *  identically: each entry is validated and appended independently. */
export interface StaticMapDescriptor {
	center?: { lat: number; lng: number };
	zoom?: number;
	size: { width: number; height: number };
	scale?: 1 | 2;
	mapType?: string;
	/** @deprecated Single-entry back-compat. Use `markers` for multi-pin. */
	marker?: StaticMapMarkerDescriptor;
	/** @deprecated Single-entry back-compat. Use `polygons` for multi-polygon. */
	polygon?: StaticMapPolygonDescriptor;
	/** Multi-pin markers — each entry is validated and appended as a repeated `markers=` param. */
	markers?: StaticMapMarkerDescriptor[];
	/** Multi-polygon paths — each entry is validated and appended as a repeated `path=` param. */
	polygons?: StaticMapPolygonDescriptor[];
}

// ── Color helpers ─────────────────────────────────────────────────────────────

/** Normalize a CSS color value to the 0xRRGGBB form Static Maps expects.
 *  '#0D4F4F' → '0x0D4F4F', '#abc' → '0xAABBCC' (3-digit expanded, uppercased).
 *  Anything not a #hex literal (rgb(), color names, empty) → NEUTRAL_MAP_COLOR.
 *  Exported for direct use and test coverage. */
export function toStaticMapsColor(value: string | undefined): string {
	const trimmed = value?.trim() ?? '';
	if (!trimmed.startsWith('#')) return NEUTRAL_MAP_COLOR;
	const hex = trimmed.slice(1);
	if (/^[0-9a-f]{6}$/i.test(hex)) return `0x${hex.toUpperCase()}`;
	if (/^[0-9a-f]{3}$/i.test(hex)) {
		const expanded = [...hex].map((c) => c + c).join('');
		return `0x${expanded.toUpperCase()}`;
	}
	return NEUTRAL_MAP_COLOR;
}

// ── Coordinate / zoom helpers ─────────────────────────────────────────────────

/** Round to COORD_DECIMALS — full float precision only bloats the URL. */
function roundCoord(value: number): number {
	const factor = 10 ** COORD_DECIMALS;
	return Math.round(value * factor) / factor;
}

/** The 'lat,lng' literal every staticmap geo param uses. */
function formatLatLng(lat: number, lng: number): string {
	return `${roundCoord(lat)},${roundCoord(lng)}`;
}

/** Saved picker zooms are fractional floats; API wants integer 0..21.
 *  Non-numbers → undefined (caller decides the default). */
function clampZoom(zoom: number | undefined): number | undefined {
	if (typeof zoom !== 'number' || !Number.isFinite(zoom)) return undefined;
	return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, Math.round(zoom)));
}

/** Allowlisted maptype or the roadmap default. Prevents Mapbox style IDs or
 *  arbitrary strings from leaking into a Google URL. */
function allowedMapType(value: string | undefined): StaticMapType {
	return (STATIC_MAP_TYPES as readonly string[]).includes(value ?? '')
		? (value as StaticMapType)
		: DEFAULT_MAP_TYPE;
}

// ── Polygon ring helpers ──────────────────────────────────────────────────────

/** Validate and flip a GeoJSON [lng, lat] ring into 'lat,lng' strings.
 *  Returns undefined when the ring is too short (< MIN_RING_POSITIONS) or any
 *  position is malformed — a partial boundary draws a WRONG shape, so the whole
 *  path is omitted rather than silently misrepresent the polygon. */
function ringToPoints(ring: Array<[number, number]>): string[] | undefined {
	if (ring.length < MIN_RING_POSITIONS) return undefined;
	const points: string[] = [];
	for (const position of ring) {
		const lng = position[0];
		const lat = position[1];
		if (!Number.isFinite(lng) || !Number.isFinite(lat)) return undefined;
		points.push(formatLatLng(lat, lng)); // [lng, lat] → lat,lng (flip here)
	}
	return points;
}

// ── Core builder ─────────────────────────────────────────────────────────────

/**
 * Build a KEYLESS relative proxy path for the server-side /api/map/static.
 *
 * Security invariants (enforced, never negotiable):
 * - Output starts with /api/map/static?
 * - Output contains NO key= parameter
 * - Output references NO googleapis.com, google.com, or gstatic host
 *
 * Auto-fit behavior: when center+zoom are omitted but a valid polygon is
 * present, center/zoom are excluded from the URL — Static Maps auto-fits the
 * camera to the path ring (spike-proven on the M-0 canvas spike).
 *
 * Returns '' (empty string) when nothing is drawable:
 * - No center+zoom AND no polygon AND no marker
 * - Marker has invalid/null-island coords
 * - Polygon ring has < MIN_RING_POSITIONS positions or malformed coords
 * - Composed URL would exceed MAX_STATIC_MAP_URL chars
 */
export function buildStaticMapPath(descriptor: StaticMapDescriptor): string {
	const { center, zoom, size, scale = 1, mapType, marker, polygon, markers, polygons } = descriptor;

	const params = new URLSearchParams();

	// Center + zoom (optional — omit for auto-fit when polygon present)
	const clampedZoom = clampZoom(zoom);
	if (center !== undefined && clampedZoom !== undefined) {
		// Guard center against null-island and invalid coords
		const centerCoords = { lat: center.lat, lng: center.lng };
		if (!isValidCoords(centerCoords) || isNullIsland(centerCoords)) {
			// Invalid center with explicit intent → nothing drawable
			return '';
		}
		params.set('center', formatLatLng(center.lat, center.lng));
		params.set('zoom', String(clampedZoom));
	}

	// Size
	const width = Math.max(1, Math.round(Number(size.width)));
	const height = Math.max(1, Math.round(Number(size.height)));
	params.set('size', `${width}x${height}`);

	// Scale
	params.set('scale', String(scale));

	// Map type
	params.set('maptype', allowedMapType(mapType));

	let hasDrawable = false;

	// Helper — encode a single marker descriptor and append (repeated param).
	const appendMarker = (m: StaticMapMarkerDescriptor): void => {
		const markerCoords = { lat: m.lat, lng: m.lng };
		if (!isValidCoords(markerCoords) || isNullIsland(markerCoords)) return;
		const color = toStaticMapsColor(m.color);
		const point = formatLatLng(m.lat, m.lng);
		params.append('markers', `color:${color}|${point}`);
		hasDrawable = true;
	};

	// Helper — encode a single polygon descriptor and append (repeated param).
	const appendPolygon = (p: StaticMapPolygonDescriptor): void => {
		const points = ringToPoints(p.ring);
		if (points === undefined) return;
		const strokeColor = toStaticMapsColor(p.strokeColor);
		const fillColor = toStaticMapsColor(p.fillColor ?? p.strokeColor);
		const weight = typeof p.weight === 'number' && p.weight > 0
			? Math.round(p.weight)
			: DEFAULT_PATH_WEIGHT;
		params.append(
			'path',
			`color:${strokeColor}${PATH_STROKE_ALPHA}|fillcolor:${fillColor}${PATH_FILL_ALPHA}|weight:${weight}|${points.join('|')}`
		);
		hasDrawable = true;
	};

	// Singular back-compat fields (the property-plot path and any caller using
	// the old single-entry interface — must not be broken).
	if (marker !== undefined) appendMarker(marker);
	if (polygon !== undefined) appendPolygon(polygon);

	// Multi-entry arrays (canvas map-source resolver uses these).
	if (markers !== undefined) {
		for (const m of markers) appendMarker(m);
	}
	if (polygons !== undefined) {
		for (const p of polygons) appendPolygon(p);
	}

	// If we have a center we can also draw it without marker/polygon
	if (center !== undefined && clampedZoom !== undefined) {
		hasDrawable = true;
	}

	if (!hasDrawable) return '';

	const path = `/api/map/static?${params.toString()}`;

	// URL length guard — a degenerate many-vertex polygon can blow past the
	// API hard limit; omit the entry rather than emit a 400-bound request.
	if (path.length > MAX_STATIC_MAP_URL) return '';

	return path;
}
