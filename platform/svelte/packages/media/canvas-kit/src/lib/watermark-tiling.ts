// Watermark tiling geometry (Slice C0 · #0299) — the SINGLE canonical "mark unit"
// bounds, shared by the tiled PREVIEW (CanvasStage), the publish-time mark-unit
// EXPORT (export.ts → exportMarkUnit), and (downstream) the apply worker. The
// document IS the full-frame overlay, so the repeat TILE is a SUB-region of the
// page: the axis-aligned bbox of the visible, non-system layers, clamped to the
// page. Defining it ONCE (define-tiling-once) keeps the preview, the published
// mark unit, and apply re-tiling pixel-identical. PURE → unit-testable.

import type { Layer, TileEffect } from './types/layer.js';

/**
 * The MARK UNIT bounds (#0299) — the axis-aligned bbox of the visible, non-system
 * layers in PAGE px, clamped to the page. This is the repeat TILE the preview/apply
 * stamps across the frame (the document itself is the full overlay, so the tile is a
 * SUB-region, not the page). Returns null when the mark is empty (a blank overlay →
 * backdrop only). Per-layer rotation is not unrolled (mark layers aren't individually
 * rotated; the whole pattern rotates via the document's watermark.rotation).
 */
export function markBounds(
	layers: readonly Layer[],
	pageWidth: number,
	pageHeight: number
): { x: number; y: number; w: number; h: number } | null {
	let minX = Infinity;
	let minY = Infinity;
	let maxX = -Infinity;
	let maxY = -Infinity;
	for (const l of layers) {
		if (l.system || l.visible === false) continue;
		const w = l.width ?? 0;
		const h = l.height ?? 0;
		if (w <= 0 || h <= 0) continue;
		const x = l.x ?? 0;
		const y = l.y ?? 0;
		minX = Math.min(minX, x);
		minY = Math.min(minY, y);
		maxX = Math.max(maxX, x + w);
		maxY = Math.max(maxY, y + h);
	}
	if (!Number.isFinite(minX)) return null;
	minX = Math.max(0, minX);
	minY = Math.max(0, minY);
	maxX = Math.min(pageWidth, maxX);
	maxY = Math.min(pageHeight, maxY);
	const w = maxX - minX;
	const h = maxY - minY;
	return w > 0 && h > 0 ? { x: minX, y: minY, w, h } : null;
}

/** A copy origin (top-left, page px) produced by `tilePositions`. `i`/`j` are the
 *  grid indices (0,0 = the source itself) so a caller can identify the source copy. */
export interface TilePlacement {
	x: number;
	y: number;
	i: number;
	j: number;
}

/** Defensive cap on copies generated per axis — a pathological gap (≈0 / negative)
 *  over a large region must never spin millions of draws. Real watermarks need a few
 *  dozen at most; 512/axis is far above any sane density yet bounds the work. */
const MAX_PER_AXIS = 512;

/**
 * The origins (top-left, page px) of every tiled COPY of a source `box`, anchored so
 * one copy sits exactly at the source (index 0,0) and the grid steps by
 * pitch = size + gap on each enabled axis to FILL `region`. Copies are emitted in the
 * un-rotated frame; the caller applies the pattern rotation around the source centre
 * (rotation is uniform, so generating in local space then rotating is equivalent and
 * keeps this pure + testable).
 *
 * Axis gating mirrors the user model: `repeatH` off → a single column (i=0 only),
 * `repeatV` off → a single row (j=0 only); both off → just the source. The `region`
 * is supplied by the caller already OVERSIZED (≥ the diagonal of the visible frame)
 * when a rotation is in play, so the rotated pattern leaves no uncovered corner — the
 * √2 coverage technique, decided by the renderer, not baked in here.
 *
 * Degenerate source (non-positive width/height) → just the source origin (never a
 * divide-by-zero or an empty result that would drop the layer). PURE → unit-testable.
 */
export function tilePositions(
	box: { x: number; y: number; w: number; h: number },
	tile: TileEffect,
	region: { minX: number; minY: number; maxX: number; maxY: number }
): TilePlacement[] {
	// Pitch = source size + gap, floored at 1px so a zero/negative gap can never make
	// the grid step nowhere (which would loop forever). A degenerate box short-circuits.
	if (!(box.w > 0) || !(box.h > 0)) return [{ x: box.x, y: box.y, i: 0, j: 0 }];
	const pitchX = Math.max(1, box.w + tile.gapX);
	const pitchY = Math.max(1, box.h + tile.gapY);

	// Inclusive index range on each axis so the copy [origin, origin+size] still
	// overlaps the region. A copy at index i has left = box.x + i*pitchX; it is visible
	// when left < region.maxX AND left + box.w > region.minX. Solving for i:
	//   iMin = ceil((region.minX - box.w - box.x) / pitchX)
	//   iMax = floor((region.maxX - box.x) / pitchX)
	const range = (
		repeat: boolean,
		origin: number,
		size: number,
		pitch: number,
		lo: number,
		hi: number
	): [number, number] => {
		if (!repeat) return [0, 0]; // single line on this axis — only the source index
		const min = Math.ceil((lo - size - origin) / pitch);
		const max = Math.floor((hi - origin) / pitch);
		if (max < min) return [0, 0]; // region degenerate → at least the source
		// Clamp the span to the safety cap, keeping it centred on the source (i=0).
		if (max - min > MAX_PER_AXIS) {
			const half = Math.floor(MAX_PER_AXIS / 2);
			return [Math.max(min, -half), Math.min(max, half)];
		}
		return [min, max];
	};

	const [iMin, iMax] = range(tile.repeatH, box.x, box.w, pitchX, region.minX, region.maxX);
	const [jMin, jMax] = range(tile.repeatV, box.y, box.h, pitchY, region.minY, region.maxY);

	const out: TilePlacement[] = [];
	for (let j = jMin; j <= jMax; j++) {
		for (let i = iMin; i <= iMax; i++) {
			// Normalise -0 → 0 (Math.ceil can yield -0) so index equality is stable for callers.
			out.push({ x: box.x + i * pitchX, y: box.y + j * pitchY, i: i === 0 ? 0 : i, j: j === 0 ? 0 : j });
		}
	}
	return out;
}
