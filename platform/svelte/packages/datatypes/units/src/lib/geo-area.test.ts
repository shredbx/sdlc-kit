import { describe, expect, it } from 'vitest';
import {
	formatLengthAuto,
	formatLengthFeet,
	type GeoCoordinate,
	geojsonPolygonAreaSqm,
	haversineMeters,
	isClosed,
	polygonAreaSqm,
	polygonFromLatLng,
	polygonPerimeterMeters,
	ringFromLatLng
} from './geo-area.js';

// Helper: build a near-square polygon centered on a TH location whose
// expected area we can compute manually. At latitude 9.75° N:
//   1° lat ≈ 110,946 m  (= EARTH_RADIUS_M * π / 180)
//   1° lng ≈ 110,946 * cos(9.75°) ≈ 109,343 m
// So a 0.001° × 0.001° box ≈ 110.946 m × 109.343 m ≈ 12,128 sqm
const KOH_PHANGAN_LAT = 9.75;
const KOH_PHANGAN_LNG = 100.0;

describe('polygonAreaSqm', () => {
	it('returns 0 for too-few points', () => {
		expect(polygonAreaSqm([])).toBe(0);
		expect(polygonAreaSqm([[100, 9.75]])).toBe(0);
		expect(
			polygonAreaSqm([
				[100, 9.75],
				[100.001, 9.75]
			])
		).toBe(0);
	});

	it('computes a ~12,130 sqm square at Koh Phangan latitude', () => {
		// 0.001° box at lat 9.75 → ~12,128 sqm
		const ring: GeoCoordinate[] = [
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG + 0.001, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG + 0.001, KOH_PHANGAN_LAT + 0.001],
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT + 0.001]
		];
		const area = polygonAreaSqm(ring);
		expect(area).toBeGreaterThan(12_000);
		expect(area).toBeLessThan(12_300);
	});

	it('tolerates both closed and open rings (same area)', () => {
		const open: GeoCoordinate[] = [
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG + 0.001, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG + 0.001, KOH_PHANGAN_LAT + 0.001],
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT + 0.001]
		];
		const closed: GeoCoordinate[] = [...open, open[0]];
		expect(polygonAreaSqm(closed)).toBeCloseTo(polygonAreaSqm(open), 6);
	});

	it('produces the same area regardless of vertex winding', () => {
		// shoelace returns a signed value — we take absolute, so CW and
		// CCW should yield the same magnitude.
		const ccw: GeoCoordinate[] = [
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG + 0.001, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG + 0.001, KOH_PHANGAN_LAT + 0.001],
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT + 0.001]
		];
		const cw = [...ccw].reverse();
		expect(polygonAreaSqm(cw)).toBeCloseTo(polygonAreaSqm(ccw), 6);
	});

	it('computes a triangle of known area', () => {
		// Right triangle: 0.001° × 0.001° → area ≈ 12,128 / 2 ≈ 6,064 sqm
		const ring: GeoCoordinate[] = [
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG + 0.001, KOH_PHANGAN_LAT],
			[KOH_PHANGAN_LNG, KOH_PHANGAN_LAT + 0.001]
		];
		const area = polygonAreaSqm(ring);
		expect(area).toBeGreaterThan(6_000);
		expect(area).toBeLessThan(6_200);
	});
});

describe('isClosed', () => {
	it('detects closed rings', () => {
		expect(
			isClosed([
				[0, 0],
				[1, 0],
				[1, 1],
				[0, 0]
			])
		).toBe(true);
	});
	it('detects open rings', () => {
		expect(
			isClosed([
				[0, 0],
				[1, 0],
				[1, 1]
			])
		).toBe(false);
	});
	it('returns false for under-2-point input', () => {
		expect(isClosed([])).toBe(false);
		expect(isClosed([[0, 0]])).toBe(false);
	});
});

describe('ringFromLatLng', () => {
	it('swaps lat/lng → [lng, lat] for GeoJSON', () => {
		const out = ringFromLatLng([
			{ lat: 9.75, lng: 100.0 },
			{ lat: 9.76, lng: 100.01 }
		]);
		expect(out).toEqual([
			[100.0, 9.75],
			[100.01, 9.76]
		]);
	});
});

describe('polygonFromLatLng', () => {
	it('returns GeoJSON Polygon with closed ring', () => {
		const poly = polygonFromLatLng([
			{ lat: 9.75, lng: 100.0 },
			{ lat: 9.75, lng: 100.001 },
			{ lat: 9.751, lng: 100.001 }
		]);
		expect(poly.type).toBe('Polygon');
		expect(poly.coordinates.length).toBe(1);
		const ring = poly.coordinates[0];
		expect(ring[0]).toEqual(ring[ring.length - 1]);
		expect(ring.length).toBe(4); // 3 input + 1 closing copy
	});
	it('preserves already-closed ring without adding a second copy', () => {
		const poly = polygonFromLatLng([
			{ lat: 9.75, lng: 100.0 },
			{ lat: 9.75, lng: 100.001 },
			{ lat: 9.751, lng: 100.001 },
			{ lat: 9.75, lng: 100.0 }
		]);
		expect(poly.coordinates[0].length).toBe(4);
	});
});

describe('haversineMeters', () => {
	it('returns 0 for identical points', () => {
		expect(haversineMeters([100.0, 9.75], [100.0, 9.75])).toBe(0);
	});

	it('computes ~111 km for 1° of latitude at the equator', () => {
		const d = haversineMeters([0, 0], [0, 1]);
		// 1° meridian ≈ 111,195 m via WGS-84 mean radius
		expect(d).toBeGreaterThan(111_000);
		expect(d).toBeLessThan(111_400);
	});

	it('computes ~110 km for 1° of longitude at 9.75°N', () => {
		const d = haversineMeters([0, 9.75], [1, 9.75]);
		// 1° lng at 9.75°N ≈ 109,600–109,800 m depending on radius
		// constant used. We use WGS-84 equatorial (6 378 137 m) so the
		// value sits at the upper end of that range.
		expect(d).toBeGreaterThan(109_400);
		expect(d).toBeLessThan(109_900);
	});

	it('is symmetric', () => {
		const a: GeoCoordinate = [100.0, 9.75];
		const b: GeoCoordinate = [100.05, 9.80];
		expect(haversineMeters(a, b)).toBeCloseTo(haversineMeters(b, a), 9);
	});
});

describe('polygonPerimeterMeters', () => {
	it('returns 0 for under-2-point input', () => {
		expect(polygonPerimeterMeters([])).toBe(0);
		expect(polygonPerimeterMeters([[100, 9.75]])).toBe(0);
	});

	it('sums edges of a closed ring', () => {
		// Square at 9.75°N, side ≈ 0.001° ≈ 110 m
		const closed: GeoCoordinate[] = [
			[100.0, 9.75],
			[100.001, 9.75],
			[100.001, 9.751],
			[100.0, 9.751],
			[100.0, 9.75]
		];
		const p = polygonPerimeterMeters(closed);
		// 4 sides × ~110 m = ~440 m
		expect(p).toBeGreaterThan(430);
		expect(p).toBeLessThan(450);
	});

	it('tolerates open ring (closes implicitly)', () => {
		const open: GeoCoordinate[] = [
			[100.0, 9.75],
			[100.001, 9.75],
			[100.001, 9.751],
			[100.0, 9.751]
		];
		const closed: GeoCoordinate[] = [...open, open[0]];
		expect(polygonPerimeterMeters(open)).toBeCloseTo(polygonPerimeterMeters(closed), 6);
	});
});

describe('formatLengthAuto', () => {
	it('formats sub-1000 m as meters', () => {
		expect(formatLengthAuto(120)).toBe('120 m');
		expect(formatLengthAuto(999.4)).toBe('999 m');
	});
	it('formats >=1000 m as km with 1-2 decimals', () => {
		expect(formatLengthAuto(1000)).toBe('1.0 km');
		expect(formatLengthAuto(1234)).toBe('1.23 km');
		expect(formatLengthAuto(12345)).toBe('12.35 km');
	});
	it('returns empty for invalid input', () => {
		expect(formatLengthAuto(Number.NaN)).toBe('');
		expect(formatLengthAuto(-5)).toBe('');
	});
});

describe('geojsonPolygonAreaSqm', () => {
	it('returns 0 for null / undefined / wrong type', () => {
		expect(geojsonPolygonAreaSqm(null)).toBe(0);
		expect(geojsonPolygonAreaSqm(undefined)).toBe(0);
		expect(geojsonPolygonAreaSqm({ type: 'Polygon', coordinates: [] })).toBe(0);
	});
	it('delegates to polygonAreaSqm on the outer ring', () => {
		const poly = polygonFromLatLng([
			{ lat: KOH_PHANGAN_LAT, lng: KOH_PHANGAN_LNG },
			{ lat: KOH_PHANGAN_LAT, lng: KOH_PHANGAN_LNG + 0.001 },
			{ lat: KOH_PHANGAN_LAT + 0.001, lng: KOH_PHANGAN_LNG + 0.001 },
			{ lat: KOH_PHANGAN_LAT + 0.001, lng: KOH_PHANGAN_LNG }
		]);
		expect(geojsonPolygonAreaSqm(poly)).toBeGreaterThan(12_000);
		expect(geojsonPolygonAreaSqm(poly)).toBeLessThan(12_300);
	});
});

describe('formatLengthFeet (imperial edge lengths)', () => {
	it('formats sub-mile distances as whole feet', () => {
		expect(formatLengthFeet(30.48)).toBe('100 ft'); // 30.48 m = 100 ft exactly
	});
	it('formats 0 m as 0 ft', () => {
		expect(formatLengthFeet(0)).toBe('0 ft');
	});
	it('formats >= 1 mile as miles with 1-2 decimals', () => {
		expect(formatLengthFeet(2000)).toBe('1.24 mi'); // 2000 m ≈ 1.24 mi (clear of the 1-mile boundary)
	});
	it('returns empty for invalid input', () => {
		expect(formatLengthFeet(Number.NaN)).toBe('');
		expect(formatLengthFeet(-5)).toBe('');
	});
});
