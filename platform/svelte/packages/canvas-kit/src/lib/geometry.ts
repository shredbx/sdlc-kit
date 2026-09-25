// Pure resize/handle math — PORTED from clients/andrei/projects/land-canvas
// (apps/web/svelte/src/lib/components/Canvas.svelte: getHandles ~164-181,
// HANDLE_CURSORS ~204-207, the resize delta math ~414-437). The land-canvas
// version is bound to drag-event state inside the Svelte component; here the
// math is extracted into pure, testable functions. The Svelte component
// (CanvasStage) keeps the event binding and feeds deltas in artboard pixels.
//
// HEADLESS: no Svelte runtime, no DOM. Pure functions over a Layer's geometry.

import type { Layer } from './types/layer.js';
import { tilePositions } from './watermark-tiling.js';

/** The eight resize handles, named by compass direction. */
export type Handle = 'nw' | 'n' | 'ne' | 'e' | 'se' | 's' | 'sw' | 'w';

/** Minimum width/height a resize may shrink a layer to (artboard pixels). */
export const MIN_SIZE = 8;

/** A handle's hit-box, in artboard pixels (centred on the handle point). */
export interface HandleRect {
	handle: Handle;
	x: number;
	y: number;
	w: number;
	h: number;
}

/** CSS cursor per handle (ported from land-canvas HANDLE_CURSORS). */
export const HANDLE_CURSORS: Record<Handle, string> = {
	nw: 'nwse-resize',
	ne: 'nesw-resize',
	sw: 'nesw-resize',
	se: 'nwse-resize',
	n: 'ns-resize',
	s: 'ns-resize',
	e: 'ew-resize',
	w: 'ew-resize'
};

/**
 * The 8 handle hit-boxes for a layer, in artboard pixels. Each box is
 * `handleSize`×`handleSize`, centred on the corner/edge-midpoint of the layer's
 * bounding rectangle (layer.x/y/width/height). Ported from getHandles().
 */
export function handleRects(layer: Layer, handleSize: number): HandleRect[] {
	const x = layer.x;
	const y = layer.y;
	const w = layer.width ?? 0;
	const h = layer.height ?? 0;
	const mx = x + w / 2;
	const my = y + h / 2;
	const half = handleSize / 2;
	const points: { handle: Handle; cx: number; cy: number }[] = [
		{ handle: 'nw', cx: x, cy: y },
		{ handle: 'n', cx: mx, cy: y },
		{ handle: 'ne', cx: x + w, cy: y },
		{ handle: 'e', cx: x + w, cy: my },
		{ handle: 'se', cx: x + w, cy: y + h },
		{ handle: 's', cx: mx, cy: y + h },
		{ handle: 'sw', cx: x, cy: y + h },
		{ handle: 'w', cx: x, cy: my }
	];
	return points.map((p) => ({ handle: p.handle, x: p.cx - half, y: p.cy - half, w: handleSize, h: handleSize }));
}

/** New geometry produced by a resize drag. */
export interface Geometry {
	x: number;
	y: number;
	width: number;
	height: number;
}

/**
 * Apply a drag delta (dx, dy in artboard pixels) to the given handle and return
 * the new geometry. PURE — never mutates the input layer. Enforces MIN_SIZE.
 * When `lockAspect` is true the original width/height ratio is preserved (the
 * Shift-key behaviour ported from Canvas.svelte ~427-437). Positions are
 * rounded to integers (land-canvas parity).
 */
export function resizeLayer(layer: Layer, handle: Handle, dx: number, dy: number, lockAspect = false): Geometry {
	const origX = layer.x;
	const origY = layer.y;
	const origW = layer.width ?? 0;
	const origH = layer.height ?? 0;

	let newX = origX;
	let newY = origY;
	let newW = origW;
	let newH = origH;

	if (handle.includes('e')) {
		newW = Math.max(MIN_SIZE, origW + dx);
	}
	if (handle.includes('w')) {
		newW = Math.max(MIN_SIZE, origW - dx);
		newX = origX + dx;
	}
	if (handle.includes('s')) {
		newH = Math.max(MIN_SIZE, origH + dy);
	}
	if (handle.includes('n')) {
		newH = Math.max(MIN_SIZE, origH - dy);
		newY = origY + dy;
	}

	// Shift = lock aspect ratio (port of Canvas.svelte ~427-437).
	if (lockAspect && origW > 0 && origH > 0 && newH > 0) {
		const ratio = origW / origH;
		if (newW / newH > ratio) {
			newW = newH * ratio;
		} else {
			newH = newW / ratio;
		}
		// Re-anchor handles that move the origin so the opposite edge stays put.
		if (handle.includes('w')) newX = origX + origW - newW;
		if (handle.includes('n')) newY = origY + origH - newH;
	}

	return {
		x: Math.round(newX),
		y: Math.round(newY),
		width: Math.round(newW),
		height: Math.round(newH)
	};
}

// --- Hit-testing (selection, rotation- + tile-aware) ------------------------

/** A page-space fill region (px) — mirrors render.ts TileRegion; the tiling hit-test
 *  walks the SAME copies the renderer drew across it. */
export interface HitRegion {
	minX: number;
	minY: number;
	maxX: number;
	maxY: number;
}

/**
 * Whether (px, py) — artboard pixels — falls on `layer`, honouring BOTH the layer's
 * rotation AND its tiling effect. The point is inverse-rotated into the layer's
 * un-rotated frame (rotation is around the box centre, matching the renderer's
 * applyRotation), then tested against the source box and — when the layer tiles and a
 * fill `region` is supplied — every repeated copy (the same origins `tilePositions`
 * feeds the renderer). A hit on ANY copy returns true, so the editor resolves a click
 * on a repeat back to this one SOURCE layer; copies are never selectable phantoms.
 * No tile / no region → just the source box (rotation still honoured — this also fixes
 * the plain rotated-layer hit-test that a raw AABB gets wrong). PURE — geometry only.
 */
export function pointInLayer(layer: Layer, px: number, py: number, region?: HitRegion): boolean {
	const w = layer.width ?? 0;
	const h = layer.height ?? 0;
	if (w <= 0 || h <= 0) return false;

	// Inverse-rotate the point about the box centre: in the un-rotated frame the source
	// box and every tile copy are axis-aligned, so a plain AABB test is exact.
	let qx = px;
	let qy = py;
	const rot = layer.rotation ?? 0;
	if (rot) {
		const cx = layer.x + w / 2;
		const cy = layer.y + h / 2;
		const a = (-rot * Math.PI) / 180; // inverse of the renderer's +rot rotation
		const cos = Math.cos(a);
		const sin = Math.sin(a);
		const ox = px - cx;
		const oy = py - cy;
		qx = cx + ox * cos - oy * sin;
		qy = cy + ox * sin + oy * cos;
	}

	const inBox = (bx: number, by: number): boolean => qx >= bx && qx <= bx + w && qy >= by && qy <= by + h;

	// The source (index 0,0) is always drawn → always test it first (cheap common case).
	if (inBox(layer.x, layer.y)) return true;

	const tile = layer.tile;
	if (tile && (tile.repeatH || tile.repeatV) && region) {
		for (const p of tilePositions({ x: layer.x, y: layer.y, w, h }, tile, region)) {
			if (inBox(p.x, p.y)) return true;
		}
	}
	return false;
}

// --- Crop gesture math (scissor handles, 2026-06-07) ------------------------

/** The four croppable edges — the on-canvas scissor handles next to the
 *  edge-midpoint pins, and Alt-drag on the matching resize pin. */
export type CropEdge = 'n' | 'e' | 's' | 'w';

/** Crop-scalar key per edge. */
export const CROP_KEYS: Record<CropEdge, 'crop_top' | 'crop_right' | 'crop_bottom' | 'crop_left'> = {
	n: 'crop_top',
	e: 'crop_right',
	s: 'crop_bottom',
	w: 'crop_left'
};

/**
 * Apply a drag delta (dx, dy in artboard pixels) to one crop edge and return
 * the patch. PURE — never mutates the input layer. Dragging INWARD increases
 * the crop (n→down, s→up, w→right, e→left). The GESTURE clamp is
 * [0, side − oppositeCrop − 1]: a drag can never invert the crop window or
 * collapse it to nothing under the cursor — typed/keyframed values may still
 * reach full-wipe zero-area (the renderer clamps those safely). Like
 * resizeLayer, feed the drag-START geometry so the result is independent of
 * intermediate rounding.
 */
export function cropLayer(layer: Layer, edge: CropEdge, dx: number, dy: number): Partial<Layer> {
	const w = layer.width ?? 0;
	const h = layer.height ?? 0;
	const t = layer.crop_top ?? 0;
	const r = layer.crop_right ?? 0;
	const b = layer.crop_bottom ?? 0;
	const l = layer.crop_left ?? 0;
	let next: number;
	let limit: number;
	switch (edge) {
		case 'n':
			next = t + dy;
			limit = h - b - 1;
			break;
		case 's':
			next = b - dy;
			limit = h - t - 1;
			break;
		case 'w':
			next = l + dx;
			limit = w - r - 1;
			break;
		case 'e':
			next = r - dx;
			limit = w - l - 1;
			break;
	}
	// Degenerate seed (opposite crop already ≥ the side, e.g. typed via the
	// Inspector): no gesture range exists — no-op rather than snapping an
	// existing value to 0.
	if (limit < 0) return {};
	const clamped = Math.min(limit, Math.max(0, next));
	return { [CROP_KEYS[edge]]: Math.round(clamped) };
}

// --- Rotation gesture math (rotation grip, 2026-06-08) ----------------------

/**
 * The rotation (degrees, clockwise, 0 = straight up) the rotation grip reports
 * when the pointer is at (px, py) relative to the layer's box CENTRE (cx, cy) —
 * all in artboard pixels. The grip sits above the box, so an undragged grip
 * already reads the layer's current rotation; dragging the pointer around the
 * centre sets rotation to the pointer's bearing from "up". Measured CLOCKWISE to
 * match the canvas-2D transform (y-down): pointer right of the centre → +90°,
 * below → 180°, left → 270°. Normalised to [0, 360). When `snap` > 0 the angle
 * snaps to the nearest `snap`° increment (the Shift-key behaviour, e.g. 15°).
 * PURE — the CanvasStage feeds raw artboard coordinates and applies the result
 * as `layer.rotation` (the same property the kit renderer rotates by).
 */
export function rotationAtPointer(cx: number, cy: number, px: number, py: number, snap = 0): number {
	// atan2(dx, -dy): 0 when the pointer is straight up, increasing clockwise.
	let deg = (Math.atan2(px - cx, -(py - cy)) * 180) / Math.PI;
	if (deg < 0) deg += 360;
	if (snap > 0) deg = Math.round(deg / snap) * snap;
	return Math.round(deg) % 360;
}

// --- Web Mercator geo↔pixel projection (Decision #0297) --------------------
//
// Google Static Maps uses the Web Mercator projection (EPSG:3857). Given a map
// center, zoom level, and output size in pixels, these functions convert between
// geographic coordinates (lat/lng) and pixel offsets within the map image.
//
// Reference: https://developers.google.com/maps/documentation/javascript/coordinates
// The formula is identical to Google Maps JS API's LatLng↔Point math:
//   worldX = (lng + 180) / 360 * 256
//   siny = sin(lat * π / 180) clamped to [-1+ε, 1-ε]
//   worldY = (0.5 - ln((1 + siny) / (1 - siny)) / (4π)) * 256
// At zoom z the world is 256 * 2^z pixels; the image maps [center ± size/2].

/** A pixel coordinate within a 2D box. */
export interface PixelPoint {
	x: number;
	y: number;
}

/** The world-map pixel for a geo coordinate at the given tile scale (zoom). */
function worldPixel(lat: number, lng: number): PixelPoint {
	const WORLD_SIZE = 256;
	const x = ((lng + 180) / 360) * WORLD_SIZE;
	const sinLat = Math.sin((lat * Math.PI) / 180);
	const clamped = Math.max(-0.9999, Math.min(0.9999, sinLat));
	const y = (0.5 - Math.log((1 + clamped) / (1 - clamped)) / (4 * Math.PI)) * WORLD_SIZE;
	return { x, y };
}

/**
 * Project geographic coordinates (lat, lng) to pixel offsets (x, y) within
 * a Google Static Maps image of the given `size` centered at `center` and
 * rendered at `zoom`. The returned point is in image pixels (top-left origin).
 *
 * This is the SAME Web Mercator formula Google Static Maps uses, so a callout
 * placed at the returned pixel lands exactly where the API would render it.
 *
 * Out-of-image coordinates are returned as-is (no clamping) — the caller
 * decides whether to skip rendering.
 */
export function projectToPixel(
	lat: number,
	lng: number,
	center: { lat: number; lng: number },
	zoom: number,
	size: { width: number; height: number }
): PixelPoint {
	const scale = Math.pow(2, zoom);
	const p = worldPixel(lat, lng);
	const c = worldPixel(center.lat, center.lng);
	return {
		x: (p.x - c.x) * scale + size.width / 2,
		y: (p.y - c.y) * scale + size.height / 2
	};
}

/**
 * Inverse of projectToPixel — convert a pixel (x, y) within a map image back
 * to geographic coordinates. Used by drop hit-testing (T5) to determine which
 * geo position a pointer drop targets on a map layer.
 */
export function unprojectFromPixel(
	x: number,
	y: number,
	center: { lat: number; lng: number },
	zoom: number,
	size: { width: number; height: number }
): { lat: number; lng: number } {
	const WORLD_SIZE = 256;
	const scale = Math.pow(2, zoom);
	const c = worldPixel(center.lat, center.lng);
	const worldX = (x - size.width / 2) / scale + c.x;
	const worldY = (y - size.height / 2) / scale + c.y;
	const lng = (worldX / WORLD_SIZE) * 360 - 180;
	const n = Math.PI - (2 * Math.PI * worldY) / WORLD_SIZE;
	const lat = (180 / Math.PI) * Math.atan(0.5 * (Math.exp(n) - Math.exp(-n)));
	return { lat, lng };
}
