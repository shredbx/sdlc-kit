import type { LngLat } from './types.js';

const LAT_MIN = -90;
const LAT_MAX = 90;
const LNG_MIN = -180;
const LNG_MAX = 180;

export function clampLat(lat: number): number {
	if (!Number.isFinite(lat)) return 0;
	return Math.min(LAT_MAX, Math.max(LAT_MIN, lat));
}

export function normalizeLng(lng: number): number {
	if (!Number.isFinite(lng)) return 0;
	let n = lng;
	while (n > LNG_MAX) n -= 360;
	while (n < LNG_MIN) n += 360;
	return n;
}

export function clampCoords(loc: LngLat): LngLat {
	return {
		lat: clampLat(loc.lat),
		lng: normalizeLng(loc.lng)
	};
}

export function isValidCoords(loc: { lat: unknown; lng: unknown } | null | undefined): boolean {
	if (!loc) return false;
	const { lat, lng } = loc as LngLat;
	return (
		typeof lat === 'number' &&
		typeof lng === 'number' &&
		Number.isFinite(lat) &&
		Number.isFinite(lng) &&
		lat >= LAT_MIN &&
		lat <= LAT_MAX &&
		lng >= LNG_MIN &&
		lng <= LNG_MAX
	);
}

// Legacy never-set sentinel. Rows from before the picker shipped commonly
// have (0, 0) which would render in the Gulf of Guinea — treat as unset.
export function isNullIsland(loc: LngLat): boolean {
	return loc.lat === 0 && loc.lng === 0;
}

export function coordsEqual(a: LngLat, b: LngLat, epsilon = 1e-6): boolean {
	return Math.abs(a.lat - b.lat) < epsilon && Math.abs(a.lng - b.lng) < epsilon;
}
