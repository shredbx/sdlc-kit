// Parcel geometry + measurement-legend logic (pure). Extracted from the Svelte
// MapViewer/PlotLegend so the math is unit-testable and never lives inside a
// template. All geo math delegates to @sbx/units (one source of truth for
// haversine / shoelace / unit conversion).
import {
	type GeoCoordinate,
	type LandSizeUnit,
	formatArea,
	formatLengthAuto,
	formatLengthFeet,
	haversineMeters,
	isClosed,
	polygonAreaSqm,
	polygonPerimeterMeters
} from '@sbx/units';
import type { BoundingBox } from './types.js';

// The legend's unit toggle is the workspace land-size unit set (Decision #0303 /
// design D5) — sqm · rai · ngan · wah · sqft, the SAME LandSizeUnit the property
// SizeInput uses. We deliberately don't invent "metric/thai/imperial" categories;
// the area renders in the picked unit via formatArea. Re-exported so the legend
// and viewer take their toggle vocabulary from one source.
export type { LandSizeUnit } from '@sbx/units';

// Defensive cap mirroring the server-side vertex cap (#0303 hardening): a
// hostile/huge ring can DoS the WebGL renderer, so the client never measures or
// renders more than this many vertices either.
export const MAX_PARCEL_VERTICES = 2000;

const LAT_MIN = -90;
const LAT_MAX = 90;
const LNG_MIN = -180;
const LNG_MAX = 180;

function isValidCoord(c: GeoCoordinate): boolean {
	if (!Array.isArray(c) || c.length < 2) return false;
	const [lng, lat] = c;
	return (
		Number.isFinite(lng) &&
		Number.isFinite(lat) &&
		lat >= LAT_MIN &&
		lat <= LAT_MAX &&
		lng >= LNG_MIN &&
		lng <= LNG_MAX
	);
}

// Clean a raw boundary ring: keep only finite, in-range [lng,lat] pairs; drop a
// closing duplicate vertex (we work with the OPEN ring for labels/edges); cap
// the vertex count. Returns an open ring (first !== last).
export function sanitizeRing(ring: readonly GeoCoordinate[] | null | undefined): GeoCoordinate[] {
	if (!ring || ring.length === 0) return [];
	const valid = ring.filter(isValidCoord).map((c): GeoCoordinate => [c[0], c[1]]);
	const open = valid.length > 1 && isClosed(valid) ? valid.slice(0, -1) : valid;
	return open.length > MAX_PARCEL_VERTICES ? open.slice(0, MAX_PARCEL_VERTICES) : open;
}

// Vertex identifiers: letters A.. while the plot has ≤ 26 nodes (cadastral
// norm); numbers 1.. once it exceeds 26 (letters stop reading clearly).
export function nodeLabels(count: number): string[] {
	if (count <= 0) return [];
	if (count <= 26) {
		return Array.from({ length: count }, (_, i) => String.fromCharCode(65 + i));
	}
	return Array.from({ length: count }, (_, i) => String(i + 1));
}

// Total area in the chosen land-size unit — '8,000 m²' · '5 rai' · '2,130 wah'
// · '86,111 sqft'. Delegates to @sbx/units formatArea (one source of truth).
export function formatParcelArea(sqm: number, unit: LandSizeUnit): string {
	return formatArea(sqm, { unit });
}

// Edge/perimeter LENGTH (linear). Land-size units are areal, so length follows
// the unit's measurement family: sqft → feet/miles, every metric-rooted unit
// (sqm/rai/ngan/wah) → metres/km. Surveyors quote Thai plots in metres, so the
// metric branch is correct for rai/ngan/wah.
export function formatParcelLength(meters: number, unit: LandSizeUnit): string {
	return unit === 'sqft' ? formatLengthFeet(meters) : formatLengthAuto(meters);
}

export interface ParcelEdge {
	/** Vertex-pair name, e.g. "A–B" (en dash). */
	name: string;
	/** Great-circle length in meters. */
	meters: number;
	/** Length formatted in the active unit system. */
	lengthLabel: string;
}

// One edge per polygon side, including the closing side (last → first).
export function parcelEdges(ring: readonly GeoCoordinate[], unit: LandSizeUnit): ParcelEdge[] {
	const open = sanitizeRing(ring);
	const n = open.length;
	if (n < 2) return [];
	const labels = nodeLabels(n);
	const edges: ParcelEdge[] = [];
	for (let i = 0; i < n; i++) {
		const next = (i + 1) % n;
		const meters = haversineMeters(open[i], open[next]);
		edges.push({
			name: `${labels[i]}–${labels[next]}`,
			meters,
			lengthLabel: formatParcelLength(meters, unit)
		});
	}
	return edges;
}

export interface ParcelStats {
	nodeCount: number;
	labels: string[];
	areaSqm: number;
	areaLabel: string;
	perimeterMeters: number;
	perimeterLabel: string;
	edges: ParcelEdge[];
}

// Everything the legend renders, derived from a raw ring + the active unit.
// Recompute on a unit switch (cheap) — keeps the toggle reactive.
export function parcelStats(ring: readonly GeoCoordinate[], unit: LandSizeUnit): ParcelStats {
	const open = sanitizeRing(ring);
	const areaSqm = polygonAreaSqm(open);
	const perimeterMeters = polygonPerimeterMeters(open);
	return {
		nodeCount: open.length,
		labels: nodeLabels(open.length),
		areaSqm,
		areaLabel: formatParcelArea(areaSqm, unit),
		perimeterMeters,
		perimeterLabel: formatParcelLength(perimeterMeters, unit),
		edges: parcelEdges(open, unit)
	};
}

export interface Point2D {
	x: number;
	y: number;
}

// Project a parcel ring into a square `size`×`size` SVG box for a thumbnail
// glyph: equirectangular (longitude scaled by cos(meanLat)) so the shape isn't
// distorted, aspect preserved, centered, y flipped so north is up. A degenerate
// (zero-span) ring collapses to the box center. Empty ring → [].
export function normalizedRingPoints(
	ring: readonly GeoCoordinate[],
	size: number,
	pad = 0
): Point2D[] {
	const open = sanitizeRing(ring);
	if (open.length === 0) return [];
	const inner = Math.max(0, size - 2 * pad);

	let latSum = 0;
	for (const [, lat] of open) latSum += lat;
	const kx = Math.cos(((latSum / open.length) * Math.PI) / 180);

	const proj = open.map(([lng, lat]) => ({ px: lng * kx, py: lat }));
	let minX = Infinity;
	let maxX = -Infinity;
	let minY = Infinity;
	let maxY = -Infinity;
	for (const { px, py } of proj) {
		if (px < minX) minX = px;
		if (px > maxX) maxX = px;
		if (py < minY) minY = py;
		if (py > maxY) maxY = py;
	}
	const spanX = maxX - minX;
	const spanY = maxY - minY;
	const span = Math.max(spanX, spanY);
	if (span <= 0) return proj.map(() => ({ x: size / 2, y: size / 2 }));

	const scale = inner / span;
	const offX = pad + (inner - spanX * scale) / 2;
	const offY = pad + (inner - spanY * scale) / 2;
	return proj.map(({ px, py }) => ({
		x: offX + (px - minX) * scale,
		y: offY + (maxY - py) * scale // flip: north up
	}));
}

// South/west/north/east extent for fitBounds(). Null when the ring is empty.
export function ringBounds(ring: readonly GeoCoordinate[]): BoundingBox | null {
	const open = sanitizeRing(ring);
	if (open.length === 0) return null;
	let south = Infinity;
	let west = Infinity;
	let north = -Infinity;
	let east = -Infinity;
	for (const [lng, lat] of open) {
		if (lat < south) south = lat;
		if (lat > north) north = lat;
		if (lng < west) west = lng;
		if (lng > east) east = lng;
	}
	return { south, west, north, east };
}
