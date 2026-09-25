// MAP-ANCHOR — pure, headless geometry for binding annotation layers (marker /
// outline / callout) to a placeable type:'map' surface (Decision #0297, slice 1B).
//
// An annotation inserted UNANCHORED (coord_system:'map' but no map_surface_id /
// geo_ref / anchors) never bakes into the static map. These functions set the
// surface link + geo so the bake (map-source.ts mapLayerStaticUrl) and the
// projection (export.ts resolveFrame) pick it up.
//
// HEADLESS: no DOM, no Svelte runtime. PURE: never mutates inputs — every result
// is a NEW layer object. Ids via crypto.randomUUID() (matches factory.ts / store.ts).
//
// COORDINATE CONVENTION (map-coord layers): an AnchorPoint stores { x: lng, y: lat }
// and geo_ref stores { lat, lng } — the same semantics map-source.ts bakes from.

import type { Layer } from './types/layer.js';
import { unprojectFromPixel } from './geometry.js';

/** The annotation layer types this slice anchors to a map surface. */
type AnnotationType = 'marker' | 'outline' | 'callout';

/** Fallback half-side (artboard px) for an outline ring when the layer has no box size. */
const OUTLINE_SEED_HALF = 40;

/** True for the annotation layer types that anchor to a map surface. */
function isAnnotation(type: Layer['type']): type is AnnotationType {
	return type === 'marker' || type === 'outline' || type === 'callout';
}

/**
 * The TOPMOST visible, non-system `type:'map'` layer whose box contains the
 * artboard point (x, y); `null` when no map sits under the point. `layers` is the
 * page's layers in z-order (last element = topmost), so we scan back-to-front.
 * Skips non-'map' layers, system layers, and hidden layers (visible === false).
 */
export function mapLayerAt(layers: Layer[], x: number, y: number): Layer | null {
	for (let i = layers.length - 1; i >= 0; i--) {
		const l = layers[i];
		if (l.type !== 'map' || l.system || l.visible === false) continue;
		const lx = l.x ?? 0;
		const ly = l.y ?? 0;
		const lw = l.width ?? 0;
		const lh = l.height ?? 0;
		if (x >= lx && x <= lx + lw && y >= ly && y <= ly + lh) return l;
	}
	return null;
}

/**
 * Anchor a marker / outline / callout layer to `map` at the ARTBOARD point (x, y)
 * (the same space as layer.x / layer.y). Returns a NEW layer (mutates nothing)
 * carrying `map_surface_id` + the geo data the bake / projection read. A
 * non-annotation layer, or a map with no `map_config`, is returned UNCHANGED.
 *
 *   - marker & callout → a single geo_ref at (x, y).
 *   - outline          → a closed ring of 4 anchors seeded as an on-SCREEN ±40px
 *                        square around (x, y), unprojected to geo so the polygon
 *                        bakes (mapLayerStaticUrl needs anchors.length >= 3).
 */
export function anchorAnnotationToMap(layer: Layer, x: number, y: number, map: Layer): Layer {
	if (!isAnnotation(layer.type)) return layer;
	const cfg = map.map_config;
	if (!cfg) return layer;

	const size = { width: map.width ?? 0, height: map.height ?? 0 };
	// The pointer is in artboard space; the projection works in the map image's
	// local pixel space, so subtract the map's origin first.
	const localX = x - (map.x ?? 0);
	const localY = y - (map.y ?? 0);

	if (layer.type === 'outline') {
		// The geo ring spans the layer's BOX (its drawn or default width/height) so a
		// click-DRAWN region matches its anchors exactly — falling back to a small square
		// when the box is unset. Corners are unprojected to geo (NW, NE, SE, SW).
		const halfW = (layer.width ?? OUTLINE_SEED_HALF * 2) / 2;
		const halfH = (layer.height ?? OUTLINE_SEED_HALF * 2) / 2;
		const corners: Array<{ x: number; y: number }> = [
			{ x: localX - halfW, y: localY - halfH }, // NW
			{ x: localX + halfW, y: localY - halfH }, // NE
			{ x: localX + halfW, y: localY + halfH }, // SE
			{ x: localX - halfW, y: localY + halfH } // SW
		];
		const anchors = corners.map((c) => {
			const geo = unprojectFromPixel(c.x, c.y, cfg.center, cfg.zoom, size);
			// Map-coord anchors store x = lng, y = lat.
			return { id: crypto.randomUUID(), x: geo.lng, y: geo.lat };
		});
		return { ...layer, map_surface_id: map.id, coord_system: 'map', closed: true, anchors };
	}

	// marker & callout — a single geo_ref at the drop point.
	const geo = unprojectFromPixel(localX, localY, cfg.center, cfg.zoom, size);
	return {
		...layer,
		map_surface_id: map.id,
		coord_system: 'map',
		geo_ref: { lat: geo.lat, lng: geo.lng, anchor_lats: [], anchor_lngs: [] }
	};
}

/** Where an annotation dropped at (x, y) anchors: which map surface, and at which
 *  artboard point. Pure — `layers` is the page's layers in z-order. */
export interface AnchorTarget {
	map: Layer;
	x: number;
	y: number;
}

/** The box-centre of a layer (artboard px). */
function boxCentre(l: Layer): { x: number; y: number } {
	return { x: (l.x ?? 0) + (l.width ?? 0) / 2, y: (l.y ?? 0) + (l.height ?? 0) / 2 };
}

/** True for a map layer that can actually be anchored to (has a viewport config). */
function isAnchorableMap(l: Layer): boolean {
	return l.type === 'map' && !l.system && l.visible !== false && !!l.map_config;
}

/**
 * Resolve which map an annotation dropped at artboard (x, y) should anchor to, and
 * the point to anchor it at:
 *   - the topmost ANCHORABLE map under (x, y)        → anchor right there, at (x, y);
 *   - else the NEAREST anchorable map (box-centre)   → anchor at THAT map's centre,
 *     so the annotation is always in-frame (R5 floor);
 *   - no anchorable map on the page                  → `null` (caller leaves it
 *     unanchored; the R4-gate prevents this in practice).
 *
 * "Anchorable" = a visible, non-system `type:'map'` layer WITH a `map_config` — so a
 * returned target is always safe to pass to `anchorAnnotationToMap` (never a silent
 * no-op). On an exact distance tie the TOPMOST map wins, matching `mapLayerAt`.
 */
export function resolveAnchorTarget(layers: Layer[], x: number, y: number): AnchorTarget | null {
	const onPoint = mapLayerAt(layers, x, y);
	if (onPoint && onPoint.map_config) return { map: onPoint, x, y };

	// No (anchorable) map under the point → fall back to the nearest one, anchored at
	// its centre. Scan in z-order with `<=` so a tie resolves to the later (topmost) map.
	const maps = layers.filter(isAnchorableMap);
	if (maps.length === 0) return null;
	let best = maps[0];
	let bestDist = Infinity;
	for (const m of maps) {
		const c = boxCentre(m);
		const d = (c.x - x) ** 2 + (c.y - y) ** 2;
		if (d <= bestDist) {
			best = m;
			bestDist = d;
		}
	}
	const c = boxCentre(best);
	return { map: best, x: c.x, y: c.y };
}
