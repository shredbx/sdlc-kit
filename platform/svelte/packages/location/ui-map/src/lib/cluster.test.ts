import { describe, it, expect } from 'vitest';
import { declusterByPixel, markersCollide, type PixelPoint } from './cluster.js';

describe('declusterByPixel', () => {
	it('leaves well-separated points untouched (one cell each, own key)', () => {
		const pts: PixelPoint[] = [
			{ key: 'a', x: 0, y: 0, count: 1 },
			{ key: 'b', x: 200, y: 0, count: 1 },
			{ key: 'c', x: 0, y: 200, count: 1 }
		];
		const cells = declusterByPixel(pts, 52);
		expect(cells).toHaveLength(3);
		expect(cells.map((c) => c.key).sort()).toEqual(['a', 'b', 'c']);
		expect(cells.every((c) => c.memberKeys.length === 1)).toBe(true);
	});

	it('merges overlapping points into one cell, summing counts', () => {
		const pts: PixelPoint[] = [
			{ key: 'a', x: 100, y: 100, count: 3 },
			{ key: 'b', x: 110, y: 105, count: 4 }, // within 52px of a
			{ key: 'c', x: 500, y: 500, count: 1 } // far away
		];
		const cells = declusterByPixel(pts, 52);
		expect(cells).toHaveLength(2);
		const cluster = cells.find((c) => c.memberKeys.length > 1)!;
		expect(cluster.memberKeys).toEqual(['a', 'b']);
		expect(cluster.count).toBe(7);
		expect(cluster.x).toBe(105); // centroid mean
		expect(cluster.key).toBe('grp:a,b');
	});

	it('merges perfectly-coincident points (BR shared centroids)', () => {
		const pts: PixelPoint[] = [
			{ key: 'x', x: 300, y: 300, count: 1 },
			{ key: 'y', x: 300, y: 300, count: 5 }
		];
		const cells = declusterByPixel(pts, 52);
		expect(cells).toHaveLength(1);
		expect(cells[0].count).toBe(6);
		expect(cells[0].memberKeys).toEqual(['x', 'y']);
	});

	it('is deterministic — the cluster key is order-independent', () => {
		const a: PixelPoint[] = [
			{ key: 'b', x: 10, y: 10, count: 1 },
			{ key: 'a', x: 12, y: 11, count: 1 }
		];
		const b: PixelPoint[] = [
			{ key: 'a', x: 12, y: 11, count: 1 },
			{ key: 'b', x: 10, y: 10, count: 1 }
		];
		expect(declusterByPixel(a, 52)[0].key).toBe('grp:a,b');
		expect(declusterByPixel(b, 52)[0].key).toBe('grp:a,b');
	});

	it('dissolves clusters as points spread apart (zoom-in behaviour)', () => {
		const close: PixelPoint[] = [
			{ key: 'a', x: 100, y: 100, count: 1 },
			{ key: 'b', x: 120, y: 100, count: 1 }
		];
		const far: PixelPoint[] = [
			{ key: 'a', x: 100, y: 100, count: 1 },
			{ key: 'b', x: 300, y: 100, count: 1 } // same points, zoomed in → 200px apart
		];
		expect(declusterByPixel(close, 52)).toHaveLength(1); // merged when close
		expect(declusterByPixel(far, 52)).toHaveLength(2); // split when apart
	});
});

describe('markersCollide (rectangle mode)', () => {
	// Two ~100px-wide price pills, 70px apart center-to-center: a 52px radius says "clear"
	// but the pills visibly overlap — the exact bug the radius-only pass shipped.
	it('catches wide pills overlapping that the radius test misses', () => {
		const a: PixelPoint = { key: 'a', x: 100, y: 100, count: 1, w: 100, h: 36 };
		const b: PixelPoint = { key: 'b', x: 170, y: 100, count: 1, w: 100, h: 36 };
		expect(markersCollide(a, b, 52)).toBe(true); // rectangles overlap
		expect(markersCollide({ ...a, w: undefined, h: undefined }, { ...b, w: undefined, h: undefined }, 52)).toBe(false); // circle test would have let them collide on screen
	});

	it('clears pills separated vertically even when horizontally aligned', () => {
		const a: PixelPoint = { key: 'a', x: 100, y: 100, count: 1, w: 100, h: 36 };
		const b: PixelPoint = { key: 'b', x: 100, y: 160, count: 1, w: 100, h: 36 };
		expect(markersCollide(a, b, 52)).toBe(false); // 60px vertical gap between 36px-tall boxes
	});

	it('declusterByPixel honours rectangle sizes when provided', () => {
		const pts: PixelPoint[] = [
			{ key: 'a', x: 100, y: 100, count: 2, w: 100, h: 36 },
			{ key: 'b', x: 170, y: 100, count: 3, w: 100, h: 36 }, // overlapping pill
			{ key: 'c', x: 400, y: 100, count: 1, w: 100, h: 36 } // clear
		];
		const cells = declusterByPixel(pts, 52);
		expect(cells).toHaveLength(2);
		const merged = cells.find((c) => c.memberKeys.length > 1)!;
		expect(merged.memberKeys).toEqual(['a', 'b']);
		expect(merged.count).toBe(5);
	});
});
