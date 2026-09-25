// Geodesic area calculation for GeoJSON polygons.
//
// Algorithm: latitude-corrected shoelace formula on an equirectangular
// projection centered on the polygon's centroid. Accurate to ~0.1% at
// typical Thai latitudes (5°–20° N) for parcels under ~1 km².
//
// Why not use @turf/area: avoids a 70 KB dependency for what is one
// well-known formula. If accuracy ever needs to be sub-0.01% for huge
// polygons spanning kilometers of latitude, swap in turf at the call
// site — the public surface (polygonAreaSqm) stays unchanged.

// GeoJSON ordering: [longitude, latitude]. This package follows the
// spec; UI-layer LngLat-objects swap at the boundary.
export type GeoCoordinate = [number, number]; // [lng, lat]

const EARTH_RADIUS_METERS = 6_378_137; // WGS-84 equatorial radius
const DEG_TO_RAD = Math.PI / 180;
const METERS_PER_DEG_LAT = (Math.PI * EARTH_RADIUS_METERS) / 180; // ≈ 111320 m

// Polygon must be a CLOSED LinearRing per GeoJSON: first and last
// coordinates equal. Callers may pass either form — we tolerate the
// open variant and treat it as closed.
export function polygonAreaSqm(ring: readonly GeoCoordinate[]): number {
	if (!ring || ring.length < 3) return 0;

	// Drop the duplicate closing vertex if present so the shoelace loop
	// doesn't double-count.
	const open = isClosed(ring) ? ring.slice(0, -1) : ring.slice();
	if (open.length < 3) return 0;

	// Reference latitude for equirectangular projection: arithmetic mean
	// of the polygon's vertex latitudes. Good enough for parcels up to
	// ~10 km north-south extent.
	let latSum = 0;
	for (const [, lat] of open) latSum += lat;
	const refLat = latSum / open.length;
	const metersPerDegLng = METERS_PER_DEG_LAT * Math.cos(refLat * DEG_TO_RAD);

	// Shoelace on projected meters.
	let area2 = 0;
	for (let i = 0; i < open.length; i++) {
		const [lng1, lat1] = open[i];
		const [lng2, lat2] = open[(i + 1) % open.length];
		const x1 = lng1 * metersPerDegLng;
		const y1 = lat1 * METERS_PER_DEG_LAT;
		const x2 = lng2 * metersPerDegLng;
		const y2 = lat2 * METERS_PER_DEG_LAT;
		area2 += x1 * y2 - x2 * y1;
	}
	return Math.abs(area2) / 2;
}

export function isClosed(ring: readonly GeoCoordinate[]): boolean {
	if (ring.length < 2) return false;
	const first = ring[0];
	const last = ring[ring.length - 1];
	return first[0] === last[0] && first[1] === last[1];
}

// Convert a ring of LngLat objects (the UI-layer convention used by
// @sbx/ui-map) into GeoJSON-ordered tuples for area math.
export function ringFromLatLng(
	points: ReadonlyArray<{ lat: number; lng: number }>
): GeoCoordinate[] {
	return points.map((p): GeoCoordinate => [p.lng, p.lat]);
}

// GeoJSON Polygon: outer ring + optional holes. We don't support holes
// in v1 — most real-estate parcels are simply-connected. Holes are
// silently ignored to stay forward-compatible.
export interface GeoJSONPolygon {
	type: 'Polygon';
	coordinates: GeoCoordinate[][];
}

export function geojsonPolygonAreaSqm(polygon: GeoJSONPolygon | null | undefined): number {
	if (!polygon || polygon.type !== 'Polygon' || polygon.coordinates.length === 0) return 0;
	return polygonAreaSqm(polygon.coordinates[0]);
}

// Great-circle (haversine) distance in meters between two GeoJSON points.
// Accurate to within a few meters at any latitude — much higher precision
// than the equirectangular shoelace, which is fine for parcels but loses
// accuracy for long edges. We use haversine for edge lengths so labels on
// long parcel sides stay correct.
export function haversineMeters(a: GeoCoordinate, b: GeoCoordinate): number {
	const [lng1, lat1] = a;
	const [lng2, lat2] = b;
	const phi1 = (lat1 * Math.PI) / 180;
	const phi2 = (lat2 * Math.PI) / 180;
	const dphi = ((lat2 - lat1) * Math.PI) / 180;
	const dlambda = ((lng2 - lng1) * Math.PI) / 180;
	const s =
		Math.sin(dphi / 2) ** 2 + Math.cos(phi1) * Math.cos(phi2) * Math.sin(dlambda / 2) ** 2;
	return 2 * EARTH_RADIUS_METERS * Math.asin(Math.min(1, Math.sqrt(s)));
}

// Total perimeter of a polygon ring in meters. Accepts open or closed
// rings; if open, the closing edge (last → first) is included.
export function polygonPerimeterMeters(ring: readonly GeoCoordinate[]): number {
	if (!ring || ring.length < 2) return 0;
	const closed = isClosed(ring) ? ring.slice(0, -1) : ring.slice();
	if (closed.length < 2) return 0;
	let total = 0;
	for (let i = 0; i < closed.length; i++) {
		total += haversineMeters(closed[i], closed[(i + 1) % closed.length]);
	}
	return total;
}

// Format a meter distance for display:
//   < 1000 m → "320 m"
//   ≥ 1000 m → "1.2 km"
// Matches Google Maps default presentation.
export function formatLengthAuto(meters: number): string {
	if (!Number.isFinite(meters) || meters < 0) return '';
	if (meters < 1000) {
		return `${Math.round(meters).toLocaleString()} m`;
	}
	return `${(meters / 1000).toLocaleString(undefined, {
		minimumFractionDigits: 1,
		maximumFractionDigits: 2
	})} km`;
}

const FEET_PER_METER = 3.280839895; // 1 m = 1/0.3048 ft (international foot)
const FEET_PER_MILE = 5280;

// Imperial sibling of formatLengthAuto:
//   < 1 mile → "328 ft"
//   ≥ 1 mile → "1.24 mi"
// Used for parcel edge lengths when the legend's unit system is imperial.
// Negative / non-finite → ''.
export function formatLengthFeet(meters: number): string {
	if (!Number.isFinite(meters) || meters < 0) return '';
	const feet = meters * FEET_PER_METER;
	if (feet < FEET_PER_MILE) {
		return `${Math.round(feet).toLocaleString()} ft`;
	}
	return `${(feet / FEET_PER_MILE).toLocaleString(undefined, {
		minimumFractionDigits: 1,
		maximumFractionDigits: 2
	})} mi`;
}

export function polygonFromLatLng(
	points: ReadonlyArray<{ lat: number; lng: number }>
): GeoJSONPolygon {
	const ring = ringFromLatLng(points);
	// Ensure closed ring per GeoJSON spec.
	if (ring.length > 0 && !isClosed(ring)) {
		ring.push([ring[0][0], ring[0][1]]);
	}
	return { type: 'Polygon', coordinates: [ring] };
}
