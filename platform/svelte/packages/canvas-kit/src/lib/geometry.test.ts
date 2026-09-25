// TDD — geometry (resize handles + resize math, ported from land-canvas).
//
// CASE ENUMERATION
//   TC-GE-01 success  handleRects returns 8 handles
//   TC-GE-02 success  the se handle sits at the bottom-right corner
//   TC-GE-03 success  resize se grows width + height, origin unchanged
//   TC-GE-04 success  resize nw moves x/y and shrinks width/height
//   TC-GE-05 success  lockAspect preserves the width/height ratio
//   TC-GE-06 edge     min-size clamp holds (cannot shrink below MIN_SIZE)
//   TC-GE-07 regress  resizeLayer does not mutate the input layer
//   TC-GE-08 success  cropLayer n: dragging down increases crop_top 1:1
//   TC-GE-09 success  cropLayer s/e: dragging inward (negative delta) increases the crop
//   TC-GE-10 edge     gesture floor: outward drag clamps at 0 (never negative)
//   TC-GE-11 edge     gesture ceiling: clamps at side − oppositeCrop − 1 (never inverts)
//   TC-GE-12 success  crop accumulates from the seed's existing crop
//   TC-GE-13 regress  cropLayer does not mutate the input layer

import { describe, expect, it } from 'vitest';
import { handleRects, resizeLayer, cropLayer, rotationAtPointer, pointInLayer, projectToPixel, unprojectFromPixel, MIN_SIZE } from './geometry.js';
import { makeImageLayer } from './fixtures.js';

const box = (over = {}) => makeImageLayer({ x: 100, y: 100, width: 200, height: 100, ...over });

describe('handleRects', () => {
	it('TC-GE-01 returns the 8 resize handles', () => {
		const rects = handleRects(box(), 8);
		expect(rects.map((r) => r.handle).sort()).toEqual(['e', 'n', 'ne', 'nw', 's', 'se', 'sw', 'w']);
	});

	it('TC-GE-02 centres the se handle on the bottom-right corner', () => {
		const se = handleRects(box(), 8).find((r) => r.handle === 'se')!;
		// corner = (x+w, y+h) = (300, 200); box is centred → top-left = corner - half
		expect(se.x).toBe(300 - 4);
		expect(se.y).toBe(200 - 4);
		expect(se.w).toBe(8);
	});
});

describe('resizeLayer', () => {
	it('TC-GE-03 se handle grows width + height, origin unchanged', () => {
		const g = resizeLayer(box(), 'se', 50, 30, false);
		expect(g.x).toBe(100);
		expect(g.y).toBe(100);
		expect(g.width).toBe(250);
		expect(g.height).toBe(130);
	});

	it('TC-GE-04 nw handle moves the origin and shrinks the box', () => {
		const g = resizeLayer(box(), 'nw', 20, 10, false);
		expect(g.x).toBe(120);
		expect(g.y).toBe(110);
		expect(g.width).toBe(180);
		expect(g.height).toBe(90);
	});

	it('TC-GE-05 lockAspect preserves the original ratio', () => {
		// 200×100 → ratio 2. Drag se by (100, 0): width would be 300, height 100
		// (ratio 3 > 2) → height re-derived to width/ratio = 150.
		const g = resizeLayer(box(), 'se', 100, 0, true);
		expect(g.width / g.height).toBeCloseTo(2, 5);
	});

	it('TC-GE-06 clamps to MIN_SIZE (cannot shrink below the minimum)', () => {
		const g = resizeLayer(box({ width: 200, height: 100 }), 'e', -500, 0, false);
		expect(g.width).toBe(MIN_SIZE);
	});

	it('TC-GE-07 does not mutate the input layer', () => {
		const layer = box();
		resizeLayer(layer, 'se', 40, 40, false);
		expect(layer.width).toBe(200);
		expect(layer.height).toBe(100);
	});
});

describe('cropLayer', () => {
	it('TC-GE-08 n edge: dragging down increases crop_top 1:1', () => {
		expect(cropLayer(box(), 'n', 0, 30)).toEqual({ crop_top: 30 });
	});

	it('TC-GE-09 s/e edges: dragging inward (negative delta) increases the crop', () => {
		expect(cropLayer(box(), 's', 0, -25)).toEqual({ crop_bottom: 25 });
		expect(cropLayer(box(), 'e', -40, 0)).toEqual({ crop_right: 40 });
		expect(cropLayer(box(), 'w', 15, 0)).toEqual({ crop_left: 15 });
	});

	it('TC-GE-10 gesture floor: an outward drag clamps at 0', () => {
		expect(cropLayer(box(), 'n', 0, -50)).toEqual({ crop_top: 0 });
		expect(cropLayer(box({ crop_left: 10 }), 'w', -30, 0)).toEqual({ crop_left: 0 });
	});

	it('TC-GE-11 gesture ceiling: clamps at side − oppositeCrop − 1 (never inverts)', () => {
		// height 100, crop_bottom 30 → crop_top may reach at most 69 (1px window survives)
		expect(cropLayer(box({ crop_bottom: 30 }), 'n', 0, 500)).toEqual({ crop_top: 69 });
		// width 200, crop_left 150 → crop_right caps at 49
		expect(cropLayer(box({ crop_left: 150 }), 'e', -999, 0)).toEqual({ crop_right: 49 });
	});

	it('TC-GE-12 accumulates from the seed crop', () => {
		expect(cropLayer(box({ crop_top: 20 }), 'n', 0, 15)).toEqual({ crop_top: 35 });
	});

	it('TC-GE-13 does not mutate the input layer', () => {
		const layer = box({ crop_top: 5 });
		cropLayer(layer, 'n', 0, 40);
		expect(layer.crop_top).toBe(5);
	});

	it('TC-GE-14 degenerate seed (opposite crop ≥ side) → no-op, never snaps to 0', () => {
		// crop_bottom 200 on height 100: no gesture range — an existing typed
		// crop_top must survive untouched.
		expect(cropLayer(box({ crop_bottom: 200, crop_top: 40 }), 'n', 0, 1)).toEqual({});
	});
});

describe('rotationAtPointer', () => {
	// Centre at (100, 100); the grip's bearing from "up" is the layer rotation.
	it('TC-GE-15 pointer straight up → 0°', () => {
		expect(rotationAtPointer(100, 100, 100, 40)).toBe(0);
	});

	it('TC-GE-16 pointer to the right → 90° (clockwise, matches canvas y-down)', () => {
		expect(rotationAtPointer(100, 100, 160, 100)).toBe(90);
	});

	it('TC-GE-17 pointer straight down → 180°', () => {
		expect(rotationAtPointer(100, 100, 100, 160)).toBe(180);
	});

	it('TC-GE-18 pointer to the left → 270° (normalised to [0,360))', () => {
		expect(rotationAtPointer(100, 100, 40, 100)).toBe(270);
	});

	it('TC-GE-19 snap rounds to the nearest increment', () => {
		// raw ≈ 100°; snap 15 → nearest is 105.
		expect(rotationAtPointer(100, 100, 159, 110, 15)).toBe(105);
	});

	it('TC-GE-20 snap that lands on 360 normalises to 0', () => {
		// pointer just left of straight-up (raw ≈ 356°); snap 15 → 360 → 0.
		expect(rotationAtPointer(100, 100, 96, 40, 15)).toBe(0);
	});
});

// --- Web Mercator projection (Decision #0297) --------------------------------
//
// Known-value tests against a simple reference:
//   At zoom 14, the world is 256 × 2^14 = 4,194,304 px wide.
//   The center of the image maps to (size.width/2, size.height/2).
//   A point at the same lat/lng as the center must land exactly there.
//   A point shifted 1° east in longitude shifts right by (1/360)*256*scale px.

describe('projectToPixel — Web Mercator projection', () => {
	const CENTER = { lat: 13.7563, lng: 100.5018 };
	const SIZE = { width: 640, height: 400 };
	const ZOOM = 14;

	it('TC-GE-P1 center coords project to the image centre pixel', () => {
		const px = projectToPixel(CENTER.lat, CENTER.lng, CENTER, ZOOM, SIZE);
		expect(px.x).toBeCloseTo(SIZE.width / 2, 3);
		expect(px.y).toBeCloseTo(SIZE.height / 2, 3);
	});

	it('TC-GE-P2 point east of center projects to a larger x', () => {
		const px = projectToPixel(CENTER.lat, CENTER.lng + 0.1, CENTER, ZOOM, SIZE);
		expect(px.x).toBeGreaterThan(SIZE.width / 2);
		expect(px.y).toBeCloseTo(SIZE.height / 2, 1); // same lat → same y
	});

	it('TC-GE-P3 point north of center (higher lat) projects to a smaller y (y-down axis)', () => {
		const px = projectToPixel(CENTER.lat + 0.1, CENTER.lng, CENTER, ZOOM, SIZE);
		expect(px.y).toBeLessThan(SIZE.height / 2);
		expect(px.x).toBeCloseTo(SIZE.width / 2, 1); // same lng → same x
	});

	it('TC-GE-P4 higher zoom → larger pixel offset for the same geo delta', () => {
		const z14 = projectToPixel(CENTER.lat, CENTER.lng + 0.1, CENTER, 14, SIZE);
		const z15 = projectToPixel(CENTER.lat, CENTER.lng + 0.1, CENTER, 15, SIZE);
		// Each zoom level doubles the scale → pixel offset doubles.
		expect(z15.x - SIZE.width / 2).toBeCloseTo((z14.x - SIZE.width / 2) * 2, 1);
	});

	it('TC-GE-P5 does not clamp out-of-image coords', () => {
		// A point far east maps outside the 640-wide image box — caller decides.
		const px = projectToPixel(CENTER.lat, CENTER.lng + 10, CENTER, ZOOM, SIZE);
		expect(px.x).toBeGreaterThan(SIZE.width);
	});
});

describe('unprojectFromPixel — Web Mercator inverse', () => {
	const CENTER = { lat: 13.7563, lng: 100.5018 };
	const SIZE = { width: 640, height: 400 };
	const ZOOM = 14;

	it('TC-GE-U1 round-trips: project → unproject recovers the original lat/lng', () => {
		const testPoints = [
			{ lat: 13.7563, lng: 100.5018 }, // center
			{ lat: 13.8, lng: 100.6 },
			{ lat: 13.65, lng: 100.4 }
		];
		for (const { lat, lng } of testPoints) {
			const px = projectToPixel(lat, lng, CENTER, ZOOM, SIZE);
			const geo = unprojectFromPixel(px.x, px.y, CENTER, ZOOM, SIZE);
			expect(geo.lat).toBeCloseTo(lat, 4);
			expect(geo.lng).toBeCloseTo(lng, 4);
		}
	});

	it('TC-GE-U2 image-centre pixel → center lat/lng', () => {
		const geo = unprojectFromPixel(SIZE.width / 2, SIZE.height / 2, CENTER, ZOOM, SIZE);
		expect(geo.lat).toBeCloseTo(CENTER.lat, 4);
		expect(geo.lng).toBeCloseTo(CENTER.lng, 4);
	});
});

describe('pointInLayer (selection — rotation + tile aware)', () => {
	// A 50×50 source at (100,100); a 400×400 page as the tile fill region.
	const SRC = () => makeImageLayer({ x: 100, y: 100, width: 50, height: 50 });
	const REGION = { minX: 0, minY: 0, maxX: 400, maxY: 400 };

	it('TC-GE-H1 hits the source box, misses outside it', () => {
		expect(pointInLayer(SRC(), 120, 120)).toBe(true); // inside
		expect(pointInLayer(SRC(), 10, 10)).toBe(false); // far outside
		expect(pointInLayer(SRC(), 160, 120)).toBe(false); // just past the right edge (x>150)
	});

	it('TC-GE-H2 degenerate (zero-size) layer is never hit', () => {
		const dead = makeImageLayer({ x: 100, y: 100, width: 0, height: 0 });
		expect(pointInLayer(dead, 100, 100)).toBe(false);
	});

	it('TC-GE-H3 a click on a TILE COPY resolves to the source (true)', () => {
		// repeatH, gapX 50 → pitch 100 → copies at x = …,100,200,300,…; a click inside the
		// copy at x=300 (300..350) must select this source layer.
		const tiled = makeImageLayer({ x: 100, y: 100, width: 50, height: 50, tile: { repeatH: true, repeatV: false, gapX: 50, gapY: 0 } });
		expect(pointInLayer(tiled, 320, 120, REGION)).toBe(true); // on the x=300 copy
		expect(pointInLayer(tiled, 320, 320, REGION)).toBe(false); // no vertical repeat → row y=100 only
	});

	it('TC-GE-H4 without a region a tiling layer only hits its source box (no phantom copies)', () => {
		const tiled = makeImageLayer({ x: 100, y: 100, width: 50, height: 50, tile: { repeatH: true, repeatV: true, gapX: 50, gapY: 50 } });
		expect(pointInLayer(tiled, 120, 120)).toBe(true); // source
		expect(pointInLayer(tiled, 320, 320)).toBe(false); // region omitted → copies not tested
	});

	it('TC-GE-H5 honours rotation — a point on the rotated box hits, its un-rotated position misses', () => {
		// 90° about the centre (125,125): the box (100..150,100..150) maps to itself (square,
		// centred), but a point offset purely on one axis swaps axes. Use a tall box so rotation
		// is observable: 20×80 at (140,60) → centre (150,100); rotate 90°.
		const tall = makeImageLayer({ x: 140, y: 60, width: 20, height: 80, rotation: 90 });
		// After +90° the tall box renders WIDE (≈110..190 in x, 90..110 in y around centre 150,100).
		expect(pointInLayer(tall, 185, 100)).toBe(true); // inside the rotated (wide) footprint
		expect(pointInLayer(tall, 150, 135)).toBe(false); // inside the UN-rotated (tall) footprint, now empty
	});
});
