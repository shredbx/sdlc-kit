// TDD — mapLayerStaticUrl (map-source.ts, Decision #0297).
//
// CASE ENUMERATION
//   TC-MS-01 success  map layer with valid config → keyless /api/map/static URL
//   TC-MS-02 edge     map layer without map_config → ''
//   TC-MS-03 edge     zero-area sizePx → ''
//   TC-MS-04 success  anchored marker layers baked as markers= params
//   TC-MS-05 success  anchored outline layers (closed, map coords) baked as path= params
//   TC-MS-06 regress  non-anchored layers (different map_surface_id) are NOT baked
//   TC-MS-07 regress  callout layers are NOT baked (text can't be in Static Maps API)
//   TC-MS-08 success  open outline layers (closed=false) are NOT baked
//   TC-MS-09 regress  anchor ring is auto-closed when first != last
//   TC-MS-10 success  multi-marker + multi-polygon all appear in URL
//   TC-MS-11 security injected builder always called with no key (builder is the guard)
//   TC-MS-12 success  R8: sizePx matches layer dimensions (aspect-locked)

import { describe, expect, it, vi } from 'vitest';
import { mapLayerStaticUrl, type MapStaticDescriptor } from './map-source.js';
import type { Layer } from './types/layer.js';

// ── Fixture helpers ───────────────────────────────────────────────────────────

const MAP_ID = 'map-1';

function makeMapLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: MAP_ID,
		type: 'map',
		name: 'Map',
		visible: true,
		locked: false,
		x: 0,
		y: 0,
		width: 640,
		height: 400,
		map_config: {
			center: { lat: 13.7563, lng: 100.5018 },
			zoom: 14,
			mapType: 'roadmap'
		},
		opacity: 1,
		...overrides
	};
}

function makeMarkerLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 'marker-1',
		type: 'marker',
		name: 'Marker',
		visible: true,
		locked: false,
		x: 0, y: 0,
		map_surface_id: MAP_ID,
		coord_system: 'map',
		geo_ref: { lat: 13.7563, lng: 100.5018, anchor_lats: [], anchor_lngs: [] },
		stroke: { color: '#e5392b', thickness: 2, dash_type: 'solid', glow_intensity: 0 },
		opacity: 1,
		...overrides
	};
}

function makeOutlineLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 'outline-1',
		type: 'outline',
		name: 'Region',
		visible: true,
		locked: false,
		x: 0, y: 0,
		map_surface_id: MAP_ID,
		coord_system: 'map',
		closed: true,
		anchors: [
			{ id: 'a1', x: 100.5, y: 13.75 },
			{ id: 'a2', x: 100.6, y: 13.75 },
			{ id: 'a3', x: 100.6, y: 13.85 }
		],
		stroke: { color: '#0d4f4f', thickness: 3, dash_type: 'solid', glow_intensity: 0 },
		fill: { color: '#c8a851', transparency: 0.8, pattern: 'solid' },
		opacity: 1,
		...overrides
	};
}

function makeCalloutLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 'callout-1',
		type: 'callout',
		name: 'Callout',
		visible: true,
		locked: false,
		x: 0, y: 0,
		map_surface_id: MAP_ID,
		coord_system: 'map',
		geo_ref: { lat: 13.76, lng: 100.51, anchor_lats: [], anchor_lngs: [] },
		content: 'Label',
		opacity: 1,
		...overrides
	};
}

// A stub builder that records calls and returns a predictable URL.
function stubBuilder(calls: MapStaticDescriptor[] = []) {
	return vi.fn((desc: MapStaticDescriptor): string => {
		calls.push(desc);
		if (!desc.center && !desc.markers?.length && !desc.polygons?.length) return '';
		return `/api/map/static?stub=1&size=${desc.size.width}x${desc.size.height}`;
	});
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('mapLayerStaticUrl — basic resolution', () => {
	it('TC-MS-01 map layer with valid config returns a URL via the builder', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		const result = mapLayerStaticUrl(makeMapLayer(), [], { width: 640, height: 400 }, builder);
		expect(result).toMatch(/^\/api\/map\/static/);
		expect(calls).toHaveLength(1);
		expect(calls[0].center).toEqual({ lat: 13.7563, lng: 100.5018 });
		expect(calls[0].zoom).toBe(14);
		expect(calls[0].mapType).toBe('roadmap');
	});

	it('TC-MS-02 map layer without map_config returns empty string', () => {
		const builder = stubBuilder();
		const result = mapLayerStaticUrl(makeMapLayer({ map_config: undefined }), [], { width: 640, height: 400 }, builder);
		expect(result).toBe('');
		expect(builder).not.toHaveBeenCalled();
	});

	it('TC-MS-03 zero-area sizePx returns empty string without calling builder', () => {
		const builder = stubBuilder();
		expect(mapLayerStaticUrl(makeMapLayer(), [], { width: 0, height: 400 }, builder)).toBe('');
		expect(mapLayerStaticUrl(makeMapLayer(), [], { width: 640, height: 0 }, builder)).toBe('');
		expect(builder).not.toHaveBeenCalled();
	});
});

describe('mapLayerStaticUrl — size passthrough + ≤640 clamp + retina scale', () => {
	it('TC-MS-12 a size within the cap passes through unchanged (no stretch)', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		mapLayerStaticUrl(makeMapLayer({ width: 480, height: 320 }), [], { width: 480, height: 320 }, builder);
		expect(calls[0].size).toEqual({ width: 480, height: 320 });
	});

	it('TC-MS-13 a size past the 640 cap clamps PROPORTIONALLY (aspect preserved, never blank)', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		// 1280×640 → max 1280 > 640 → k = 0.5 → 640×320 (aspect 2:1 preserved).
		mapLayerStaticUrl(makeMapLayer(), [], { width: 1280, height: 640 }, builder);
		expect(calls[0].size).toEqual({ width: 640, height: 320 });
		// A square over-cap clamps to 640×640.
		mapLayerStaticUrl(makeMapLayer(), [], { width: 900, height: 900 }, builder);
		expect(calls[1].size).toEqual({ width: 640, height: 640 });
	});

	it('TC-MS-14 the builder is always asked for retina (scale:2)', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		mapLayerStaticUrl(makeMapLayer(), [], { width: 480, height: 320 }, builder);
		expect(calls[0].scale).toBe(2);
	});
});

describe('mapLayerStaticUrl — anchored markers', () => {
	it('TC-MS-04 anchored marker layer appears in the markers array', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		const marker = makeMarkerLayer();
		mapLayerStaticUrl(makeMapLayer(), [marker], { width: 640, height: 400 }, builder);
		expect(calls[0].markers).toHaveLength(1);
		expect(calls[0].markers![0].lat).toBe(13.7563);
		expect(calls[0].markers![0].lng).toBe(100.5018);
		expect(calls[0].markers![0].color).toBe('#e5392b');
	});

	it('TC-MS-06 layer with a different map_surface_id is NOT included', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		const otherMarker = makeMarkerLayer({ map_surface_id: 'different-map' });
		mapLayerStaticUrl(makeMapLayer(), [otherMarker], { width: 640, height: 400 }, builder);
		expect(calls[0].markers).toBeUndefined();
	});
});

describe('mapLayerStaticUrl — anchored outlines', () => {
	it('TC-MS-05 closed outline in map coords appears as a polygon', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		mapLayerStaticUrl(makeMapLayer(), [makeOutlineLayer()], { width: 640, height: 400 }, builder);
		expect(calls[0].polygons).toHaveLength(1);
		// The ring should have [lng, lat] order: anchors store x=lng, y=lat.
		const ring = calls[0].polygons![0].ring;
		expect(ring[0]).toEqual([100.5, 13.75]); // anchor a1: x=lng, y=lat
		expect(calls[0].polygons![0].strokeColor).toBe('#0d4f4f');
		expect(calls[0].polygons![0].fillColor).toBe('#c8a851');
		expect(calls[0].polygons![0].weight).toBe(3);
	});

	it('TC-MS-08 open outline (closed=false) is NOT baked', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		mapLayerStaticUrl(makeMapLayer(), [makeOutlineLayer({ closed: false })], { width: 640, height: 400 }, builder);
		expect(calls[0].polygons).toBeUndefined();
	});

	it('TC-MS-09 ring is auto-closed when first != last position', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		// makeOutlineLayer anchors: a1=[100.5,13.75], a2=[100.6,13.75], a3=[100.6,13.85]
		// first != last → auto-close appended
		mapLayerStaticUrl(makeMapLayer(), [makeOutlineLayer()], { width: 640, height: 400 }, builder);
		const ring = calls[0].polygons![0].ring;
		const first = ring[0];
		const last = ring[ring.length - 1];
		expect(first).toEqual(last); // closed ring
		expect(ring.length).toBe(4); // 3 anchors + closing position
	});
});

describe('mapLayerStaticUrl — callout not baked', () => {
	it('TC-MS-07 callout layers are excluded from baked data', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		mapLayerStaticUrl(makeMapLayer(), [makeCalloutLayer()], { width: 640, height: 400 }, builder);
		// No markers, no polygons from a callout
		expect(calls[0].markers).toBeUndefined();
		expect(calls[0].polygons).toBeUndefined();
	});
});

describe('mapLayerStaticUrl — multi-annotation', () => {
	it('TC-MS-10 multiple markers + polygons all appear in the descriptor', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		const marker1 = makeMarkerLayer({ id: 'm1', geo_ref: { lat: 13.75, lng: 100.5, anchor_lats: [], anchor_lngs: [] } });
		const marker2 = makeMarkerLayer({ id: 'm2', geo_ref: { lat: 13.80, lng: 100.6, anchor_lats: [], anchor_lngs: [] } });
		const outline = makeOutlineLayer();
		mapLayerStaticUrl(makeMapLayer(), [marker1, marker2, outline], { width: 640, height: 400 }, builder);
		expect(calls[0].markers).toHaveLength(2);
		expect(calls[0].polygons).toHaveLength(1);
	});
});

describe('mapLayerStaticUrl — security invariant', () => {
	it('TC-MS-11 builder is the only URL-issuing surface — no key or googleapis in descriptor', () => {
		const calls: MapStaticDescriptor[] = [];
		const builder = stubBuilder(calls);
		mapLayerStaticUrl(makeMapLayer(), [makeMarkerLayer(), makeOutlineLayer()], { width: 640, height: 400 }, builder);
		// The descriptor itself never contains a key or host reference.
		const descriptorStr = JSON.stringify(calls[0]);
		expect(descriptorStr).not.toMatch(/key=/i);
		expect(descriptorStr).not.toContain('googleapis');
		expect(descriptorStr).not.toContain('google.com');
	});
});
