import { describe, expect, it } from 'vitest';
import type { GeoCoordinate } from '@sbx/units';
import {
	MAX_PARCEL_VERTICES,
	formatParcelArea,
	formatParcelLength,
	nodeLabels,
	normalizedRingPoints,
	parcelStats,
	ringBounds,
	sanitizeRing
} from './parcel.js';

// A ~0.001° square at Koh Phangan latitude — ~12,128 sqm, ~440 m perimeter
// (mirrors the @sbx/units geo-area fixtures so the numbers are cross-checked).
const SQUARE: GeoCoordinate[] = [
	[100.0, 9.75],
	[100.001, 9.75],
	[100.001, 9.751],
	[100.0, 9.751]
];

describe('nodeLabels', () => {
	it('uses letters A.. for <= 26 nodes', () => {
		expect(nodeLabels(4)).toEqual(['A', 'B', 'C', 'D']);
		expect(nodeLabels(26)[25]).toBe('Z');
		expect(nodeLabels(26)).toHaveLength(26);
	});
	it('switches the whole plot to numbers when > 26 nodes', () => {
		const labels = nodeLabels(27);
		expect(labels).toHaveLength(27);
		expect(labels[0]).toBe('1');
		expect(labels[26]).toBe('27');
	});
	it('returns empty for zero nodes', () => {
		expect(nodeLabels(0)).toEqual([]);
	});
});

describe('sanitizeRing', () => {
	it('drops a closing duplicate vertex', () => {
		expect(sanitizeRing([...SQUARE, SQUARE[0]])).toEqual(SQUARE);
	});
	it('filters non-finite coordinates', () => {
		const dirty: GeoCoordinate[] = [...SQUARE, [Number.NaN, 9.75]];
		expect(sanitizeRing(dirty)).toEqual(SQUARE);
	});
	it('filters out-of-range lat/lng', () => {
		const dirty: GeoCoordinate[] = [...SQUARE, [100.0, 200], [400, 9.75]];
		expect(sanitizeRing(dirty)).toEqual(SQUARE);
	});
	it('caps the vertex count to MAX_PARCEL_VERTICES', () => {
		const huge: GeoCoordinate[] = Array.from(
			{ length: MAX_PARCEL_VERTICES + 50 },
			(_, i): GeoCoordinate => [100.0 + i * 0.00001, 9.75]
		);
		expect(sanitizeRing(huge)).toHaveLength(MAX_PARCEL_VERTICES);
	});
});

describe('ringBounds', () => {
	it('returns the south/west/north/east extent', () => {
		expect(ringBounds(SQUARE)).toEqual({
			south: 9.75,
			west: 100.0,
			north: 9.751,
			east: 100.001
		});
	});
	it('returns null for an empty ring', () => {
		expect(ringBounds([])).toBeNull();
	});
});

describe('formatParcelArea', () => {
	it('renders the m² label for the sqm unit', () => {
		expect(formatParcelArea(8000, 'sqm')).toBe('8,000 m²');
	});
	it('renders rai', () => {
		expect(formatParcelArea(1600, 'rai')).toBe('1 rai');
	});
	it('renders wah² from sqm (8520 → 2,130 wah)', () => {
		expect(formatParcelArea(8520, 'wah')).toBe('2,130 wah');
	});
	it('renders sqft for the imperial unit', () => {
		expect(formatParcelArea(8000, 'sqft')).toContain('sqft');
	});
});

describe('formatParcelLength', () => {
	it('uses metres for every metric-rooted unit (sqm/rai/ngan/wah)', () => {
		expect(formatParcelLength(120, 'sqm')).toBe('120 m');
		expect(formatParcelLength(120, 'rai')).toBe('120 m');
		expect(formatParcelLength(120, 'wah')).toBe('120 m');
	});
	it('uses feet for sqft', () => {
		expect(formatParcelLength(30.48, 'sqft')).toBe('100 ft');
	});
});

describe('normalizedRingPoints', () => {
	it('maps each vertex into a north-up box, aspect-corrected and finite', () => {
		const pts = normalizedRingPoints(SQUARE, 40, 4);
		expect(pts).toHaveLength(4);
		for (const p of pts) {
			expect(Number.isFinite(p.x)).toBe(true);
			expect(Number.isFinite(p.y)).toBe(true);
			expect(p.x).toBeGreaterThanOrEqual(0);
			expect(p.x).toBeLessThanOrEqual(40);
			expect(p.y).toBeGreaterThanOrEqual(0);
			expect(p.y).toBeLessThanOrEqual(40);
		}
	});
	it('centers a degenerate (zero-span) ring at the box center', () => {
		const pts = normalizedRingPoints(
			[
				[100, 9.75],
				[100, 9.75],
				[100, 9.75]
			],
			40,
			0
		);
		for (const p of pts) {
			expect(p.x).toBe(20);
			expect(p.y).toBe(20);
		}
	});
	it('returns empty for an empty ring', () => {
		expect(normalizedRingPoints([], 40)).toEqual([]);
	});
});

describe('parcelStats', () => {
	it('computes node count, labels, area, perimeter and edges for a square', () => {
		const stats = parcelStats(SQUARE, 'sqm');
		expect(stats.nodeCount).toBe(4);
		expect(stats.labels).toEqual(['A', 'B', 'C', 'D']);
		expect(stats.areaSqm).toBeGreaterThan(12_000);
		expect(stats.areaSqm).toBeLessThan(12_300);
		expect(stats.areaLabel).toContain('m²');
		expect(stats.perimeterMeters).toBeGreaterThan(430);
		expect(stats.perimeterMeters).toBeLessThan(450);
		expect(stats.edges).toHaveLength(4);
		expect(stats.edges[0].name).toBe('A–B');
		expect(stats.edges[3].name).toBe('D–A'); // closing edge
		expect(stats.edges[0].meters).toBeGreaterThan(0);
	});
	it('tolerates a closed ring (no phantom zero-length closing edge)', () => {
		const stats = parcelStats([...SQUARE, SQUARE[0]], 'sqm');
		expect(stats.nodeCount).toBe(4);
		expect(stats.edges).toHaveLength(4);
	});
	it('switches area formatting with the selected unit', () => {
		expect(parcelStats(SQUARE, 'rai').areaLabel).toMatch(/rai/);
		expect(parcelStats(SQUARE, 'wah').areaLabel).toMatch(/wah/);
	});
});
