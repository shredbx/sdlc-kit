// TDD — watermark tiling geometry (Slice C0 · #0299). markBounds is the SINGLE
// canonical "mark unit" bbox shared by the tiled preview, the publish-time mark-unit
// export, and the apply worker. Lifted faithfully from CanvasStage's private copy.
//
// CASE ENUMERATION
//   TC-WT-01 success  a single mark layer → its own bbox
//   TC-WT-02 success  two layers → the union bbox
//   TC-WT-03 success  a system:true layer is excluded from the union
//   TC-WT-04 success  a visible:false layer is excluded from the union
//   TC-WT-05 edge     bounds clamp to the page (off-page extents are trimmed)
//   TC-WT-06 edge     no qualifying layers → null
//   TC-WT-07 edge     a zero-size layer is skipped (degenerate → null when it's the only one)

import { describe, expect, it } from 'vitest';
import { markBounds, tilePositions } from './watermark-tiling.js';
import { makeImageLayer } from './fixtures.js';
import type { TileEffect } from './types/layer.js';

describe('markBounds (#0299)', () => {
	it('TC-WT-01 returns a single mark layer’s own bbox', () => {
		const layers = [makeImageLayer({ id: 'a', x: 100, y: 80, width: 200, height: 120 })];
		expect(markBounds(layers, 1080, 1350)).toEqual({ x: 100, y: 80, w: 200, h: 120 });
	});

	it('TC-WT-02 unions two layers into the enclosing bbox', () => {
		const layers = [
			makeImageLayer({ id: 'a', x: 100, y: 80, width: 200, height: 120 }), // → (100,80)-(300,200)
			makeImageLayer({ id: 'b', x: 400, y: 50, width: 100, height: 100 }) // → (400,50)-(500,150)
		];
		expect(markBounds(layers, 1080, 1350)).toEqual({ x: 100, y: 50, w: 400, h: 150 });
	});

	it('TC-WT-03 excludes a system:true layer from the union', () => {
		const layers = [
			makeImageLayer({ id: 'mark', x: 100, y: 80, width: 200, height: 120 }),
			// A full-bleed system background would otherwise blow the bbox out to the page.
			makeImageLayer({ id: 'bg', system: true, x: 0, y: 0, width: 1080, height: 1350 })
		];
		expect(markBounds(layers, 1080, 1350)).toEqual({ x: 100, y: 80, w: 200, h: 120 });
	});

	it('TC-WT-04 excludes a visible:false layer from the union', () => {
		const layers = [
			makeImageLayer({ id: 'mark', x: 100, y: 80, width: 200, height: 120 }),
			makeImageLayer({ id: 'hidden', visible: false, x: 0, y: 0, width: 1080, height: 1350 })
		];
		expect(markBounds(layers, 1080, 1350)).toEqual({ x: 100, y: 80, w: 200, h: 120 });
	});

	it('TC-WT-05 clamps the bounds to the page extents', () => {
		// A layer overflowing the page on the top-left AND bottom-right corners.
		const layers = [makeImageLayer({ id: 'big', x: -50, y: -30, width: 1200, height: 1500 })];
		// minX/minY clamp to 0; maxX/maxY clamp to page width/height.
		expect(markBounds(layers, 1080, 1350)).toEqual({ x: 0, y: 0, w: 1080, h: 1350 });
	});

	it('TC-WT-06 returns null when no layer qualifies', () => {
		expect(markBounds([], 1080, 1350)).toBeNull();
		// Only a system + a hidden layer → nothing to union.
		const layers = [
			makeImageLayer({ id: 'bg', system: true, x: 0, y: 0, width: 1080, height: 1350 }),
			makeImageLayer({ id: 'hidden', visible: false, x: 10, y: 10, width: 50, height: 50 })
		];
		expect(markBounds(layers, 1080, 1350)).toBeNull();
	});

	it('TC-WT-07 skips a zero-size layer (degenerate → null when it is the only one)', () => {
		const layers = [makeImageLayer({ id: 'empty', x: 100, y: 100, width: 0, height: 0 })];
		expect(markBounds(layers, 1080, 1350)).toBeNull();
	});
});

describe('tilePositions (WM tiling effect)', () => {
	const BOX = { x: 100, y: 100, w: 50, h: 50 };
	const REGION = { minX: 0, minY: 0, maxX: 400, maxY: 400 };
	const tile = (over: Partial<TileEffect> = {}): TileEffect => ({
		repeatH: false,
		repeatV: false,
		gapX: 0,
		gapY: 0,
		...over
	});

	// TC-TP-01 both repeats off → only the source copy (index 0,0) at the box origin.
	it('TC-TP-01 returns only the source when neither axis repeats', () => {
		expect(tilePositions(BOX, tile(), REGION)).toEqual([{ x: 100, y: 100, i: 0, j: 0 }]);
	});

	// TC-TP-02 repeat-H only → a single ROW; pitch = w+gap = 100; copies cover the region
	// symmetrically around the source (i=-1..3), the source landing exactly at box.x.
	it('TC-TP-02 tiles a row when only repeatH is on', () => {
		const out = tilePositions(BOX, tile({ repeatH: true, gapX: 50 }), REGION);
		expect(out.map((p) => p.x)).toEqual([0, 100, 200, 300, 400]);
		expect(out.every((p) => p.y === 100 && p.j === 0)).toBe(true);
		// The source copy is present, at the box origin, unshifted.
		expect(out).toContainEqual({ x: 100, y: 100, i: 0, j: 0 });
	});

	// TC-TP-03 repeat-V only → a single COLUMN (mirror of TC-TP-02 on the y axis).
	it('TC-TP-03 tiles a column when only repeatV is on', () => {
		const out = tilePositions(BOX, tile({ repeatV: true, gapY: 50 }), REGION);
		expect(out.map((p) => p.y)).toEqual([0, 100, 200, 300, 400]);
		expect(out.every((p) => p.x === 100 && p.i === 0)).toBe(true);
	});

	// TC-TP-04 both axes → a full grid; count = rows × cols (5 × 5 here).
	it('TC-TP-04 tiles a full grid when both axes repeat', () => {
		const out = tilePositions(BOX, tile({ repeatH: true, repeatV: true, gapX: 50, gapY: 50 }), REGION);
		expect(out).toHaveLength(25);
		expect(out).toContainEqual({ x: 100, y: 100, i: 0, j: 0 });
	});

	// TC-TP-05 a larger gap widens the pitch → fewer copies fit the same region.
	it('TC-TP-05 a larger gap produces fewer copies', () => {
		const dense = tilePositions(BOX, tile({ repeatH: true, gapX: 50 }), REGION);
		const sparse = tilePositions(BOX, tile({ repeatH: true, gapX: 350 }), REGION);
		expect(sparse.length).toBeLessThan(dense.length);
		expect(sparse).toContainEqual({ x: 100, y: 100, i: 0, j: 0 }); // source always kept
	});

	// TC-TP-06 a degenerate (zero-size) source never divides by zero — just the source.
	it('TC-TP-06 returns only the source for a zero-size box', () => {
		expect(tilePositions({ x: 10, y: 20, w: 0, h: 50 }, tile({ repeatH: true, repeatV: true }), REGION)).toEqual([
			{ x: 10, y: 20, i: 0, j: 0 }
		]);
	});

	// TC-TP-07 a near-zero pitch over a huge region stays finite + capped (no runaway),
	// and still centres on the source (index 0 present).
	it('TC-TP-07 caps the copy count for a pathological pitch/region', () => {
		const huge = { minX: -100000, minY: 0, maxX: 100000, maxY: 0 };
		const out = tilePositions(BOX, tile({ repeatH: true, gapX: -BOX.w }), huge); // pitch floored to 1
		expect(out.length).toBeLessThanOrEqual(513); // MAX_PER_AXIS (512) + the source
		expect(out.some((p) => p.i === 0)).toBe(true);
	});
});
