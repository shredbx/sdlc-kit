// MAP LAYER STATIC URL BUILDER — canvas-kit seam (Decision #0297).
//
// Builds the keyless /api/map/static proxy URL for a map layer by composing:
//   • The map_config (center / zoom / mapType)
//   • Anchored marker layers      → repeated `markers=` params (baked into image)
//   • Anchored outline layers     → repeated `path=` params   (baked into image)
//   • Anchored callout layers     → NOT baked (text labels cannot be baked via Static
//                                   Maps API); the render pass projects them onto the
//                                   canvas over the map image instead.
//
// This module is HEADLESS (no DOM, no Svelte). @sbx/ui-map is NOT a dependency of
// @sbx/canvas-kit (the kit is purposefully dependency-light). The URL builder is
// INJECTED as `MapStaticUrlBuilder` — the same modular-architecture-first pattern
// as the Rasterizer in export.ts. The UI layer (@sbx/canvas-ui) threads in
// buildStaticMapPath from @sbx/ui-map; tests pass a stub.
//
// R8 ENFORCEMENT: `sizePx` MUST match the map layer's aspect (width/height ratio).
// The caller (resolveMapSrc in export.ts) always passes the layer's own pixel
// dimensions, so the static image is never stretched.

import type { Layer } from './types/layer.js';

// Default marker color for anchored marker layers with no explicit color set.
const DEFAULT_MARKER_COLOR = '#333333';

// Default polygon stroke color for anchored outline layers with no stroke color set.
const DEFAULT_OUTLINE_COLOR = '#333333';

/** Minimal descriptor for a single map marker — mirrors StaticMapMarkerDescriptor
 *  in @sbx/ui-map without importing it (no cross-package dependency). */
export interface MapMarkerInput {
	lat: number;
	lng: number;
	color?: string;
}

/** Minimal descriptor for a single polygon path. */
export interface MapPolygonInput {
	ring: Array<[number, number]>; // [lng, lat] GeoJSON order
	strokeColor?: string;
	fillColor?: string;
	weight?: number;
}

/** The subset of the StaticMapDescriptor that mapLayerStaticUrl populates.
 *  The injected builder accepts this shape (assignable to StaticMapDescriptor). */
export interface MapStaticDescriptor {
	center?: { lat: number; lng: number };
	zoom?: number;
	size: { width: number; height: number };
	/** Static-map pixel density. 2 = retina (Google returns a 2× image drawn into
	 *  the logical box). The canvas always requests scale:2 for crisp tiles. */
	scale?: 1 | 2;
	mapType?: string;
	markers?: MapMarkerInput[];
	polygons?: MapPolygonInput[];
}

/** Google Static Maps caps the LOGICAL size at 640 per dimension (free tier; the
 *  scale:2 we request doubles the returned PIXELS, not this logical cap). A request
 *  past 640 returns an error image — the root cause of the "map blanks on resize"
 *  bug — so we clamp here. */
const STATIC_MAX_DIM = 640;

/** Clamp a pixel size to ≤ STATIC_MAX_DIM per dimension, scaling PROPORTIONALLY so
 *  the aspect ratio (hence the geography shown) is never distorted. A box larger
 *  than the cap shows the 640-equivalent area upscaled into the box — soft, but
 *  never blank (true >640 sharpness needs tiling — a later slice). */
function clampStaticSize(width: number, height: number): { width: number; height: number } {
	const max = Math.max(width, height);
	if (max <= STATIC_MAX_DIM) return { width, height };
	const k = STATIC_MAX_DIM / max;
	return {
		width: Math.max(1, Math.round(width * k)),
		height: Math.max(1, Math.round(height * k))
	};
}

/** Injected static-map URL builder — @sbx/canvas-ui wires in buildStaticMapPath
 *  from @sbx/ui-map; unit tests pass a stub. Returns '' for invalid/empty input.
 *  MUST NEVER emit a Google API key or reference googleapis in the returned string
 *  (security invariant enforced by the ui-map implementation). */
export type MapStaticUrlBuilder = (descriptor: MapStaticDescriptor) => string;

/**
 * Build a single keyless /api/map/static URL for a type:'map' layer, baking in:
 *   - The map viewport (center/zoom/mapType from `mapLayer.map_config`)
 *   - All anchored type:'marker' layers as repeated `markers=` params
 *   - All anchored type:'outline' layers (closed rings in map coords) as `path=` params
 *
 * `anchoredAnnotations` is the subset of page layers whose `map_surface_id` equals
 * the map layer's `id` — the caller (resolveMapSrc) computes this.
 *
 * `sizePx` MUST equal `{ width: mapLayer.width, height: mapLayer.height }` (R8
 * aspect-lock — the static image must never stretch).
 *
 * `buildUrl` is the injected URL builder (wired to buildStaticMapPath by the UI layer).
 *
 * Returns `''` when:
 *   - `mapLayer.map_config` is absent
 *   - `sizePx` is zero-area
 *   - The injected `buildUrl` returns '' (security / length / validity guards)
 */
export function mapLayerStaticUrl(
	mapLayer: Layer,
	anchoredAnnotations: Layer[],
	sizePx: { width: number; height: number },
	buildUrl: MapStaticUrlBuilder
): string {
	const cfg = mapLayer.map_config;
	if (!cfg) return '';

	const w = Math.round(sizePx.width);
	const h = Math.round(sizePx.height);
	if (w <= 0 || h <= 0) return '';

	// Defensive: only process annotations actually anchored to this map layer.
	// The caller (resolveFrame) pre-filters, but guard here for belt-and-suspenders.
	const anchored = anchoredAnnotations.filter((l) => l.map_surface_id === mapLayer.id);

	// Collect baked markers from anchored type:'marker' layers.
	const markerDescriptors: MapMarkerInput[] = anchored
		.filter((l) => l.type === 'marker' && l.geo_ref)
		.map((l) => ({
			lat: l.geo_ref!.lat,
			lng: l.geo_ref!.lng,
			// stroke.color carries the marker color in the canvas-kit style model; fall back to DEFAULT.
			color: l.stroke?.color ?? DEFAULT_MARKER_COLOR
		}));

	// Collect baked outlines from anchored type:'outline' layers.
	// Only outline layers in 'map' coord_system with a closed ring of anchors are baked.
	const polygonDescriptors: MapPolygonInput[] = anchored
		.filter(
			(l) =>
				l.type === 'outline' &&
				l.coord_system === 'map' &&
				l.closed === true &&
				Array.isArray(l.anchors) &&
				l.anchors.length >= 3
		)
		.map((l) => {
			// Anchors on map-coord outline layers store { x: lng, y: lat } (canvas-kit convention
			// for map-space anchors: x = longitude, y = latitude — matching land-canvas AnchorPoint
			// semantics when coord_system:'map').
			const anchors = l.anchors!;
			// Build a GeoJSON [lng, lat] ring (Static Maps convention, flipped by buildStaticMapPath).
			const ring: Array<[number, number]> = anchors.map((a) => [a.x, a.y]);
			// Close the ring if not already closed (first != last).
			const first = ring[0];
			const last = ring[ring.length - 1];
			if (first[0] !== last[0] || first[1] !== last[1]) {
				ring.push([first[0], first[1]]);
			}
			return {
				ring,
				strokeColor: l.stroke?.color ?? DEFAULT_OUTLINE_COLOR,
				fillColor: l.fill?.color,
				weight: l.stroke?.thickness
			};
		});

	return buildUrl({
		center: cfg.center,
		zoom: cfg.zoom,
		// Clamp to the Static Maps logical cap (never blank), request retina pixels.
		size: clampStaticSize(w, h),
		scale: 2,
		mapType: cfg.mapType,
		markers: markerDescriptors.length > 0 ? markerDescriptors : undefined,
		polygons: polygonDescriptors.length > 0 ? polygonDescriptors : undefined
	});
}
